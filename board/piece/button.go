package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

// PressedMsg is sent when a button is pressed
type PressedMsg struct{}

// Button is a pressable button cell
type Button struct {
	label string
	key   string // Key that triggers the button
}

func NewButton(label, key string) Button {
	return Button{
		label: label,
		key:   key,
	}
}

func (b Button) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == b.key {
			return b, func() tea.Msg {
				return PressedMsg{}
			}
		}
	}
	return b, nil
}

func (b Button) Label() string {
	return b.label
}

func (b Button) Render() string {
	return b.label
}

func (b Button) Value() string {
	return b.label
}
