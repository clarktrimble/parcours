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
func (pnl DetailPanel) SetColumns(columns []Column) DetailPanel {
	pnl.columns = columns
	// Re-render if we have data
	if pnl.line != nil {
		pnl.computeContentLines()
	}
	return pnl
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
func (pnl *DetailPanel) computeContentLines() {

	if pnl.line == nil {
		pnl.contentLines = nil
		// Todo: this is error?
		return
	}

	data, err := parseJsonFields(pnl.line, pnl.columns)
	if err != nil {
		pnl.contentLines = []string{"Error parsing JSON fields: " + err.Error()}
		return
	}

	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(data)
	if err != nil {
		pnl.contentLines = []string{"Error pretty-printing JSON: " + err.Error()}
		// Todo: can this be errorMsg?
		return
	}

	// Split into lines
	content := strings.TrimSuffix(buf.String(), "\n")
	pnl.contentLines = strings.Split(content, "\n")
}

func (pnl DetailPanel) Update(msg tea.Msg) (DetailPanel, tea.Cmd) {

	switch msg := msg.(type) {

	case lineMsg:
		pnl.line = msg.line
		pnl.computeContentLines()
		pnl.ScrollOffset = 0

	case tea.KeyPressMsg:
		if !pnl.Focused {
			return pnl, nil
		}

		switch msg.String() {
		case "up", "k":
			if pnl.ScrollOffset > 0 {
				pnl.ScrollOffset--
			}

		case "down", "j":
			// Only allow scrolling if content exceeds viewport
			if pnl.Height > 0 && len(pnl.contentLines) > pnl.Height {
				maxScroll := len(pnl.contentLines) - pnl.Height
				if pnl.ScrollOffset < maxScroll {
					pnl.ScrollOffset++
				}
			}
			// Todo: pageup/down
		}

	case tea.WindowSizeMsg:
		pnl.Width = msg.Width
		pnl.Height = msg.Height
		pnl.ScrollOffset = 0
		// Todo: better ScrollOffset and may need to recompute contentLines
	}

	return pnl, nil
}

// Render renders the detail view (pure - no state mutation)
func (pnl DetailPanel) Render() string {
	if pnl.contentLines == nil {
		return "Loading full record..."
	}

	// Show visible portion based on scroll offset and height
	visibleLines := pnl.contentLines[pnl.ScrollOffset:]
	if pnl.Height > 0 && len(visibleLines) > pnl.Height {
		visibleLines = visibleLines[:pnl.Height] // Todo: dont crash
	}

	return strings.Join(visibleLines, "\n")
}
