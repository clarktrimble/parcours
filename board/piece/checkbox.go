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

func (c Checkbox) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
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

func (c Checkbox) Checked() bool {
	return c.checked
}

func (c Checkbox) Render() string {
	if c.checked {
		return "[x]"
	}
	return "[ ]"
}

func (c Checkbox) Value() string {
	if c.checked {
		return "true"
	}
	return "false"
}
