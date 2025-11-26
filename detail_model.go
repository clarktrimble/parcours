package parcours

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// DetailPane handles the detail/full record JSON view display state
type DetailPane struct {
	// Display state only
	Width        int
	Height       int
	Focused      bool
	ScrollOffset int // Line offset for scrolling content
}

func NewDetailPane() *DetailPane {
	return &DetailPane{}
}

func (m *DetailPane) Update(msg tea.Msg) (*DetailPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Only handle keys when focused
		if !m.Focused {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			// Scroll content up
			if m.ScrollOffset > 0 {
				m.ScrollOffset--
			}

		case "down", "j":
			// Scroll content down
			m.ScrollOffset++
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Reset scroll when window resizes
		m.ScrollOffset = 0
	}

	return m, nil
}

// Render renders the detail view with the given record data
func (m *DetailPane) Render(fullRecord map[string]any) string {
	if fullRecord == nil {
		return "Loading full record..."
	}

	// Pretty-print JSON with HTML escaping disabled
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	err := encoder.Encode(fullRecord)
	if err != nil {
		return "Error pretty-printing JSON: " + err.Error()
	}

	// Split into lines for scrolling
	content := strings.TrimSuffix(buf.String(), "\n")
	lines := strings.Split(content, "\n")

	// Apply scroll offset
	if m.ScrollOffset >= len(lines) {
		m.ScrollOffset = len(lines) - 1
		if m.ScrollOffset < 0 {
			m.ScrollOffset = 0
		}
	}

	// Show visible portion based on height
	visibleLines := lines[m.ScrollOffset:]
	if m.Height > 0 && len(visibleLines) > m.Height {
		visibleLines = visibleLines[:m.Height]
	}

	return strings.Join(visibleLines, "\n")
}
