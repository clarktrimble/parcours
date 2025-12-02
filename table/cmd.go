package table

import (
	"parcours/message"

	tea "charm.land/bubbletea/v2"
	"github.com/pkg/errors"
)

func (pnl TablePanel) selectedCmd() tea.Cmd {

	local := pnl.selectedLocal()

	if local < 0 || local >= len(pnl.lines) {
		return message.ErrorCmd(errors.Errorf("cannot index %d in page of %d lines", local, len(pnl.lines)))
	}

	row := pnl.selected + 1 // Todo: herd row/line confusion
	id := pnl.lines[local].Id

	return func() tea.Msg {
		return message.SelectedMsg{
			Row: row,
			Id:  id,
		}
	}
}
