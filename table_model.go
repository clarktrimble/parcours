package parcours

import (
	tea "charm.land/bubbletea/v2"
)

// TableModel handles the table view display and navigation
type TableModel struct {
	Store  Store
	Layout *Layout

	// View data
	Fields     []Field
	Lines      []Line
	TotalLines int

	// Display state
	ScrollOffset int
	SelectedRow  int
	Width        int
	Height       int
	Focused      bool
}

type tableLoadDataMsg struct {
	fields []Field
	lines  []Line
	count  int
	err    error
}

func NewTableModel(store Store, layout *Layout) *TableModel {
	return &TableModel{
		Store:  store,
		Layout: layout,
	}
}

func (m *TableModel) Init() tea.Cmd {
	return m.loadData()
}

func (m *TableModel) loadData() tea.Cmd {
	return func() tea.Msg {
		fields, count, err := m.Store.GetView()
		if err != nil {
			return tableLoadDataMsg{err: err}
		}

		lines, err := m.Store.GetPage(m.ScrollOffset, 20)
		if err != nil {
			return tableLoadDataMsg{err: err}
		}

		return tableLoadDataMsg{
			fields: fields,
			lines:  lines,
			count:  count,
		}
	}
}

func (m *TableModel) Update(msg tea.Msg) (*TableModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tableLoadDataMsg:
		if msg.err != nil {
			// TODO: handle error
			return m, nil
		}
		m.Fields = msg.fields
		m.Lines = msg.lines
		m.TotalLines = msg.count
		return m, nil

	case tea.KeyPressMsg:
		// Only handle keys when focused
		if !m.Focused {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.SelectedRow > 0 {
				m.SelectedRow--
			} else if m.ScrollOffset > 0 {
				m.ScrollOffset--
				return m, m.loadData()
			}
		case "down", "j":
			if m.SelectedRow < len(m.Lines)-1 {
				m.SelectedRow++
			} else if m.ScrollOffset+len(m.Lines) < m.TotalLines {
				m.ScrollOffset++
				return m, m.loadData()
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, nil
}

func (m *TableModel) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	return RenderTable(m.Fields, m.Lines, m.SelectedRow, m.Width, m.Layout)
}

// GetSelectedID returns the ID of the currently selected line
func (m *TableModel) GetSelectedID() string {
	if len(m.Lines) == 0 || m.SelectedRow >= len(m.Lines) {
		return ""
	}
	return m.Lines[m.SelectedRow][0].String()
}
