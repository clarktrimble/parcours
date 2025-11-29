package parcours

import tea "charm.land/bubbletea/v2"

// errorCmd creates an error cmd
func errorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{err: err}
	}
}

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
// Todo: this is bucket-brigade?
func (m Model) getLine(id string) tea.Cmd {
	return func() tea.Msg {
		line, err := m.Store.GetLine(id)
		if err != nil {
			return errorMsg{err: err}
		}

		return lineMsg{line: line}
	}
}

// switchToTable switches to the table screen and manages focus
func (m Model) switchToTable() (Model, tea.Cmd) {
	m.CurrentScreen = TableScreen
	m.TablePanel.Focused = true
	m.DetailPanel.Focused = false

	return m, nil // honorary cmd here
}

// switchToDetail switches to the detail screen and manages focus
func (m Model) switchToDetail() (Model, tea.Cmd) {
	m.CurrentScreen = DetailScreen
	m.TablePanel.Focused = false
	m.DetailPanel.Focused = true

	id, err := m.TablePanel.SelectedId(m.Lines)
	if err != nil {
		return m, errorCmd(err)
	}

	return m, m.getLine(id)
}

// reloadColumns loads layout from file and updates, and gets page
func (m Model) reloadColumns() (Model, tea.Cmd) {

	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		return m, errorCmd(err)
	}

	err = layout.promote(m.Store)
	if err != nil {
		return m, errorCmd(err)
	}

	// Get updated fields after promotion
	fields, _, err := m.Store.GetView()
	if err != nil {
		return m, errorCmd(err)
	}

	m.Layout = layout
	m.Lines = nil // Clear old lines to avoid render mismatch
	m.TablePanel = m.TablePanel.SetColumns(layout.Columns, fields)
	m.DetailPanel = m.DetailPanel.SetColumns(layout.Columns)

	return m, m.getPage(m.TablePanel.ScrollOffset, m.TablePanel.pageSize())
}

// reloadFilter loads layout from file and updates, and resets and gets page
func (m Model) reloadFilter() (Model, tea.Cmd) {
	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		return m, errorCmd(err)
	}

	err = m.Store.SetView(layout.Filter, nil)
	if err != nil {
		return m, errorCmd(err)
	}

	// Get updated count after filter change
	_, count, err := m.Store.GetView()
	if err != nil {
		return m, errorCmd(err)
	}

	m.Layout = layout
	m.Lines = nil // Clear old lines to avoid render mismatch
	m.TablePanel.TotalLines = count
	m.TablePanel.SelectedLine = 0
	m.TablePanel.ScrollOffset = 0

	return m, m.getPage(0, m.TablePanel.pageSize())
}
