package linepanel

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/pkg/errors"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
	"parcours/message"
)

// columnFile implements board.File
type columnFile struct {
	name       string
	width      int
	fieldIndex int // index into lp.fields / line.Values
}

func (f columnFile) Name() string { return f.name }
func (f columnFile) Width() int   { return f.width }

const (
	headerHeight = 2 // Header row + separator line
)

// LinePanel displays paginated log lines using Board
type LinePanel struct {
	board tea.Model

	// Data state
	lines         []nt.Line // Current page of lines
	offset        int       // Page offset
	total         int       // Total lines available
	scrollingDown bool      // Whether last navigation was downward

	// Column state
	columns []nt.Column          // Column configuration
	fields  []nt.Field           // Field metadata from store
	colMap  map[string]nt.Column // Cached map of field name to column config

	// Current cell info (derived from board position)
	files        []board.File // visible files (always columnFile, assert on use)
	currentField string
	currentValue string

	// Size
	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

func New(ctx context.Context, columns []nt.Column, fields []nt.Field, count int, lgr nt.Logger) LinePanel {
	lp := LinePanel{
		columns: columns,
		fields:  fields,
		total:   count,
		ctx:     ctx,
		logger:  lgr,
	}
	lp.buildColMap()

	// Initialize with minimal 1x1 board
	// Todo: find a better approach
	brd, _ := board.New(
		[]board.Rank{board.NewRank([]tea.Model{piece.NewLabel("")})},
		[]board.File{columnFile{}},
		0, 0,
	)
	lp.board = brd
	lp.files = []board.File{columnFile{}}
	return lp
}

func (lp LinePanel) Init() tea.Cmd {
	return lp.board.Init()
}

func (lp LinePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		// Forward size to board
		lp.board, _ = lp.board.Update(board.SizeMsg{Width: msg.Width, Height: msg.Height})
		// Request initial page of data (board will be built when PageMsg arrives)
		return lp, message.GetPageCmd(lp.offset, lp.PageSize())

	case PageMsg:
		lp.lines = msg.Lines
		lp.total = msg.Count
		// Try to replace board data (preserves cursor position)
		ranks := lp.buildRanks()
		var cmd tea.Cmd
		lp.board, cmd = lp.board.Update(board.ReplaceMsg{Ranks: ranks})
		// If replace failed (dimensions changed), rebuild board
		if cmd != nil {
			// Todo: explicitly signal rather than error cmd hax
			if _, isErr := cmd().(message.ErrorMsg); isErr {
				lp.board, lp.files = lp.buildBoard()
				cmd = nil
			}
		}
		return lp, cmd

	case ColumnsMsg:
		lp.columns = msg.Columns
		lp.fields = msg.Fields
		lp.buildColMap()
		// Request new page with new columns (board will rebuild when dimensions change)
		return lp, message.GetPageCmd(lp.offset, lp.PageSize())

	case ResetMsg:
		lp.offset = 0
		return lp, message.GetPageCmd(0, lp.PageSize())

	case board.SquareMsg:
		field, value, err := lp.cellAt(msg.Rank, msg.File)
		if err != nil {
			return lp, message.ErrorCmd(err)
		}
		lp.currentField = field
		lp.currentValue = value
		// Calculate absolute row and send SelectedMsg
		absoluteRow := lp.offset + msg.Rank
		lineId := lp.lines[msg.Rank].Id
		return lp, func() tea.Msg {
			return message.SelectedMsg{Row: absoluteRow, Id: lineId}
		}

	case board.NavMsg:
		// Board hit a boundary - scroll the dataset
		pageSize := lp.PageSize()
		lp.scrollingDown = false // default to upward/top positioning
		switch msg.Direction {
		case board.NavDown:
			// Scroll down one line
			if lp.offset+pageSize < lp.total {
				lp.offset++
				lp.scrollingDown = true
				lp.ensureFullPage(pageSize)
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
		case board.NavUp:
			// Scroll up one line
			if lp.offset > 0 {
				lp.offset--
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
		case board.NavPageDown:
			// Jump to next page
			if lp.offset+pageSize < lp.total {
				lp.offset += pageSize
				lp.scrollingDown = true
				lp.ensureFullPage(pageSize)
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
			// Already at end, move cursor to bottom directly
			var cmd tea.Cmd
			lp.board, cmd = lp.board.Update(board.MoveToMsg{MoveTo: board.Bottom})
			return lp, cmd
		case board.NavPageUp:
			// Jump to previous page
			if lp.offset > 0 {
				lp.offset -= pageSize
				if lp.offset < 0 {
					lp.offset = 0
				}
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
			// Already at top, move cursor to top directly
			var cmd tea.Cmd
			lp.board, cmd = lp.board.Update(board.MoveToMsg{MoveTo: board.Top})
			return lp, cmd
		case board.NavTop:
			// Jump to first page
			if lp.offset != 0 {
				lp.offset = 0
				return lp, message.GetPageCmd(0, pageSize)
			}
		case board.NavBottom:
			// Jump to last page
			newOffset := max(0, lp.total-pageSize)
			if lp.offset != newOffset {
				lp.offset = newOffset
				lp.scrollingDown = true
				return lp, message.GetPageCmd(newOffset, pageSize)
			}
		}
		return lp, nil

	case tea.KeyPressMsg:
		if msg.String() == "c" {
			return lp, lp.filterCmd()
		}
		// Pass other keys to board
		var cmd tea.Cmd
		lp.board, cmd = lp.board.Update(msg)
		return lp, cmd

	default:
		// Pass everything else to board
		var cmd tea.Cmd
		lp.board, cmd = lp.board.Update(msg)
		return lp, cmd
	}
}

// PageSize returns the number of rows that fit on panel
func (lp LinePanel) PageSize() int {
	if lp.height < headerHeight {
		return 0
	}
	return lp.height - headerHeight
}

// ensureFullPage adjusts offset to guarantee a full page request
func (lp *LinePanel) ensureFullPage(pageSize int) {
	if lp.offset+pageSize > lp.total {
		lp.offset = max(0, lp.total-pageSize)
	}
}

// buildColMap builds and caches the column map
func (lp *LinePanel) buildColMap() {
	lp.colMap = make(map[string]nt.Column)
	for _, col := range lp.columns {
		lp.colMap[col.Field] = col
	}
}

// filterCmd returns a command to open the filter dialog with the selected cell
func (lp LinePanel) filterCmd() tea.Cmd {
	if lp.currentField == "" {
		return nil
	}
	return func() tea.Msg {
		return message.OpenFilterMsg{
			Field: lp.currentField,
			Value: lp.currentValue,
		}
	}
}

func (lp LinePanel) View() tea.View {
	return lp.board.View()
}

// buildRanks converts current lines into board Ranks
func (lp LinePanel) buildRanks() []board.Rank {
	var ranks []board.Rank
	for _, line := range lp.lines {
		var pieces []tea.Model
		for i, val := range line.Values {
			if i >= len(lp.fields) {
				continue
			}
			field := lp.fields[i]
			col, exists := lp.colMap[field.Name]
			if !exists || col.Hidden || col.Demote {
				continue
			}
			formatter := makeFormatter(field.Type, col.Format)
			// Todo: add truncate with ellipsis to formatter yah
			pieces = append(pieces, piece.NewValue(val, formatter))
		}
		ranks = append(ranks, board.NewRank(pieces))
	}
	return ranks
}

// makeFormatter creates a formatter function based on field type and format string
// Todo: un-trainwreck
func makeFormatter(fieldType, format string) func(nt.Value) string {
	if format != "" && fieldType == "TIMESTAMP" {
		return func(val nt.Value) string {
			t, err := val.Time()
			if err == nil {
				return t.Format(format)
			}
			return val.String()
		}
	}
	return func(v nt.Value) string {
		return v.String()
	}
}

// buildBoard converts current lines and columns into a Board
// Todo: rethink Board genisis, like totally
func (lp LinePanel) buildBoard() (board.Board, []board.File) {
	var files []board.File
	for i, field := range lp.fields {
		col, exists := lp.colMap[field.Name]
		if !exists || col.Hidden || col.Demote {
			continue
		}
		files = append(files, columnFile{name: field.Name, width: col.Width, fieldIndex: i})
	}

	ranks := lp.buildRanks()

	// Position board based on last navigation direction
	startRank := 0
	if lp.scrollingDown {
		startRank = len(ranks) - 1
	}

	brd, _ := board.New(ranks, files, startRank, 0)

	// Apply current viewport size to new board
	if lp.width > 0 {
		sized, _ := brd.Update(board.SizeMsg{Width: lp.width, Height: lp.height})
		brd = sized.(board.Board) // Todo: unfuck
	}

	return brd, files
}

// cellAt returns the field name and value at the given board position
func (lp LinePanel) cellAt(rank, file int) (field, value string, err error) {
	if file < 0 || file >= len(lp.files) || rank < 0 || rank >= len(lp.lines) {
		err = errors.Errorf("cell out of range: file=%d, rank=%d", file, rank)
		return
	}
	f := lp.files[file].(columnFile)
	if f.fieldIndex < 0 || f.fieldIndex >= len(lp.lines[rank].Values) {
		err = errors.Errorf("fieldIndex out of range: %d", f.fieldIndex)
		return
	}
	field = f.name
	value = lp.lines[rank].Values[f.fieldIndex].String()
	return
}
