package cell

import tea "charm.land/bubbletea/v2"

// Checkbox is a toggleable checkbox cell
type Checkbox struct {
	checked bool
}

func NewCheckbox(checked bool) Checkbox {
	return Checkbox{checked: checked}
}

func (c Checkbox) Init() tea.Cmd {
	return nil
}

func (c Checkbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "t" || msg.String() == " " {
			c.checked = !c.checked
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

func (c Checkbox) Checked() bool {
	return c.checked
}

func (c Checkbox) Render() string {
	if c.checked {
		return "[x]"
	}
	return "[ ]"
}
