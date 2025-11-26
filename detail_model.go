package parcours

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// DetailModel handles the detail/full record JSON view
type DetailModel struct {
	Store      Store
	Layout     *Layout
	TableModel *TableModel // Reference to navigate between records

	// Display state
	FullRecord map[string]any
	Width      int
	Height     int
	Focused    bool
}

type detailLoadMsg struct {
	data map[string]any
	err  error
}

func NewDetailModel(store Store, layout *Layout, tableModel *TableModel) *DetailModel {
	return &DetailModel{
		Store:      store,
		Layout:     layout,
		TableModel: tableModel,
	}
}

func (m *DetailModel) Init() tea.Cmd {
	return nil
}

func (m *DetailModel) loadRecord(id string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.Store.GetLine(id)
		return detailLoadMsg{data: data, err: err}
	}
}

func (m *DetailModel) Update(msg tea.Msg) (*DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case detailLoadMsg:
		if msg.err != nil {
			// Show error in JSON view
			m.FullRecord = map[string]any{"error": msg.err.Error()}
			return m, nil
		}
		// Parse JSON fields before storing
		m.FullRecord = parseJsonFields(msg.data, m.Layout)
		return m, nil

	case tea.KeyPressMsg:
		// Only handle keys when focused
		if !m.Focused {
			return m, nil
		}

		switch msg.String() {
		/*
			case "up", "k":
				// Navigate to previous record
				// Delegate navigation to table model, then fetch new record
				var cmd tea.Cmd
				m.TableModel, cmd = m.TableModel.Update(msg)

				id := m.TableModel.GetSelectedID()
				if id != "" {
					return m, tea.Sequence(cmd, m.loadRecord(id))
				}
				return m, cmd

			case "down", "j":
				// Navigate to next record
				var cmd tea.Cmd
				m.TableModel, cmd = m.TableModel.Update(msg)

				id := m.TableModel.GetSelectedID()
				if id != "" {
					return m, tea.Sequence(cmd, m.loadRecord(id))
				}
				return m, cmd
		*/
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, nil
}

func (m *DetailModel) View() string {
	var b strings.Builder

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

	return b.String()
}

// LoadCurrentRecord loads the currently selected record from the table
func (m *DetailModel) LoadCurrentRecord() tea.Cmd {
	id := m.TableModel.GetSelectedID()
	if id != "" {
		return m.loadRecord(id)
	}
	return nil
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
