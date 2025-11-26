package parcours

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// DetailPane handles the detail/full record JSON view display state
type DetailPane struct {
	// Display state only
	Width   int
	Height  int
	Focused bool
}

func NewDetailPane() *DetailPane {
	return &DetailPane{}
}

func (m *DetailPane) Update(msg tea.Msg) (*DetailPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, nil
}

// Render renders the detail view with the given record data
func (m *DetailPane) Render(fullRecord map[string]any) string {
	var b strings.Builder

	// Show full record JSON
	if fullRecord != nil {
		// Pretty-print JSON with HTML escaping disabled
		var buf strings.Builder
		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)

		err := encoder.Encode(fullRecord)
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
