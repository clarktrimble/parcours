package linespanel

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
	"parcours/message"
)

const (
	headerHeight = 2 // Header row + separator line
)

// LinesPanel displays paginated log lines using Board
type LinesPanel struct {
	board tea.Model

	// Data state
	lines         []nt.Line // Current page of lines
	offset        int       // Page offset
	total         int       // Total lines available
	scrollingDown bool      // Whether last navigation was downward

	// Column state
	columns []nt.Column
	fields  []nt.Field

	// Size
	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

func New(ctx context.Context, columns []nt.Column, fields []nt.Field, count int, lgr nt.Logger) LinesPanel {
	lp := LinesPanel{
		columns: columns,
		fields:  fields,
		total:   count,
		ctx:     ctx,
		logger:  lgr,
	}

	// Initialize with minimal 1x1 board
	// Todo: find a better approach
	brd, _ := board.New(
		[]board.Rank{board.NewRank([]board.Piece{piece.NewLabel("")})},
		[]board.File{board.NewFile(piece.NewLabel(""))},
		0, 0,
	)
	lp.board = brd
	return lp
}

func (lp LinesPanel) Init() tea.Cmd {
	return lp.board.Init()
}

func (lp LinesPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		// Request initial page of data (board will be built when PageMsg arrives)
		return lp, message.GetPageCmd(lp.offset, lp.PageSize())

	case PageMsg:
		lp.lines = msg.Lines
		lp.total = msg.Count
		// Try to replace board data (preserves cursor position)
		ranks := lp.buildRanks()
		brd, err := lp.board.(board.Board).Replace(ranks)
		if err != nil {
			// Dimensions don't match, rebuild board instead
			lp.board = lp.buildBoard()
			return lp, nil
		}
		lp.board = brd
		return lp, nil

	case ColumnsMsg:
		lp.columns = msg.Columns
		lp.fields = msg.Fields
		// Rebuild board with new columns
		lp.board = lp.buildBoard()
		// Request new page with new columns
		return lp, message.GetPageCmd(lp.offset, lp.PageSize())

	case ResetMsg:
		lp.offset = 0
		return lp, message.GetPageCmd(0, lp.PageSize())

	case message.NavMsg:
		// Board hit a boundary - scroll the dataset
		pageSize := lp.PageSize()
		lp.scrollingDown = false // default to upward/top positioning
		switch msg.Direction {
		case message.NavDown:
			// Scroll down one line
			if lp.offset+pageSize < lp.total {
				lp.offset++
				lp.scrollingDown = true
				// Ensure we request a full page
				if lp.offset+pageSize > lp.total {
					lp.offset = max(0, lp.total-pageSize)
				}
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
		case message.NavUp:
			// Scroll up one line
			if lp.offset > 0 {
				lp.offset--
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
		case message.NavPageDown:
			// Jump to next page
			if lp.offset+pageSize < lp.total {
				lp.offset += pageSize
				lp.scrollingDown = true
				// Ensure we always request a full page
				if lp.offset+pageSize > lp.total {
					lp.offset = max(0, lp.total-pageSize)
				}
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
			// Already at end, move cursor to bottom
			return lp, func() tea.Msg {
				return board.MoveToMsg{MoveTo: board.Bottom}
			}
		case message.NavPageUp:
			// Jump to previous page
			if lp.offset > 0 {
				lp.offset -= pageSize
				if lp.offset < 0 {
					lp.offset = 0
				}
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
		case message.NavTop:
			// Jump to first page
			if lp.offset != 0 {
				lp.offset = 0
				return lp, message.GetPageCmd(0, pageSize)
			}
		case message.NavBottom:
			// Jump to last page
			newOffset := ((lp.total - 1) / pageSize) * pageSize
			if lp.offset != newOffset {
				lp.offset = newOffset
				lp.scrollingDown = true
				return lp, message.GetPageCmd(newOffset, pageSize)
			}
		}
		return lp, nil

	default:
		// Pass everything else to board
		var cmd tea.Cmd
		lp.board, cmd = lp.board.Update(msg)
		return lp, cmd
	}
}

// PageSize returns the number of rows that fit on panel
func (lp LinesPanel) PageSize() int {
	if lp.height < headerHeight {
		return 0
	}
	return lp.height - headerHeight
}

func (lp LinesPanel) View() tea.View {
	return lp.board.View()
}

// buildRanks converts current lines into board Ranks
func (lp LinesPanel) buildRanks() []board.Rank {
	colMap := make(map[string]nt.Column)
	for _, col := range lp.columns {
		colMap[col.Field] = col
	}

	var ranks []board.Rank
	for _, line := range lp.lines {
		var pieces []board.Piece
		for i, val := range line.Values {
			if i >= len(lp.fields) {
				continue
			}
			col, exists := colMap[lp.fields[i].Name]
			if exists && (col.Hidden || col.Demote) {
				continue
			}
			pieces = append(pieces, piece.NewLabel(val.String()))
		}
		ranks = append(ranks, board.NewRank(pieces))
	}
	return ranks
}

// buildBoard converts current lines and columns into a Board
func (lp LinesPanel) buildBoard() board.Board {
	var files []board.File
	colMap := make(map[string]nt.Column)
	for _, col := range lp.columns {
		colMap[col.Field] = col
	}

	for _, field := range lp.fields {
		col, exists := colMap[field.Name]
		if exists && (col.Hidden || col.Demote) {
			continue
		}
		files = append(files, board.NewFile(piece.NewLabel(field.Name)))
	}

	ranks := lp.buildRanks()

	// Position board based on last navigation direction
	startRank := 0
	if lp.scrollingDown {
		startRank = len(ranks) - 1
	}

	brd, _ := board.New(ranks, files, startRank, 0)
	return brd
}
