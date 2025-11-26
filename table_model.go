package parcours

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const (
	// tableHeaderLines is the number of lines used by the table header
	// (header row + separator line)
	tableHeaderLines = 2
)

// TablePane handles the table view display and navigation state
type TablePane struct {
	// Display state
	ScrollOffset   int
	SelectedRow    int
	Width          int
	Height         int
	Focused        bool
	CurrentLines   int // Number of lines currently loaded
	TotalLines     int // Total lines available (set by parent)
	initialized    bool // Whether initial data has been requested

	// Cached table for rendering
	table *table.Table
}

// getPageMsg signals parent to load a new page of data
type getPageMsg struct {
	Offset int
	Size   int
}

func NewTablePane() *TablePane {
	return &TablePane{
		Focused: true, // Start with table focused
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
				pageSize := m.Height - tableHeaderLines
				return m, func() tea.Msg {
					return getPageMsg{
						Offset: m.ScrollOffset,
						Size:   pageSize,
					}
				}
			}

		case "down", "j":
			if m.SelectedRow < m.CurrentLines-1 {
				m.SelectedRow++
			} else if m.ScrollOffset+m.CurrentLines < m.TotalLines {
				m.ScrollOffset++
				// Signal parent to load new page
				pageSize := m.Height - tableHeaderLines
				return m, func() tea.Msg {
					return getPageMsg{
						Offset: m.ScrollOffset,
						Size:   pageSize,
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
			pageSize := m.Height - tableHeaderLines
			if pageSize > 0 {
				return m, func() tea.Msg {
					return getPageMsg{
						Offset: 0,
						Size:   pageSize,
					}
				}
			}
		}
	}

	return m, nil
}

// Render renders the table with the given data
func (m *TablePane) Render(fields []Field, lines []Line, layout *Layout) string {
	if m.Width == 0 {
		return "Loading..."
	}

	// Initialize table on first render
	if m.table == nil {
		m.table = table.New()

		// Set headers
		var headers []string
		for _, col := range layout.Columns {
			if col.Hidden || col.Demote {
				continue
			}
			headers = append(headers, col.Field)
		}
		m.table.Headers(headers...)

		// Configure styling - only separator between header and data
		m.table.Border(lipgloss.Border{
			Top:         "─", // Horizontal parts of separator
			Middle:      "─", // Between columns in separator
			MiddleLeft:  "─", // Left edge of separator
			MiddleRight: "─", // Right edge of separator
		}).
			BorderTop(false).     // Disable top border
			BorderBottom(false).  // Disable bottom border
			BorderLeft(false).    // Disable left border
			BorderRight(false).   // Disable right border
			BorderColumn(false).  // Disable column separators
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240")))

		m.table.StyleFunc(func(row, col int) lipgloss.Style {
			if row == m.SelectedRow {
				return lipgloss.NewStyle().Background(lipgloss.Color("63"))
			}
			return lipgloss.NewStyle()
		})
	}

	// Update data
	return RenderTable(m.table, fields, lines, m.SelectedRow, m.Width, layout)
}

// GetSelectedID returns the ID of the currently selected line
func (m *TablePane) GetSelectedID(lines []Line) string {
	if len(lines) == 0 || m.SelectedRow >= len(lines) {
		return ""
	}
	return lines[m.SelectedRow][0].String()
}
