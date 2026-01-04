package parcours

import (
	tea "charm.land/bubbletea/v2"

	"parcours/detail"
	"parcours/linepanel"
)

// getPage gets a page of lines from the store
func (m Model) getPage(offset, size int) tea.Cmd {
	return func() tea.Msg {
		_, count, err := m.Store.GetView()
		if err != nil {
			return err
		}

		linesData, err := m.Store.GetPage(offset, size)
		if err != nil {
			return err
		}

		return linepanel.PageMsg{
			Lines: linesData,
			Count: count,
		}
	}
}

// getLine gets a full line from the store
func (m Model) getLine(id string) tea.Cmd {
	return func() tea.Msg {
		line, err := m.Store.GetLine(id)
		if err != nil {
			return err
		}

		return detail.LineMsg{Line: line}
	}
}

// reloadColumns loads layout from file and updates, and gets page
func (m Model) reloadColumns() tea.Cmd {

	layout, err := loadLayout(layoutFile)
	if err != nil {
		return func() tea.Msg { return err }
	}

	err = layout.promote(m.Store)
	if err != nil {
		return func() tea.Msg { return err }
	}

	// Get updated fields after promotion
	fields, _, err := m.Store.GetView()
	if err != nil {
		return func() tea.Msg { return err }
	}

	// Send column updates to both panels
	return tea.Batch(
		func() tea.Msg {
			return linepanel.ColumnsMsg{Columns: layout.Columns, Fields: fields}
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
		return func() tea.Msg { return err }
	}

	// Todo: what about "sorts"?
	err = m.Store.SetView(layout.Filter, nil)
	if err != nil {
		return func() tea.Msg { return err }
	}

	return func() tea.Msg { return linepanel.ResetMsg{} }
	//return tea.Batch( // Todo: or Sequence??
	//func() tea.Msg { return lines.ResetMsg{} },
	//m.getPage(0, m.TablePanel.PageSize()),
	//)
}
