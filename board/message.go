package board

// SquareMsg contains the board cursor position and selected cell info
type SquareMsg struct {
	Rank  int    // Row position within board (0-indexed)
	File  int    // Column position within board (0-indexed)
	Field string // Field name from the column header
	Value string // Cell value from the piece
}

// PositionMsg tells a piece its position on the board
type PositionMsg struct {
	Rank int
	File int
}

// Navigation directions
const (
	NavDown     = "down"
	NavUp       = "up"
	NavPageDown = "pagedown"
	NavPageUp   = "pageup"
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
