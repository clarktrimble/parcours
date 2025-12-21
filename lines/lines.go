package lines

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
)

// LinesPanel displays paginated log lines using Board
type LinesPanel struct {
	board tea.Model

	// Data state
	lines  []nt.Line // Current page of lines
	offset int       // Page offset
	total  int       // Total lines available

	// Column state
	columns []nt.Column
	fields  []nt.Field

	// Size
	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

func NewLinesPanel(ctx context.Context, columns []nt.Column, fields []nt.Field, count int, lgr nt.Logger) (LinesPanel, error) {
	lp := LinesPanel{
		columns: columns,
		fields:  fields,
		total:   count,
		ctx:     ctx,
		logger:  lgr,
	}

	// Start with empty board - will be populated when we get size and data
	brd, err := board.New(
		[]board.Rank{},
		[]board.File{},
		0,
		0,
	)
	if err != nil {
		return lp, err
	}

	lp.board = brd
	return lp, nil
}

func (lp LinesPanel) Init() tea.Cmd {
	return lp.board.Init()
}

func (lp LinesPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		// Rebuild board with new size
		if lp.width > 0 && lp.height > 0 {
			brd, err := lp.buildBoard()
			if err != nil {
				lp.logger.Error(lp.ctx, "failed to build board", err)
				return lp, nil
			}
			lp.board = brd
		}
		return lp, nil

	case PageMsg:
		lp.lines = msg.Lines
		lp.total = msg.Count
		// Rebuild board with new data
		brd, err := lp.buildBoard()
		if err != nil {
			lp.logger.Error(lp.ctx, "failed to build board", err)
			return lp, nil
		}
		lp.board = brd
		return lp, nil

	case ColumnsMsg:
		lp.columns = msg.Columns
		lp.fields = msg.Fields
		// Rebuild board with new columns
		brd, err := lp.buildBoard()
		if err != nil {
			lp.logger.Error(lp.ctx, "failed to build board", err)
			return lp, nil
		}
		lp.board = brd
		return lp, nil

	default:
		// Pass everything else to board
		var cmd tea.Cmd
		lp.board, cmd = lp.board.Update(msg)
		return lp, cmd
	}
}

func (lp LinesPanel) View() tea.View {
	return lp.board.View()
}

// buildBoard converts current lines into a Board
func (lp LinesPanel) buildBoard() (board.Board, error) {
	// No size yet, return empty board
	if lp.width == 0 || lp.height == 0 {
		return board.New([]board.Rank{}, []board.File{}, 0, 0)
	}

	// Build files (column headers) from columns
	var files []board.File
	for _, col := range lp.columns {
		if col.Hidden || col.Demote {
			continue
		}
		files = append(files, board.NewFile(piece.NewLabel(col.Field)))
	}

	if len(files) == 0 {
		return board.New([]board.Rank{}, []board.File{}, 0, 0)
	}

	// Build ranks (rows) from lines
	var ranks []board.Rank
	for _, line := range lp.lines {
		var pieces []board.Piece
		for _, col := range lp.columns {
			if col.Hidden || col.Demote {
				continue
			}
			// Find the field index for this column
			fieldIdx := -1
			for i, field := range lp.fields {
				if field.Name == col.Field {
					fieldIdx = i
					break
				}
			}
			if fieldIdx < 0 || fieldIdx >= len(line.Values) {
				pieces = append(pieces, piece.NewLabel(""))
				continue
			}
			// For now, just use labels - we can make these editable later
			pieces = append(pieces, piece.NewLabel(line.Values[fieldIdx].String()))
		}
		ranks = append(ranks, board.NewRank(pieces))
	}

	// If no data, create at least one empty rank to avoid errors
	if len(ranks) == 0 {
		emptyPieces := make([]board.Piece, len(files))
		for i := range emptyPieces {
			emptyPieces[i] = piece.NewLabel("")
		}
		ranks = append(ranks, board.NewRank(emptyPieces))
	}

	return board.New(ranks, files, 0, 0)
}
