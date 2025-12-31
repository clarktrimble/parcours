package board

// SquareMsg contains the board cursor position
type SquareMsg struct {
	Rank int // Row position within board (0-indexed)
	File int // Column position within board (0-indexed)
}

// PositionMsg tells a piece its position on the board
type PositionMsg struct {
	Rank int
	File int
}

// Navigation directions
const (
	NavUp       = "up"
	NavDown     = "down"
	NavLeft     = "left"
	NavRight    = "right"
	NavPageUp   = "pageup"
	NavPageDown = "pagedown"
	NavTop      = "top"
	NavBottom   = "bottom"
)

// NavMsg signals navigation that hit a boundary
type NavMsg struct {
	Direction string
}

// MoveTo positions
type MoveTo int

const (
	Top MoveTo = iota
	Bottom
)

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
