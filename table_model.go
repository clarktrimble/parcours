package parcours

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/pkg/errors"
)

const (
	headerHeight = 2
)

// TablePane handles the table view display and navigation state
type TablePane struct {
	// Display state
	ScrollOffset int
	SelectedRow  int
	Width        int
	Height       int
	Focused      bool
	CurrentLines int  // Number of lines currently loaded
	TotalLines   int  // Total lines available (set by parent)
	initialized  bool // Whether initial data has been requested

	layout *Layout
	table  *table.Table
}

func NewTablePane(layout *Layout) *TablePane {

	lgt := table.New()

	// Set headers
	var headers []string
	for _, col := range layout.Columns {
		if col.Hidden || col.Demote {
			continue
		}
		headers = append(headers, col.Field)
	}
	lgt.Headers(headers...)

	// Configure styling - only separator between header and data
	lgt.Border(lipgloss.Border{
		Top:         "─", // Horizontal parts of separator
		Middle:      "─", // Between columns in separator
		MiddleLeft:  "─", // Left edge of separator
		MiddleRight: "─", // Right edge of separator
	}).
		BorderTop(false).    // Disable top border
		BorderBottom(false). // Disable bottom border
		BorderLeft(false).   // Disable left border
		BorderRight(false).  // Disable right border
		BorderColumn(false). // Disable column separators
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240")))

	return &TablePane{
		Focused: true, // Start with table focused // Todo: elsewhere
		table:   lgt,
		layout:  layout,
	}
}

func (m *TablePane) Update(msg tea.Msg) (*TablePane, tea.Cmd) {
	switch msg := msg.(type) {
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
				// Signal parent to load new page
				pageSize := m.Height - headerHeight
				return m, func() tea.Msg {
					return getPageMsg{
						offset: m.ScrollOffset,
						size:   pageSize,
					}
				}
			}

		case "down", "j":
			if m.SelectedRow < m.CurrentLines-1 {
				m.SelectedRow++
			} else if m.ScrollOffset+m.CurrentLines < m.TotalLines {
				m.ScrollOffset++
				// Signal parent to load new page
				pageSize := m.Height - headerHeight
				return m, func() tea.Msg {
					return getPageMsg{
						offset: m.ScrollOffset,
						size:   pageSize,
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Request initial data load when we first get dimensions
		if !m.initialized && m.Height > 0 {
			m.initialized = true
			pageSize := m.Height - headerHeight
			if pageSize > 0 {
				return m, func() tea.Msg {
					return getPageMsg{
						offset: 0,
						size:   pageSize,
					}
				}
			}
		}
	}

	return m, nil
}

// SelectedId returns the id of the currently selected line
func (m *TablePane) SelectedId(lines []Line) (id string, err error) {

	if len(lines) == 0 || m.SelectedRow >= len(lines) {
		err = errors.Errorf("index %d is out of bounds of %d lines", m.SelectedRow, len(lines))
		return
	}

	id = lines[m.SelectedRow][0].String() //Todo: add Id() method to Line?
	return
}

// Render renders the table with the given data
func (m *TablePane) Render(fields []Field, lines []Line, layout *Layout) string {

	// Todo: elsewhere and use initialized
	if m.Width == 0 {
		return "Loading..."
	}

	// setup style func and field index

	m.table.StyleFunc(func(row, col int) lipgloss.Style {
		if row == m.SelectedRow {
			return lipgloss.NewStyle().Background(lipgloss.Color("63"))
		}
		return lipgloss.NewStyle()
	})

	fieldIndex := make(map[string]int)
	for i, f := range fields {
		fieldIndex[f.Name] = i
	}

	// repopulate table

	m.table.ClearRows()
	for _, line := range lines {
		var row []string
		for _, col := range m.layout.Columns {
			if col.Hidden || col.Demote {
				continue
			}

			// Get field and format value
			field := fields[fieldIndex[col.Field]]
			idx := fieldIndex[col.Field]
			formatted := formatValue(line[idx], field.Type, col.Format)

			// Pad/truncate to exact width
			padded := fmt.Sprintf("%-*.*s", col.Width, col.Width, formatted)
			row = append(row, padded)
		}
		m.table.Row(row...)
	}

	return m.table.Render()
}
