package piece

import "parcours/board"

// Ensure messages implement board.PieceMsg
var (
	_ board.PieceMsg = CheckedMsg{}
	_ board.PieceMsg = OperatorChangedMsg{}
	_ board.PieceMsg = ValueChangedMsg{}
)

// CheckedMsg is sent when a checkbox is toggled
type CheckedMsg struct {
	Rank    int
	File    int
	Checked bool
}

func (CheckedMsg) IsPieceMsg() {}

// OperatorChangedMsg is sent when an operator selection changes
type OperatorChangedMsg struct {
	Rank     int
	File     int
	Selected string
	Index    int
}

func (OperatorChangedMsg) IsPieceMsg() {}

// ValueChangedMsg is sent when a text input value changes
type ValueChangedMsg struct {
	Rank  int
	File  int
	Value string
}

func (ValueChangedMsg) IsPieceMsg() {}
