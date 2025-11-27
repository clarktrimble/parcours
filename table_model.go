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
	// Navigation state
	SelectedLine int  // Absolute line position (0 to TotalLines-1)
	ScrollOffset int  // Line at top of viewport
	TotalLines   int  // Total lines available (set by parent)
	Width        int
	Height       int
	Focused      bool
	initialized  bool // Whether initial data has been requested

	layout *Layout
	table  *table.Table
}

// pageSize returns the number of rows that fit on screen
func (m *TablePane) pageSize() int {
	return m.Height - headerHeight
}

// selectedRow returns the row position within the current page
func (m *TablePane) selectedRow() int {
	return m.SelectedLine - m.ScrollOffset
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

		oldScrollOffset := m.ScrollOffset
		pageSize := m.pageSize()

		switch msg.String() {
		case "up", "k":
			if m.SelectedLine > 0 {
				m.SelectedLine--
			}

		case "down", "j":
			if m.SelectedLine < m.TotalLines-1 {
				m.SelectedLine++
			}

		case "pgup", "ctrl+u":
			m.SelectedLine -= pageSize
			if m.SelectedLine < 0 {
				m.SelectedLine = 0
			}

		case "pgdown", "ctrl+d":
			m.SelectedLine += pageSize
			if m.SelectedLine >= m.TotalLines {
				m.SelectedLine = m.TotalLines - 1
			}

		case "g":
			m.SelectedLine = 0

		case "G":
			m.SelectedLine = m.TotalLines - 1
		}

		// Adjust ScrollOffset to keep SelectedLine visible
		if m.SelectedLine < m.ScrollOffset {
			m.ScrollOffset = m.SelectedLine
		} else if m.SelectedLine >= m.ScrollOffset+pageSize {
			m.ScrollOffset = m.SelectedLine - pageSize + 1
		}

		// If we've scrolled to a different page, request new data
		if m.ScrollOffset != oldScrollOffset {
			return m, func() tea.Msg {
				return getPageMsg{
					offset: m.ScrollOffset,
					size:   pageSize,
				}
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Request initial data load when we first get dimensions
		if !m.initialized && m.Height > 0 {
			m.initialized = true
			pageSize := m.pageSize()
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
	selectedRow := m.selectedRow()

	if len(lines) == 0 || selectedRow >= len(lines) {
		err = errors.Errorf("index %d is out of bounds of %d lines", selectedRow, len(lines))
		return
	}

	id = lines[selectedRow][0].String() //Todo: add Id() method to Line?
	return
}

// Render renders the table with the given data
func (m *TablePane) Render(fields []Field, lines []Line, layout *Layout) string {

	// Todo: elsewhere and use initialized
	if m.Width == 0 {
		return "Loading..."
	}

	// setup style func and field index
	selectedRow := m.selectedRow()

	m.table.StyleFunc(func(row, col int) lipgloss.Style {
		if row == selectedRow {
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
