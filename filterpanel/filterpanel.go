package filterpanel

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
	"parcours/message"
)

// FilterPanel displays a modal dialog for editing filters using Board
type FilterPanel struct {
	board   tea.Model
	filters []nt.Filter

	width             int
	height            int
	selectedFilterIdx int

	// Snapshot for cancel support - restored on esc
	filtersSnapshot []nt.Filter

	ctx    context.Context
	logger nt.Logger
}

// Todo: put op stuffs in Filter plz
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

// opIndex maps FilterOp to index in opStrings
var opIndex = map[nt.FilterOp]int{
	nt.Eq:       0,
	nt.Ne:       1,
	nt.Contains: 2,
	nt.Match:    3,
	nt.Gt:       4,
	nt.Gte:      5,
	nt.Lt:       6,
	nt.Lte:      7,
}

var filterFiles = []board.File{
	{Header: "", Width: 3},
	{Header: "Field", Width: 15},
	{Header: "Op", Width: 10},
	{Header: "Value", Width: 30},
}

func New(ctx context.Context, lgr nt.Logger) FilterPanel {
	return FilterPanel{
		ctx:    ctx,
		logger: lgr,
		board:  board.Placeholder{},
	}
}

func (pnl FilterPanel) Init() tea.Cmd {
	return nil
}

func (pnl FilterPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case message.OpenFilterMsg:
		// Start from committed state
		pnl.filters = make([]nt.Filter, len(pnl.filtersSnapshot))
		copy(pnl.filters, pnl.filtersSnapshot)

		newFilter := nt.Filter{
			Op:      nt.Ne,
			Field:   msg.Field,
			Value:   msg.Value,
			Enabled: true,
		}

		pnl.filters, pnl.selectedFilterIdx = pnl.placeFilter(newFilter)

		pnl.board = pnl.buildBoard()
		return pnl, nil

	case tea.WindowSizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height
		pnl.board, _ = pnl.board.Update(msg)
		return pnl, nil

	case piece.CheckedMsg:
		if msg.Rank >= 0 && msg.Rank < len(pnl.filters) {
			pnl.filters[msg.Rank].Enabled = msg.Checked
			// Todo: dont ignore range issues, pretty please
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
	// Todo: what if we had more than one checkbox -- look at file
	// so yeah smooth down some, start with x,y over file,rank

	case board.PositionMsg:
		pnl.selectedFilterIdx = msg.Rank
		return pnl, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "p":
			// Commit working state to snapshot and apply
			pnl.filtersSnapshot = pnl.filters
			return pnl, pnl.applyCmd()
		case "delete":
			// Delete selected filter
			if len(pnl.filters) > 0 && pnl.selectedFilterIdx < len(pnl.filters) {
				pnl.filters = append(pnl.filters[:pnl.selectedFilterIdx], pnl.filters[pnl.selectedFilterIdx+1:]...)
				// Adjust selection if we deleted the last item
				if pnl.selectedFilterIdx >= len(pnl.filters) && pnl.selectedFilterIdx > 0 {
					pnl.selectedFilterIdx--
				}
				pnl.board = pnl.buildBoard()
			}
			return pnl, nil
		default:
			// Pass to board
			var cmd tea.Cmd
			pnl.board, cmd = pnl.board.Update(msg)
			return pnl, Wrap(cmd)
		}
	}

	return pnl, nil
}

func (pnl FilterPanel) View() tea.View {
	return pnl.board.View()
	// Todo: where to decorate/pad "dialog" and maybe center or sommat?
}

func (pnl FilterPanel) applyCmd() tea.Cmd {
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

func (pnl FilterPanel) buildBoard() tea.Model {
	if len(pnl.filters) == 0 {
		return board.NewPlaceholder("no filters")
	}

	var ranks []board.Rank
	for _, filter := range pnl.filters {
		rank := board.NewRank([]tea.Model{
			piece.NewCheckbox(filter.Enabled),
			piece.NewLabel(filter.Field),
			piece.NewOperator(opStrings, opIndex[filter.Op]),
			piece.NewTextInput(fmt.Sprintf("%v", filter.Value), 50),
		})
		ranks = append(ranks, rank)
	}

	// Todo: dont ignore error yah yahb
	brd, _ := board.New(ranks, filterFiles, pnl.selectedFilterIdx, 0, pnl.width)
	return brd
}

func (pnl FilterPanel) placeFilter(f nt.Filter) ([]nt.Filter, int) {
	for i, existing := range pnl.filters {
		if filtersMatch(existing, f) {
			return pnl.filters, i
		}
	}
	filters := append(pnl.filters, f)
	return filters, len(filters) - 1
}

func filtersMatch(a, b nt.Filter) bool {
	return a.Field == b.Field && a.Value == b.Value && a.Op == b.Op
}
