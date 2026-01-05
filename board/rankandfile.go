package board

import tea "charm.land/bubbletea/v2"

// Square represents a square of the board.
type Square struct {
	piece tea.Model
}

// Rank represents a rank across the board.
type Rank struct {
	squares []Square
}

// NewRank creates a Rank from pieces.
func NewRank(pieces []tea.Model) Rank {
	squares := make([]Square, len(pieces))
	for i, piece := range pieces {
		squares[i] = Square{
			piece: piece,
		}
	}
	return Rank{squares: squares}
}

// Append appends a piece.
func (rank *Rank) Append(piece tea.Model) {
	rank.squares = append(rank.squares, Square{piece: piece})
}

// Ranks is a slice of Rank.
type Ranks []Rank

// File represents a file down the board.
type File struct {
	Header string
	Width  int
	SrcIdx int // Index into source data
}

// Files is a slice of File.
type Files []File

// Equal checks if Files have same Headers and Widths.
func (files Files) Equal(other Files) (equal bool) {
	if len(files) != len(other) {
		return
	}

	for i := range files {
		if files[i].Header != other[i].Header || files[i].Width != other[i].Width {
			return
		}
	}

	equal = true
	return
}
