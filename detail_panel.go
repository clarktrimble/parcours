package parcours

import (
	"encoding/json"
	"maps"
	"strings"

	"github.com/pkg/errors"

	tea "charm.land/bubbletea/v2"
)

// Todo: honor width

// DetailPanel handles the detail/full record JSON view display state
type DetailPanel struct {
	columns []Column // For JSON field parsing

	line         map[string]any // The record data to display
	contentLines []string       // Rendered content split into lines (cached)

	// Display state
	Width        int
	Height       int
	Focused      bool
	ScrollOffset int // Line offset for scrolling content
}

func NewDetailPanel(columns []Column) DetailPanel {
	return DetailPanel{
		columns: columns,
	}
}

// SetColumns updates the columns configuration
func (m DetailPanel) SetColumns(columns []Column) DetailPanel {
	m.columns = columns
	// Re-render if we have data
	if m.line != nil {
		m.computeContentLines()
	}
	return m
}

// parseJsonFields parses JSON-escaped strings in configured fields
// Returns a new map with parsed fields
func parseJsonFields(data map[string]any, columns []Column) (map[string]any, error) {

	// Build map of fields that should be parsed
	jsonFields := make(map[string]bool)
	for _, col := range columns {
		if col.Json {
			jsonFields[col.Field] = true
		}
	}

	// Create result map as a copy
	result := make(map[string]any, len(data))
	maps.Copy(result, data)

	// Parse configured JSON fields
	for key, val := range result {
		if !jsonFields[key] {
			continue
		}

		// Field must be a string
		str, ok := val.(string)
		if !ok {
			return nil, errors.Errorf("field %q marked as JSON but is not a string", key)
		}

		// Skip empty strings
		if str == "" {
			continue
		}

		// Try to parse as JSON
		var parsed any
		err := json.Unmarshal([]byte(str), &parsed)
		if err == nil {
			result[key] = parsed
		}
		// If parsing fails, keep original string value
	}

	return result, nil
}

// computeContentLines renders the line data as JSON and splits into lines
func (m *DetailPanel) computeContentLines() {

	if m.line == nil {
		m.contentLines = nil
		// Todo: this is error?
		return
	}

	data, err := parseJsonFields(m.line, m.columns)
	if err != nil {
		m.contentLines = []string{"Error parsing JSON fields: " + err.Error()}
		return
	}

	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(data)
	if err != nil {
		m.contentLines = []string{"Error pretty-printing JSON: " + err.Error()}
		// Todo: can this be errorMsg?
		return
	}

	// Split into lines
	content := strings.TrimSuffix(buf.String(), "\n")
	m.contentLines = strings.Split(content, "\n")
}

func (m DetailPanel) Update(msg tea.Msg) (DetailPanel, tea.Cmd) {

	switch msg := msg.(type) {

	case lineMsg:
		m.line = msg.line
		m.computeContentLines()
		m.ScrollOffset = 0

	case tea.KeyPressMsg:
		if !m.Focused {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.ScrollOffset > 0 {
				m.ScrollOffset--
			}

		case "down", "j":
			// Only allow scrolling if content exceeds viewport
			if m.Height > 0 && len(m.contentLines) > m.Height {
				maxScroll := len(m.contentLines) - m.Height
				if m.ScrollOffset < maxScroll {
					m.ScrollOffset++
				}
			}
			// Todo: pageup/down
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.ScrollOffset = 0
		// Todo: better ScrollOffset and may need to recompute contentLines
	}

	return m, nil
}

// Render renders the detail view (pure - no state mutation)
func (m DetailPanel) Render() string {
	if m.contentLines == nil {
		return "Loading full record..."
	}

	// Show visible portion based on scroll offset and height
	visibleLines := m.contentLines[m.ScrollOffset:]
	if m.Height > 0 && len(visibleLines) > m.Height {
		visibleLines = visibleLines[:m.Height] // Todo: dont crash
	}

	return strings.Join(visibleLines, "\n")
}
