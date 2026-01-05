package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

// Checkbox is a toggleable checkbox cell
type Checkbox struct {
	checked bool
	rank    int
	file    int
}

func NewCheckbox(checked bool) Checkbox {
	return Checkbox{checked: checked}
}

func (c Checkbox) Init() tea.Cmd { return nil }

func (c Checkbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case board.PositionMsg:
		c.rank = msg.Rank
		c.file = msg.File
		return c, nil
	case tea.KeyPressMsg:
		if msg.String() == "t" || msg.String() == " " {
			c.checked = !c.checked
			return c, func() tea.Msg {
				return CheckedMsg{Rank: c.rank, File: c.file, Checked: c.checked}
			}
		}
	}
	return c, nil
}

func (c Checkbox) View() tea.View {
	if c.checked {
		return tea.NewView("[x]")
	}
	return tea.NewView("[ ]")
}
