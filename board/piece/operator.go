package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

// Operator cycles through a list of options
type Operator struct {
	options  []string
	selected int
}

func NewOperator(options []string, selected int) Operator {
	if selected < 0 || selected >= len(options) {
		selected = 0
	}
	return Operator{
		options:  options,
		selected: selected,
	}
}

func (o Operator) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "h":
			o.selected--
			if o.selected < 0 {
				o.selected = len(o.options) - 1
			}
			return o, o.changedCmd()
		case "right", "l":
			o.selected++
			if o.selected >= len(o.options) {
				o.selected = 0
			}
			return o, o.changedCmd()
		}
	}
	return o, nil
}

func (o Operator) changedCmd() tea.Cmd {
	return func() tea.Msg {
		return &OperatorChangedMsg{
			Selected: o.Selected(),
			Index:    o.selected,
		}
	}
}

func (o Operator) Selected() string {
	if o.selected < 0 || o.selected >= len(o.options) {
		return ""
	}
	return o.options[o.selected]
}

func (o Operator) SelectedIndex() int {
	return o.selected
}

func (o Operator) Render() string {
	if o.selected < 0 || o.selected >= len(o.options) {
		return "?"
	}
	return o.options[o.selected]
}

func (o Operator) Value() string {
	return o.Selected()
}
