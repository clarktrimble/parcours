package board

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/pkg/errors"
)

type Empty struct{}

func (empty Empty) String() string {
	return "empty"
}

func (empty Empty) Init() tea.Cmd {
	return nil
}

func (empty Empty) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return empty, nil
}

func (empty Empty) View() tea.View {
	return tea.NewView("empty")
}

/*
type Value interface {
	//String() string
	Init() tea.Cmd
	Update(tea.Msg) (tea.Model, tea.Cmd)
	View() tea.View
}
*/

type Field interface {
	String() string
}

// Piece represents a board piece that can update and render itself.
type Piece interface {
	Update(tea.Msg) (tea.Model, tea.Cmd)
	Render() string
}

type Square struct {
	model    tea.Model
	position position // Todo: use/lose
}

type Rank struct {
	squares []Square
}

// NewRank creates a Rank from a slice of models.
func NewRank(models []tea.Model) Rank {
	squares := make([]Square, len(models))
	for i, model := range models {
		squares[i] = Square{
			model: model,
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
}

func NewBoard(width, height int) Board {
	ranks := make([]Rank, height)
	for r := range ranks {
		squares := make([]Square, width)
		for f := range squares {
			squares[f] = Square{
				model:    Empty{},
				position: position{file: f, rank: r},
			}
		}
		ranks[r] = Rank{squares: squares}
	}

	return Board{
		ranks:    ranks,
		files:    make([]File, width),
		width:    width,
		height:   height,
		position: position{file: 0, rank: 0},
	}
}

func New(ranks []Rank, files []File) (board Board, err error) {

	if len(ranks) == 0 || len(files) == 0 {
		err = errors.Errorf("board requires non-zero ranks and files")
		return
	}

	width := len(files)
	for i, rank := range ranks {
		if len(rank.squares) != width {
			err = errors.Errorf("rank %d length does not equal width", i)
			return
		}
	}

	//ranks[0].squares[0].position = position{file: 0, rank: 0}

	board = Board{
		ranks:    ranks,
		files:    files,
		width:    width,
		height:   len(ranks),
		position: position{file: 0, rank: 0},
	}

	return
}

func (brd Board) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, rank := range brd.ranks {
		for _, square := range rank.squares {
			if cmd := square.model.Init(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return tea.Batch(cmds...)
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
			return brd.MoveUp(), nil
		case "down", "j":
			return brd.MoveDown(), nil
		case "left", "h":
			return brd.MoveLeft(), nil
		case "right", "l":
			return brd.MoveRight(), nil
		}
	}

	// Pass message to the focused square
	square := brd.ranks[brd.position.rank].squares[brd.position.file]
	updatedModel, cmd := square.model.Update(msg)

	// Update the square with the new model
	// NOTE: This mutates the shared ranks slice. Safe for bubbletea's single-model
	// event loop, but NOT safe if you keep multiple Board instances alive simultaneously
	// (e.g., for undo/redo or snapshots). If that's needed, clone ranks before mutation.
	brd.ranks[brd.position.rank].squares[brd.position.file].model = updatedModel

	return brd, cmd
}

func (brd Board) View() tea.View {
	tbl := table.New()

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
		tbl.Headers(headers...)
	}

	// Build rows from ranks
	for _, rank := range brd.ranks {
		var row []string
		for _, square := range rank.squares {
			// Render each square's piece as a table cell
			if piece, ok := square.model.(Piece); ok {
				row = append(row, piece.Render())
			} else {
				row = append(row, "?")
			}
		}
		tbl.Row(row...)
	}

	// Apply styling to highlight focused cell
	tbl.StyleFunc(func(row, col int) lipgloss.Style {
		if row == brd.position.rank && col == brd.position.file {
			// Highlight the focused square
			return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
		}
		return lipgloss.NewStyle()
	})

	return tea.NewView(tbl)
}

func (brd Board) MoveUp() Board {
	if brd.position.rank > 0 {
		brd.position.rank--
	}
	return brd
}

func (brd Board) MoveDown() Board {
	if brd.position.rank < brd.height-1 {
		brd.position.rank++
	}
	return brd
}

func (brd Board) MoveLeft() Board {
	if brd.position.file > 0 {
		brd.position.file--
	}
	return brd
}

func (brd Board) MoveRight() Board {
	if brd.position.file < brd.width-1 {
		brd.position.file++
	}
	return brd
}

// unexported

type position struct {
	rank int
	file int
}
