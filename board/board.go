package board

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/pkg/errors"

	"parcours/message"
	"parcours/style"
)

type Field interface {
	String() string
}

// Piece represents a board piece that can update and render itself.
type Piece interface {
	Update(tea.Msg) (Piece, tea.Cmd)
	Render() string
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

type File struct {
	field Field
}

// NewFile creates a File with the given field.
func NewFile(field Field) File {
	return File{field: field}
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
}

func New(ranks []Rank, files []File, rank, file int) (board Board, err error) {

	if len(ranks) == 0 || len(files) == 0 {
		err = errors.Errorf("board requires non-zero ranks and files")
		return
	}

	width := len(files)
	height := len(ranks)

	// Validate all ranks have same width
	for i, r := range ranks {
		if len(r.squares) != width {
			err = errors.Errorf("rank %d length does not equal width", i)
			return
		}
	}

	// Validate position
	if rank < 0 || rank >= height {
		err = errors.Errorf("rank %d out of bounds [0, %d)", rank, height)
		return
	}
	if file < 0 || file >= width {
		err = errors.Errorf("file %d out of bounds [0, %d)", file, width)
		return
	}

	tbl := table.New()
	style.StyleTable(tbl)

	board = Board{
		ranks:    ranks,
		files:    files,
		width:    width,
		height:   height,
		position: position{file: file, rank: rank},
		table:    tbl,
	}

	return
}

func (brd Board) Init() tea.Cmd {
	return nil
}

func (brd Board) Rank() []Square {
	return slices.Clone(brd.ranks[brd.position.rank].squares)
}

func (brd Board) Square() Square {
	return brd.ranks[brd.position.rank].squares[brd.position.file]
}

func (brd Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// Handle navigation keys
	switch msg := msg.(type) {
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

	return brd, cmd
}

func (brd Board) View() tea.View {
	// Build headers from files
	var headers []string
	for _, file := range brd.files {
		if file.field != nil {
			headers = append(headers, file.field.String())
		} else {
			headers = append(headers, "")
		}
	}
	if len(headers) > 0 {
		brd.table.Headers(headers...)
	}

	// Build rows from ranks
	brd.table.ClearRows()
	for _, rank := range brd.ranks {
		var row []string
		for _, square := range rank.squares {
			row = append(row, square.piece.Render())
		}
		brd.table.Row(row...)
	}

	// Apply styling to highlight focused square, rank, and file
	brd.table.StyleFunc(style.CellStyler(brd.position.rank, brd.position.file))

	return tea.NewView(brd.table)
}

func (brd Board) moveUp() (Board, tea.Cmd) {
	if brd.position.rank > 0 {
		brd.position.rank--
		return brd, nil
	}
	// Hit top edge
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavUp}
	}
}

func (brd Board) moveDown() (Board, tea.Cmd) {
	if brd.position.rank < brd.height-1 {
		brd.position.rank++
		return brd, nil
	}
	// Hit bottom edge
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavDown}
	}
}

func (brd Board) moveLeft() (Board, tea.Cmd) {
	if brd.position.file > 0 {
		brd.position.file--
	}
	return brd, nil
}

func (brd Board) moveRight() (Board, tea.Cmd) {
	if brd.position.file < brd.width-1 {
		brd.position.file++
	}
	return brd, nil
}

func (brd Board) moveTop() (Board, tea.Cmd) {
	// Always move to top of board and signal want absolute top of dataset
	brd.position.rank = 0
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavTop}
	}
}

func (brd Board) moveBottom() (Board, tea.Cmd) {
	// Always move to bottom of board and signal want absolute bottom of dataset
	brd.position.rank = brd.height - 1
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavBottom}
	}
}

func (brd Board) movePageUp() (Board, tea.Cmd) {
	// Page up always means previous page (board height = page size)
	brd.position.rank = 0
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavPageUp}
	}
}

func (brd Board) movePageDown() (Board, tea.Cmd) {
	// Page down always means next page (board height = page size)
	brd.position.rank = brd.height - 1
	return brd, func() tea.Msg {
		return message.NavMsg{Direction: message.NavPageDown}
	}
}

// unexported

type position struct {
	rank int
	file int
}
