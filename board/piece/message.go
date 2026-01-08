package piece

// PressedMsg is sent when a button is pressed.
type PressedMsg struct {
	Rank int
	File int
}

// CheckedMsg is sent when a checkbox is toggled.
type CheckedMsg struct {
	Rank    int
	File    int
	Checked bool
}

// OperatorChangedMsg is sent when an operator selection changes.
type OperatorChangedMsg struct {
	Rank     int
	File     int
	Selected string
	Index    int
}

// ValueChangedMsg is sent when a text input value changes.
type ValueChangedMsg struct {
	Rank  int
	File  int
	Value string
}
