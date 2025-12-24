package message

import tea "charm.land/bubbletea/v2"

// GetPageCmd returns a command to request a page of data
func GetPageCmd(offset, size int) tea.Cmd {
	return func() tea.Msg {
		return GetPageMsg{
			Offset: offset,
			Size:   size,
		}
	}
}
