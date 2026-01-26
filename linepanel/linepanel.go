package linepanel

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/pkg/errors"

	"parcours/board"
	nt "parcours/entity"
	"parcours/message"
)

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
	files       board.Files // visible files (SrcIdx maps to line.Values)
	currentRank int
	currentFile int

	// Size
	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

func New(ctx context.Context, lgr nt.Logger, size tea.WindowSizeMsg) LinePanel {
	return LinePanel{
		ctx:    ctx,
		logger: lgr,
		width:  size.Width,
		height: size.Height,
		board:  board.NewPlaceholder("Loading..."),
	}
}

func (lp LinePanel) Init() tea.Cmd {
	return Wrap(lp.board.Init())
}

func (lp LinePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		// Forward size to board
		lp.board, _ = lp.board.Update(msg)
		// Request initial page of data (board will be built when PageMsg arrives)
		return lp, message.GetPageCmd(lp.offset, lp.PageSize())

	case PageMsg:
		lp.lines = msg.Lines
		lp.total = msg.Count
		lp, cmd := lp.applyPage()
		return lp, tea.Batch(cmd, func() tea.Msg {
			return message.CountMsg{Count: msg.Count}
		})

	case ColumnsMsg:
		lp.columns = msg.Columns
		lp.fields = msg.Fields
		lp.buildColMap()
		// Request new page with new columns (board will rebuild when dimensions change)
		return lp, message.GetPageCmd(lp.offset, lp.PageSize())

	case ResetMsg:
		lp.offset = 0
		return lp, message.GetPageCmd(0, lp.PageSize())

	case board.PositionMsg:
		lp.currentRank = msg.Rank
		lp.currentFile = msg.File
		return lp, lp.selectedCmd(msg.Rank)

	case board.NavMsg:
		// Board hit a boundary - scroll the dataset
		pageSize := lp.PageSize()
		lp.scrollingDown = false // default to upward/top positioning
		switch msg.Direction {
		case board.NavDown:
			// Scroll down by msg.Ranks
			if lp.offset+pageSize < lp.total {
				lp.offset += msg.Count
				lp.scrollingDown = true
				lp.ensureFullPage(pageSize)
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
			// Can't scroll further, move cursor to bottom
			var cmd tea.Cmd
			lp.board, cmd = lp.board.Update(board.MoveToMsg{MoveTo: board.Bottom})
			return lp, cmd
		case board.NavUp:
			// Scroll up by msg.Ranks
			if lp.offset > 0 {
				lp.offset -= msg.Count
				if lp.offset < 0 {
					lp.offset = 0
				}
				return lp, message.GetPageCmd(lp.offset, pageSize)
			}
			// Can't scroll further, move cursor to top
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
		switch msg.String() {
		case "esc":
			return lp, func() tea.Msg { return message.CloseMsg{} }
		case "c":
			return lp, lp.filterCmd()
		case "f":
			return lp, func() tea.Msg { return message.OpenFilterMsg{} }
		case "r":
			return lp, func() tea.Msg { return message.ReloadColumnsMsg{} }
		case "o":
			return lp, func() tea.Msg { return message.OpenIntakeMsg{} }
		case "enter":
			return lp, lp.openDetailCmd()
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

// selectedCmd returns a command to send the selected row info
func (lp LinePanel) selectedCmd(rank int) tea.Cmd {
	if rank < 0 || rank >= len(lp.lines) {
		err := errors.Errorf("rank out of range: %d", rank)
		return func() tea.Msg { return err }
	}
	absoluteRow := lp.offset + rank
	lineId := lp.lines[rank].Id
	return func() tea.Msg {
		return message.SelectedMsg{Row: absoluteRow, Id: lineId}
	}
}

// filterCmd returns a command to open the filter dialog with the selected cell
func (lp LinePanel) filterCmd() tea.Cmd {
	field, value, err := lp.cellAt(lp.currentRank, lp.currentFile)
	if err != nil {
		return func() tea.Msg { return err }
	}
	return func() tea.Msg {
		return message.OpenFilterMsg{
			Field: field,
			Value: value,
		}
	}
}

// openDetailCmd returns a command to open detail view for the selected line
func (lp LinePanel) openDetailCmd() tea.Cmd {
	if lp.currentRank < 0 || lp.currentRank >= len(lp.lines) {
		return nil
	}
	id := lp.lines[lp.currentRank].Id
	cols := lp.columns
	return func() tea.Msg {
		return message.OpenDetailMsg{Id: id, Columns: cols}
	}
}

func (lp LinePanel) View() tea.View {
	return lp.board.View()
}

// cellAt returns the field name and value at the given board position
func (lp LinePanel) cellAt(rank, file int) (field string, value nt.Value, err error) {
	if file < 0 || file >= len(lp.files) || rank < 0 || rank >= len(lp.lines) {
		err = errors.Errorf("cell out of range: file=%d, rank=%d", file, rank)
		return
	}
	f := lp.files[file]
	if f.SrcIdx >= len(lp.lines[rank].Values) {
		err = errors.Errorf("SrcIdx out of range: %d", f.SrcIdx)
		return
	}
	field = f.Header
	value = lp.lines[rank].Values[f.SrcIdx]
	return
}
