package piece

import "parcours/board"

// Ensure messages implement board.PieceMsg
var (
	_ board.PieceMsg = (*CheckedMsg)(nil)
	_ board.PieceMsg = (*OperatorChangedMsg)(nil)
	_ board.PieceMsg = (*ValueChangedMsg)(nil)
)

// CheckedMsg is sent when a checkbox is toggled
type CheckedMsg struct {
	Rank    int
	File    int
	Checked bool
}

func (CheckedMsg) IsPieceMsg() {}
func (m *CheckedMsg) SetPosition(rank, file int) {
	m.Rank = rank
	m.File = file
}

// OperatorChangedMsg is sent when an operator selection changes
type OperatorChangedMsg struct {
	Rank     int
	File     int
	Selected string
	Index    int
}

func (OperatorChangedMsg) IsPieceMsg() {}
func (m *OperatorChangedMsg) SetPosition(rank, file int) {
	m.Rank = rank
	m.File = file
}

// ValueChangedMsg is sent when a text input value changes
type ValueChangedMsg struct {
	Rank  int
	File  int
	Value string
}

func (ValueChangedMsg) IsPieceMsg() {}
func (m *ValueChangedMsg) SetPosition(rank, file int) {
	m.Rank = rank
	m.File = file
}
