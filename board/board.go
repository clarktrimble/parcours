package board

import "slices"

type empty struct{}

func (empty empty) String() string {
	return "empty"
}

type Value interface {
	String() string
}

type Field interface {
	String() string
}

type Square struct {
	value    Value
	position position
}

type Rank struct {
	squares []Square
}

type File struct {
	field Field
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
				value:    empty{},
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

func (brd Board) Rank() []Square {
	return slices.Clone(brd.ranks[brd.position.rank].squares)
}

func (brd Board) Square() Square {
	return brd.ranks[brd.position.rank].squares[brd.position.file]
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
