package table

import (
	"parcours/message"

	tea "charm.land/bubbletea/v2"
)

func (pnl TablePanel) selectedCmd() tea.Cmd {

	line, err := pnl.selectedLine()
	if err != nil {
		return message.ErrorCmd(err)
	}

	row := pnl.selected + 1 // Todo: herd row/line confusion

	return func() tea.Msg {
		return message.SelectedMsg{
			Row: row,
			Id:  line.Id,
		}
	}
}

func (pnl TablePanel) filterCmd() tea.Cmd {

	field, value, err := pnl.selectedCell()
	if err != nil {
		return message.ErrorCmd(err)
	}

	return func() tea.Msg {
		return message.OpenFilterMsg{
			Field: field,
			Value: value.String(),
		}
	}
}
