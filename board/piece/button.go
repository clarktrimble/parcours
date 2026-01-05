package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

// Button is a pressable button cell
type Button struct {
	label string
	key   string // Key that triggers the button
	rank  int
	file  int
}

func NewButton(label, key string) Button {
	return Button{
		label: label,
		key:   key,
	}
}

func (b Button) Init() tea.Cmd { return nil }

func (b Button) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case board.PositionMsg:
		b.rank = msg.Rank
		b.file = msg.File
		return b, nil
	case tea.KeyPressMsg:
		if msg.String() == b.key {
			return b, func() tea.Msg {
				return PressedMsg{Rank: b.rank, File: b.file}
			}
		}
	}
	return b, nil
}

func (b Button) View() tea.View {
	return tea.NewView(b.label)
}
