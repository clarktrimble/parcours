package parcours

import tea "charm.land/bubbletea/v2"

// getPage loads a page of data from the store
func (m Model) getPage(offset, size int) tea.Cmd {
	return func() tea.Msg {
		fields, count, err := m.Store.GetView()
		if err != nil {
			return errorMsg{err: err}
		}

		lines, err := m.Store.GetPage(offset, size)
		if err != nil {
			return errorMsg{err: err}
		}

		return pageMsg{
			fields: fields,
			lines:  lines,
			count:  count,
		}
	}
}

// getLine loads a full record from the store
func (m Model) getLine(id string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.Store.GetLine(id)
		if err != nil {
			return errorMsg{err: err}
		}

		parsed := parseJsonFields(data, m.Layout)
		return lineMsg{data: parsed}
	}
}

// switchToTable switches to the table screen and manages focus
func (m *Model) switchToTable() tea.Cmd {
	m.CurrentScreen = TableScreen
	m.TablePane.Focused = true
	m.DetailPane.Focused = false

	return nil // honorary cmd here
}

// switchToDetail switches to the detail screen and manages focus
func (m *Model) switchToDetail() tea.Cmd {
	m.CurrentScreen = DetailScreen
	m.TablePane.Focused = false
	m.DetailPane.Focused = true

	m.DetailPane.ScrollOffset = 0

	id, err := m.TablePane.SelectedId(m.Lines)
	if err != nil {
		return func() tea.Msg {
			return errorMsg{err: err}
		}
	}

	return m.getLine(id)
}
