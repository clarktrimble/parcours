package board

import tea "charm.land/bubbletea/v2"

// PositionMsg contains a board position (rank and file)
type PositionMsg struct {
	Rank int
	File int
}

// Navigation directions
const (
	NavUp     = "up"
	NavDown   = "down"
	NavLeft   = "left"
	NavRight  = "right"
	NavTop    = "top"
	NavBottom = "bottom"
)

// NavMsg signals navigation that hit a boundary
type NavMsg struct {
	Direction string
	Count     int
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

// AppendMsg adds a rank to the board
type AppendMsg struct {
	Rank Rank
}

// RemoveMsg removes a rank from the board
type RemoveMsg struct {
	Index int
}

// cmd helpers

func (brd Board) positionCmd() tea.Cmd {
	pos := PositionMsg{
		Rank: brd.cursor.rank,
		File: brd.cursor.file,
	}
	return func() tea.Msg { return pos }
}

func navCmd(dir string) tea.Cmd {
	return func() tea.Msg { return NavMsg{Direction: dir, Count: 1} }
}
