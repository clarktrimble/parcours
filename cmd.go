package parcours

import (
	tea "charm.land/bubbletea/v2"

	"parcours/detail"
	"parcours/message"
	"parcours/table"
)

// errorCmd creates an error cmd
func errorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return message.ErrorMsg{Err: err}
	}
}

// getPage gets a page of lines from the store
func (m Model) getPage(offset, size int) tea.Cmd {
	return func() tea.Msg {

		fields, count, err := m.Store.GetView()
		if err != nil {
			return message.ErrorMsg{Err: err}
		}

		lines, err := m.Store.GetPage(offset, size)
		if err != nil {
			return message.ErrorMsg{Err: err}
		}

		return table.PageMsg{
			Fields: fields,
			Lines:  lines,
			Count:  count,
		}
	}
}

// getLine gets a full line from the store
func (m Model) getLine(id string) tea.Cmd {
	return func() tea.Msg {
		line, err := m.Store.GetLine(id)
		if err != nil {
			return message.ErrorMsg{Err: err}
		}

		return detail.LineMsg{Line: line}
	}
}

// Todo: the following "cmd"s return model as well, is this bt/elm legit?

// switchToTable switches to the table screen and manages focus
func (m Model) switchToTable() (Model, tea.Cmd) {
	m.CurrentScreen = TableScreen
	m.TablePanel.Focused = true // Todo: elmify
	m.DetailPanel.Focused = false

	return m, nil // honorary cmd here
}

// switchToDetail switches to the detail screen and manages focus
func (m Model) switchToDetail() (Model, tea.Cmd) {
	m.CurrentScreen = DetailScreen
	m.TablePanel.Focused = false // Todo: elmify
	m.DetailPanel.Focused = true

	id, err := m.TablePanel.SelectedId()
	if err != nil {
		return m, errorCmd(err)
	}

	return m, m.getLine(id)
}

// reloadColumns loads layout from file and updates, and gets page
func (m Model) reloadColumns() tea.Cmd {

	layout, err := loadLayout(layoutFile)
	if err != nil {
		return errorCmd(err)
	}

	err = layout.promote(m.Store)
	if err != nil {
		return errorCmd(err)
	}

	// Get updated fields after promotion
	fields, _, err := m.Store.GetView()
	if err != nil {
		return errorCmd(err)
	}

	// Send column updates to both panels
	return tea.Batch(
		func() tea.Msg {
			return table.ColumnsMsg{Columns: layout.Columns, Fields: fields}
		},
		func() tea.Msg {
			return detail.ColumnsMsg{Columns: layout.Columns}
		},
	)
}

// reloadFilter loads layout from file and updates, and resets and gets page
func (m Model) reloadFilter() tea.Cmd {

	layout, err := loadLayout(layoutFile)
	if err != nil {
		return errorCmd(err)
	}

	// Todo: what about "sorts"?
	err = m.Store.SetView(layout.Filter, nil)
	if err != nil {
		return errorCmd(err)
	}

	return tea.Batch( // Todo: or Sequence??
		func() tea.Msg { return table.ResetMsg{} },
		m.getPage(0, m.TablePanel.PageSize()),
	)
}
