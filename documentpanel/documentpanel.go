package documentpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	tea "charm.land/bubbletea/v2"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
	"parcours/message"
)

// Top fields shown first, in order
var topFields = []string{"ts", "timestamp", "level", "msg", "message"}

// DetailPanelToo displays key/value pairs using Board
type DetailPanelToo struct {
	board   tea.Model
	columns []nt.Column
	line    map[string]any

	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

func New(ctx context.Context, columns []nt.Column, lgr nt.Logger, size tea.WindowSizeMsg) DetailPanelToo {
	return DetailPanelToo{
		ctx:     ctx,
		logger:  lgr,
		columns: columns,
		width:   size.Width,
		height:  size.Height,
		board:   board.NewPlaceholder("Loading..."),
	}
}

func (pnl DetailPanelToo) Init() tea.Cmd {
	return nil
}

func (pnl DetailPanelToo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height
		pnl.board, _ = pnl.board.Update(msg)
		return pnl, nil

	case LineMsg:
		pnl.line = msg.Line
		return pnl.rebuildBoard()

	case ColumnsMsg:
		pnl.columns = msg.Columns
		if pnl.line != nil {
			return pnl.rebuildBoard()
		}
		return pnl, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return pnl, func() tea.Msg { return message.CloseMsg{} }
		case "v":
			return pnl, func() tea.Msg {
				return message.OpenJsonDetailMsg{
					Line:    pnl.line,
					Columns: pnl.columns,
				}
			}
		default:
			var cmd tea.Cmd
			pnl.board, cmd = pnl.board.Update(msg)
			return pnl, cmd
		}
	}

	// Pass everything else to board
	var cmd tea.Cmd
	pnl.board, cmd = pnl.board.Update(msg)
	return pnl, cmd
}

func (pnl DetailPanelToo) View() tea.View {
	return pnl.board.View()
}

// unexported

func (pnl DetailPanelToo) rebuildBoard() (DetailPanelToo, tea.Cmd) {
	var err error
	pnl.board, err = pnl.buildBoard()
	if err != nil {
		return pnl, func() tea.Msg { return err }
	}
	return pnl, nil
}

func (pnl DetailPanelToo) buildBoard() (tea.Model, error) {
	if pnl.line == nil {
		return board.NewPlaceholder("No data"), nil
	}

	keys := pnl.orderedKeys()
	if len(keys) == 0 {
		return board.NewPlaceholder("Empty record"), nil
	}

	// Compute key column width
	keyWidth := pnl.maxKeyWidth(keys)

	// Value column gets remaining width (minus key, gutter, some padding)
	valueWidth := pnl.width - keyWidth - 3
	if valueWidth < 20 {
		valueWidth = 20
	}

	files := []board.File{
		{Header: "Key", Width: keyWidth},
		{Header: "Value", Width: valueWidth},
	}

	var ranks []board.Rank
	for _, key := range keys {
		val := pnl.formatValue(pnl.line[key])
		rank := board.NewRank([]tea.Model{
			piece.NewLabel(key),
			piece.NewLabel(val),
		})
		ranks = append(ranks, rank)
	}

	return board.New(ranks, files, 0, 0, pnl.width)
}

// orderedKeys returns keys with top fields first, then remaining alphabetically
func (pnl DetailPanelToo) orderedKeys() []string {
	// Collect all keys
	allKeys := make([]string, 0, len(pnl.line))
	for k := range pnl.line {
		allKeys = append(allKeys, k)
	}

	var ordered []string

	// Add top fields that exist
	for _, top := range topFields {
		if _, ok := pnl.line[top]; ok {
			ordered = append(ordered, top)
		}
	}

	// Sort remaining keys alphabetically
	var rest []string
	for _, k := range allKeys {
		if !slices.Contains(topFields, k) {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)

	return append(ordered, rest...)
}

// maxKeyWidth returns the max width needed for keys
func (pnl DetailPanelToo) maxKeyWidth(keys []string) int {
	maxW := 0
	for _, k := range keys {
		if len(k) > maxW {
			maxW = len(k)
		}
	}
	// Add some padding
	return maxW + 1
}

// formatValue converts a value to display string
func (pnl DetailPanelToo) formatValue(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case float64, int, int64, bool:
		return fmt.Sprintf("%v", v)
	default:
		// Objects/arrays: compact JSON
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	}
}
