package filter

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
	"parcours/message"
	"parcours/style"
)

// FilterPanelToo displays a modal dialog for editing filters using Board
type FilterPanelToo struct {
	board   board.Board
	filters []nt.Filter

	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

// opStrings for Operator piece
var opStrings = []string{
	"==",
	"!=",
	"contains",
	"matches",
	">",
	">=",
	"<",
	"<=",
}

// opFromString maps operator string back to FilterOp
var opFromString = map[string]nt.FilterOp{
	"==":       nt.Eq,
	"!=":       nt.Ne,
	"contains": nt.Contains,
	"matches":  nt.Match,
	">":        nt.Gt,
	">=":       nt.Gte,
	"<":        nt.Lt,
	"<=":       nt.Lte,
}

func NewFilterPanelToo(ctx context.Context, lgr nt.Logger) FilterPanelToo {
	pnl := FilterPanelToo{
		ctx:    ctx,
		logger: lgr,
	}
	pnl.board = pnl.buildBoard()
	return pnl
}

func (pnl FilterPanelToo) Init() tea.Cmd {
	return nil
}

func (pnl FilterPanelToo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case message.OpenFilterMsg:
		// Add new filter from cell selection
		newFilter := nt.Filter{
			Op:      nt.Eq,
			Field:   msg.Field,
			Value:   msg.Value,
			Enabled: true,
		}
		pnl.filters = append(pnl.filters, newFilter)
		pnl.board = pnl.buildBoard()
		return pnl, nil

	case SizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height
		sized, _ := pnl.board.Update(board.SizeMsg{Width: msg.Width, Height: msg.Height})
		pnl.board = sized.(board.Board)
		return pnl, nil

	case piece.CheckedMsg:
		if msg.Rank >= 0 && msg.Rank < len(pnl.filters) {
			pnl.filters[msg.Rank].Enabled = msg.Checked
		}
		return pnl, nil

	case piece.OperatorChangedMsg:
		if msg.Rank >= 0 && msg.Rank < len(pnl.filters) {
			if op, ok := opFromString[msg.Selected]; ok {
				pnl.filters[msg.Rank].Op = op
			}
		}
		return pnl, nil

	case piece.ValueChangedMsg:
		if msg.Rank >= 0 && msg.Rank < len(pnl.filters) {
			pnl.filters[msg.Rank].Value = msg.Value
		}
		return pnl, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "p":
			// Apply all enabled filters
			return pnl, pnl.applyCmd()
		default:
			// Pass to board
			var cmd tea.Cmd
			updated, cmd := pnl.board.Update(msg)
			pnl.board = updated.(board.Board)
			return pnl, cmd
		}
	}

	return pnl, nil
}

func (pnl FilterPanelToo) View() tea.View {
	// Help text
	helpText := "t: toggle  ←→: change op  p: apply  Esc: cancel"

	// Create a bordered box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(60)

	// Get board content as string via fmt
	boardContent := fmt.Sprintf("%s", pnl.board.View().Content)

	// Combine board view with help
	dialogContent := fmt.Sprintf("Filters:\n%s\n\n%s",
		boardContent,
		style.MutedStyle.Render(helpText))

	dialog := dialogStyle.Render(dialogContent)

	// Center the dialog
	if pnl.width > 0 && pnl.height > 0 {
		dialogHeight := strings.Count(dialog, "\n") + 1
		dialogWidth := 64

		vPad := (pnl.height - dialogHeight) / 2
		hPad := (pnl.width - dialogWidth) / 2

		if vPad < 0 {
			vPad = 0
		}
		if hPad < 0 {
			hPad = 0
		}

		dialogLayer := lipgloss.NewLayer("filter", dialog).
			X(hPad).
			Y(vPad)

		return tea.NewView(dialogLayer)
	}

	dialogLayer := lipgloss.NewLayer("filter", dialog)
	return tea.NewView(dialogLayer)
}

func (pnl FilterPanelToo) applyCmd() tea.Cmd {
	var enabledFilters []nt.Filter
	for _, f := range pnl.filters {
		if f.Enabled {
			enabledFilters = append(enabledFilters, f)
		}
	}

	var filterToApply nt.Filter
	if len(enabledFilters) == 0 {
		filterToApply = nt.Filter{}
	} else if len(enabledFilters) == 1 {
		filterToApply = enabledFilters[0]
	} else {
		filterToApply = nt.Filter{
			Op:       nt.And,
			Children: enabledFilters,
		}
	}

	return func() tea.Msg {
		return message.SetFilterMsg{Filter: filterToApply}
	}
}

func (pnl FilterPanelToo) buildBoard() board.Board {
	if len(pnl.filters) == 0 {
		// Empty board with placeholder
		brd, _ := board.New(
			[]board.Rank{board.NewRank([]board.Piece{piece.NewLabel("(no filters)")})},
			[]board.File{filterFile{name: "", width: 20}},
			0, 0,
		)
		return brd
	}

	var ranks []board.Rank
	for _, f := range pnl.filters {
		opIndex := 0
		for i, op := range opStrings {
			if opFromString[op] == f.Op {
				opIndex = i
				break
			}
		}

		rank := board.NewRank([]board.Piece{
			piece.NewCheckbox(f.Enabled),
			piece.NewLabel(f.Field),
			piece.NewOperator(opStrings, opIndex),
			piece.NewTextInput(fmt.Sprintf("%v", f.Value), 50),
		})
		ranks = append(ranks, rank)
	}

	files := []board.File{
		filterFile{name: "", width: 3},      // checkbox
		filterFile{name: "Field", width: 15}, // field name
		filterFile{name: "Op", width: 10},    // operator
		filterFile{name: "Value", width: 30}, // value
	}

	brd, _ := board.New(ranks, files, 0, 0)
	return brd
}

// filterFile implements board.File
type filterFile struct {
	name  string
	width int
}

func (f filterFile) Name() string { return f.name }
func (f filterFile) Width() int   { return f.width }
