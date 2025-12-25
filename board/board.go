package board

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/pkg/errors"

	"parcours/message"
	"parcours/style"
)

type File interface {
	Name() string
	Width() int
}

// MoveTo positions
type MoveTo int

const (
	Top MoveTo = iota
	Bottom
)

const gutter = 1 // space between columns

// MoveToMsg signals cursor should move to a position
type MoveToMsg struct {
	MoveTo MoveTo
}

// SizeMsg tells the board its display size
type SizeMsg struct {
	Width  int
	Height int
}

// ReplaceMsg signals the board should replace its ranks
type ReplaceMsg struct {
	Ranks []Rank
}

// Piece represents a board piece that can update and render itself.
type Piece interface {
	Update(tea.Msg) (Piece, tea.Cmd)
	Render() string
	Value() string // Returns the raw value (for filtering, etc.) Todo: nt.Value ??
}

// PieceMsg is the interface for messages from interactive pieces.
// Board injects position via SetPosition before returning the cmd.
type PieceMsg interface {
	IsPieceMsg()
	SetPosition(rank, file int)
}

type Square struct {
	piece    Piece
	position position // Todo: use/lose
}

type Rank struct {
	squares []Square
}

// NewRank creates a Rank from a slice of pieces.
func NewRank(pieces []Piece) Rank {
	squares := make([]Square, len(pieces))
	for i, piece := range pieces {
		squares[i] = Square{
			piece: piece,
		}
	}
	return Rank{squares: squares}
}

// Board represents a 2D grid of squares organized into ranks (rows).
// Board is designed for immutable use in bubbletea/Elm architecture:
// - Navigation methods (MoveUp/Down/Left/Right) return new Board with updated position
// - The underlying ranks slice is shared between Board instances (copy-on-write)
// - This is safe as long as square values are never mutated after board creation
// - If you need to modify square values, clone the ranks slice first
type Board struct {
	ranks    []Rank
	files    []File
	position position
	width    int // Number of files
	height   int // Number of ranks
	table    *table.Table

	// Viewport
	viewportWidth int // Display width in characters
	fileOffset    int // Index of leftmost visible file (for horizontal scrolling)
}

func New(ranks []Rank, files []File, rank, file int) (board Board, err error) {

	tbl := table.New()
	style.StyleTable(tbl)

	board = Board{
		ranks:    ranks,
		files:    files,
		width:    len(files),
		height:   len(ranks),
		position: position{file: file, rank: rank},
		table:    tbl,
	}

	board.setSquarePositions()
	err = board.validate()
	return
}

func (brd Board) Replace(ranks []Rank) (board Board, err error) {

	brd.ranks = ranks
	brd.setSquarePositions()

	err = brd.validate()
	if err != nil {
		return
	}

	board = brd
	return
}

func (brd Board) Init() tea.Cmd {
	return nil
}

func (brd Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case SizeMsg:
		brd.viewportWidth = msg.Width
		brd.fileOffset = brd.adjustFileOffset()
		return brd, nil
	case ReplaceMsg:
		newBrd, err := brd.Replace(msg.Ranks)
		if err != nil {
			return brd, func() tea.Msg {
				return message.ErrorMsg{Err: err}
			}
		}
		return newBrd, nil
	case MoveToMsg:
		switch msg.MoveTo {
		case Top:
			brd.position.rank = 0
		case Bottom:
			brd.position.rank = brd.height - 1
		}
		return brd, brd.positionCmd()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			return brd.moveUp()
		case "down", "j":
			return brd.moveDown()
		case "left", "h":
			return brd.moveLeft()
		case "right", "l":
			return brd.moveRight()
		case "g":
			return brd.moveTop()
		case "G":
			return brd.moveBottom()
		case "pgup", "ctrl+u":
			return brd.movePageUp()
		case "pgdown", "ctrl+d":
			return brd.movePageDown()
		}
	}

	// Pass message to the focused square
	square := brd.ranks[brd.position.rank].squares[brd.position.file]
	updatedPiece, cmd := square.piece.Update(msg)

	// Update the square with the new piece
	// Note: This mutates the shared ranks slice. Safe for bubbletea's single-model
	// event loop, but NOT safe if you keep multiple Board instances alive simultaneously
	// (e.g., for undo/redo or snapshots). If that's needed, clone ranks before mutation.
	brd.ranks[brd.position.rank].squares[brd.position.file].piece = updatedPiece

	// Wrap cmd to inject position into PieceMsg
	if cmd != nil {
		pos := square.position
		originalCmd := cmd
		cmd = func() tea.Msg {
			msg := originalCmd()
			if pm, ok := msg.(PieceMsg); ok {
				pm.SetPosition(pos.rank, pos.file)
			}
			return msg
		}
	}

	return brd, cmd
}

func (brd Board) View() tea.View {
	// Get visible file range
	visStart, visEnd := brd.visibleFiles()

	// Build headers from visible files only
	var headers []string
	for i := visStart; i < visEnd; i++ {
		file := brd.files[i]
		headers = append(headers, fmt.Sprintf("%-*s", file.Width()+gutter, file.Name()))
	}
	if len(headers) > 0 {
		brd.table.Headers(headers...)
	}

	// Build rows from ranks, including only visible files
	brd.table.ClearRows()
	for _, rank := range brd.ranks {
		var row []string
		for i := visStart; i < visEnd; i++ {
			square := rank.squares[i]
			row = append(row, truncate(square.piece.Render(), brd.files[i].Width()))
		}
		brd.table.Row(row...)
	}

	// Apply styling - adjust file index to be relative to visible range
	visualFile := brd.position.file - visStart
	brd.table.StyleFunc(style.CellStyler(brd.position.rank, visualFile))

	return tea.NewView(brd.table)
}

func (brd Board) moveUp() (Board, tea.Cmd) {
	if brd.position.rank > 0 {
		brd.position.rank--
		return brd, brd.positionCmd()
	}
	// Hit top edge
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavUp}
	}
}

func (brd Board) moveDown() (Board, tea.Cmd) {
	if brd.position.rank < brd.height-1 {
		brd.position.rank++
		return brd, brd.positionCmd()
	}
	// Hit bottom edge
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavDown}
	}
}

func (brd Board) moveLeft() (Board, tea.Cmd) {
	if brd.position.file > 0 {
		brd.position.file--
		brd.fileOffset = brd.adjustFileOffset()
		return brd, brd.positionCmd()
	}
	return brd, nil
}

func (brd Board) moveRight() (Board, tea.Cmd) {
	if brd.position.file < brd.width-1 {
		brd.position.file++
		brd.fileOffset = brd.adjustFileOffset()
		return brd, brd.positionCmd()
	}
	return brd, nil
}

func (brd Board) moveTop() (Board, tea.Cmd) {
	// Always move to top of board and signal want absolute top of dataset
	brd.position.rank = 0
	return brd, tea.Batch(
		brd.positionCmd(),
		func() tea.Msg { return message.NavMsg{Direction: message.NavTop} },
	)
}

func (brd Board) moveBottom() (Board, tea.Cmd) {
	// Always move to bottom of board and signal want absolute bottom of dataset
	brd.position.rank = brd.height - 1
	return brd, tea.Batch(
		brd.positionCmd(),
		func() tea.Msg { return message.NavMsg{Direction: message.NavBottom} },
	)
}

func (brd Board) movePageUp() (Board, tea.Cmd) {
	// Page up: request previous page, preserve cursor position
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavPageUp}
	}
}

func (brd Board) movePageDown() (Board, tea.Cmd) {
	// Page down: request next page, preserve cursor position
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavPageDown}
	}
}

// unexported

type position struct {
	rank int
	file int
}

func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + style.MutedStyle.Render("…")
}

// positionCmd returns a command that sends the current position and cell info
func (brd Board) positionCmd() tea.Cmd {
	pos := message.PositionMsg{
		Rank:  brd.position.rank,
		File:  brd.position.file,
		Field: brd.files[brd.position.file].Name(),
		Value: brd.ranks[brd.position.rank].squares[brd.position.file].piece.Value(),
	}
	return func() tea.Msg { return pos }
}

// visibleFiles returns the range of files [start, end) that fit in the viewport width
func (brd Board) visibleFiles() (start, end int) {
	return brd.visibleFilesFrom(brd.fileOffset)
}

// visibleFilesFrom returns the range of files [start, end) that fit starting from the given offset
func (brd Board) visibleFilesFrom(fileOffset int) (start, end int) {
	if fileOffset >= len(brd.files) {
		return 0, 0
	}
	if brd.viewportWidth == 0 {
		// No width constraint, show all files from offset
		return fileOffset, len(brd.files)
	}

	start = fileOffset
	usedWidth := 0

	for i := fileOffset; i < len(brd.files); i++ {
		fileWidth := brd.files[i].Width() + gutter
		if usedWidth+fileWidth > brd.viewportWidth {
			break
		}
		usedWidth += fileWidth
		end = i + 1
	}

	return start, end
}

// adjustFileOffset returns fileOffset adjusted to keep position.file visible
func (brd Board) adjustFileOffset() int {
	// If selected file is before visible range, scroll left
	if brd.position.file < brd.fileOffset {
		return brd.position.file
	}

	// If selected file is after visible range, scroll right minimally
	_, visEnd := brd.visibleFiles()
	if brd.position.file >= visEnd {
		// Increment offset until selected file is just visible (at right edge)
		for offset := brd.fileOffset + 1; offset <= brd.position.file; offset++ {
			_, end := brd.visibleFilesFrom(offset)
			if brd.position.file < end {
				return offset
			}
		}
		// Fallback: put selected file at left edge
		return brd.position.file
	}

	return brd.fileOffset
}

func (brd *Board) setSquarePositions() {
	for r := range brd.ranks {
		for f := range brd.ranks[r].squares {
			brd.ranks[r].squares[f].position = position{rank: r, file: f}
		}
	}
}

func (brd Board) validate() error {

	if len(brd.ranks) == 0 || len(brd.files) == 0 {
		return errors.Errorf("board requires non-zero ranks and files")
	}

	if len(brd.ranks) != brd.height {
		return errors.Errorf("ranks length %d does not match height %d", len(brd.ranks), brd.height)
	}
	if len(brd.files) != brd.width {
		return errors.Errorf("files length %d does not match width %d", len(brd.files), brd.width)
	}

	for i, r := range brd.ranks {
		if len(r.squares) != brd.width {
			return errors.Errorf("rank %d length does not equal width", i)
		}
	}

	if brd.position.rank < 0 || brd.position.rank >= brd.height {
		return errors.Errorf("rank %d out of bounds [0, %d)", brd.position.rank, brd.height)
	}
	if brd.position.file < 0 || brd.position.file >= brd.width {
		return errors.Errorf("file %d out of bounds [0, %d)", brd.position.file, brd.width)
	}

	return nil
}
