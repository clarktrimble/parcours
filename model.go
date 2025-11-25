package parcours

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
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
	ShowFull     bool
	FullRecord   map[string]any
}

type loadDataMsg struct {
	fields []Field
	lines  []Line
	count  int
	err    error
}

type fullRecordMsg struct {
	data map[string]any
	err  error
}

// NewModel creates a new TUI model with the given store.
func NewModel(store Store) Model {
	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		panic(err) // TODO: handle better
	}

	// Promote fields from layout
	// TODO: improve error handling/logging so we can see promotion failures
	for _, col := range layout.Columns {
		// Skip demoted fields
		if col.Demote {
			continue
		}
		// Skip base fields that already exist
		if col.Field == "timestamp" || col.Field == "message" {
			continue
		}
		if err := store.Promote(col.Field); err != nil {
			// TODO: log error instead of panicking
			panic(err)
		}
	}

	return Model{
		Store:  store,
		Layout: layout,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadData()
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		fields, count, err := m.Store.GetView()
		if err != nil {
			return loadDataMsg{err: err}
		}

		lines, err := m.Store.GetPage(m.ScrollOffset, 20)
		if err != nil {
			return loadDataMsg{err: err}
		}

		return loadDataMsg{
			fields: fields,
			lines:  lines,
			count:  count,
		}
	}
}

func (m Model) fetchFullRecord(id string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.Store.GetJson(id)
		return fullRecordMsg{data: data, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadDataMsg:
		if msg.err != nil {
			// TODO: handle error
			return m, nil
		}
		m.Fields = msg.fields
		m.Lines = msg.lines
		m.TotalLines = msg.count
		return m, nil

	case fullRecordMsg:
		if msg.err != nil {
			// Show error in JSON view
			m.FullRecord = map[string]any{"error": msg.err.Error()}
			return m, nil
		}
		// Parse JSON fields before storing
		m.FullRecord = parseJsonFields(msg.data, m.Layout)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "enter":
			m.ShowFull = !m.ShowFull
			if m.ShowFull && len(m.Lines) > 0 {
				// Fetch JSON for selected line
				id := m.Lines[m.SelectedRow][0].String()
				return m, m.fetchFullRecord(id)
			} else {
				// Clear data when closing full record view
				m.FullRecord = nil
			}
		case "up", "k":
			if m.ShowFull {
				// Navigate in full record view
				if m.SelectedRow > 0 {
					m.SelectedRow--
					id := m.Lines[m.SelectedRow][0].String()
					return m, m.fetchFullRecord(id)
				} else if m.ScrollOffset > 0 {
					m.ScrollOffset--
					return m, tea.Sequence(m.loadData(), func() tea.Msg {
						// After loading, fetch the new record
						if len(m.Lines) > 0 {
							id := m.Lines[m.SelectedRow][0].String()
							return m.fetchFullRecord(id)()
						}
						return nil
					})
				}
				return m, nil
			}
			if m.SelectedRow > 0 {
				m.SelectedRow--
			} else if m.ScrollOffset > 0 {
				m.ScrollOffset--
				return m, m.loadData()
			}
		case "down", "j":
			if m.ShowFull {
				// Navigate in full record view
				if m.SelectedRow < len(m.Lines)-1 {
					m.SelectedRow++
					id := m.Lines[m.SelectedRow][0].String()
					return m, m.fetchFullRecord(id)
				} else if m.ScrollOffset+len(m.Lines) < m.TotalLines {
					m.ScrollOffset++
					return m, tea.Sequence(m.loadData(), func() tea.Msg {
						// After loading, fetch the new record
						if len(m.Lines) > 0 {
							id := m.Lines[m.SelectedRow][0].String()
							return m.fetchFullRecord(id)()
						}
						return nil
					})
				}
				return m, nil
			}
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

func (m Model) View() tea.View {
	if m.Width == 0 {
		return tea.NewView("Loading...")
	}

	var b strings.Builder

	if m.ShowFull {
		// Show full record JSON
		if m.FullRecord != nil {
			// Pretty-print JSON with HTML escaping disabled
			var buf strings.Builder
			encoder := json.NewEncoder(&buf)
			encoder.SetIndent("", "  ")
			encoder.SetEscapeHTML(false)

			err := encoder.Encode(m.FullRecord)
			if err != nil {
				b.WriteString("Error pretty-printing JSON: " + err.Error())
			} else {
				// Encode adds a trailing newline, trim it
				b.WriteString(strings.TrimSuffix(buf.String(), "\n"))
			}
		} else {
			b.WriteString("Loading full record...")
		}
	} else {
		// Render table
		table := RenderTable(m.Fields, m.Lines, m.SelectedRow, m.Width, m.Layout)
		b.WriteString(table)
	}

	// Render footer
	b.WriteString("\n")
	footer := RenderFooter(m.TotalLines, m.Width)
	b.WriteString(footer)

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// parseJsonFields parses JSON-escaped strings in configured fields
// Note: mutates data map in place
func parseJsonFields(data map[string]any, layout *Layout) map[string]any {
	// Build map of fields that should be parsed
	jsonFields := make(map[string]bool)
	for _, col := range layout.Columns {
		if col.Json {
			jsonFields[col.Field] = true
		}
	}

	// Loop over actual data fields
	for key, val := range data {
		// Skip if not configured for JSON parsing
		if !jsonFields[key] {
			continue
		}

		// Check if field is a string
		str, ok := val.(string)
		if !ok {
			continue
		}

		// Skip empty strings
		if str == "" {
			continue
		}

		// Try to parse as JSON
		var parsed any
		err := json.Unmarshal([]byte(str), &parsed)
		if err == nil {
			data[key] = parsed
		}
		// If parsing fails, keep original string value
	}

	return data
}
