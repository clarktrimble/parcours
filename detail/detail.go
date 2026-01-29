package detail

import (
	"context"
	"encoding/json"
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/pkg/errors"

	nt "parcours/entity"
	"parcours/message"
)

// Todo: honor width
// Todo: show key/val rather than json view
// Todo: collapsable val
// Todo: ts, level, info at top

// DetailPanel handles the detail/full record JSON view display state
type DetailPanel struct {
	columns []nt.Column // For JSON field parsing

	line         map[string]any // The record data to display
	contentLines []string       // Rendered content split into lines

	width        int
	height       int
	scrollOffset int // Line offset for scrolling content

	ctx    context.Context
	logger nt.Logger
}

func New(ctx context.Context, columns []nt.Column, lgr nt.Logger, size tea.WindowSizeMsg) DetailPanel {
	return DetailPanel{
		columns: columns,
		ctx:     ctx,
		logger:  lgr,
		width:   size.Width,
		height:  size.Height,
	}
}

func (pnl DetailPanel) Init() tea.Cmd {
	return nil
}

func (pnl DetailPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case LineMsg:
		pnl.line = msg.Line
		pnl.computeContentLines()
		pnl.scrollOffset = 0

	case tea.WindowSizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height
		pnl.scrollOffset = 0
		// Todo: better ScrollOffset and may need to recompute contentLines

	case ColumnsMsg:
		pnl.columns = msg.Columns
		// Re-render if we have data
		if pnl.line != nil {
			pnl.computeContentLines()
		}

	case tea.KeyPressMsg:

		switch msg.String() {
		case "esc":
			return pnl, func() tea.Msg { return message.CloseMsg{} }

		case "up", "k":
			if pnl.scrollOffset > 0 {
				pnl.scrollOffset--
			}

		case "down", "j":
			// Only allow scrolling if content exceeds viewport
			if pnl.height > 0 && len(pnl.contentLines) > pnl.height {
				maxScroll := len(pnl.contentLines) - pnl.height
				if pnl.scrollOffset < maxScroll {
					pnl.scrollOffset++
				}
			}
			// Todo: pageup/down
		}
	}

	return pnl, nil
}

// View renders the detail view
func (pnl DetailPanel) View() tea.View {
	if pnl.contentLines == nil {
		return tea.NewView("Loading full record...")
	}

	// Show visible portion based on scroll offset and height
	visibleLines := pnl.contentLines[pnl.scrollOffset:]
	if pnl.height > 0 && len(visibleLines) > pnl.height {
		visibleLines = visibleLines[:pnl.height]
	}

	return tea.NewView(strings.Join(visibleLines, "\n"))
}

// unexported

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

// repairTruncatedJSON attempts to make truncated JSON valid by closing
// unclosed strings, arrays, and objects.
func repairTruncatedJSON(s string) string {
	const marker = "--truncated--"
	if !strings.HasSuffix(s, marker) {
		return s
	}

	// Trim the marker
	s = strings.TrimSuffix(s, marker)

	// Track state as we scan
	var stack []rune // '{' or '['
	inString := false
	escaped := false

	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}

		if r == '\\' && inString {
			escaped = true
			continue
		}

		if r == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch r {
		case '{', '[':
			stack = append(stack, r)
		case '}':
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				stack = stack[:len(stack)-1]
			}
		case ']':
			if len(stack) > 0 && stack[len(stack)-1] == '[' {
				stack = stack[:len(stack)-1]
			}
		}
	}

	// Build repair suffix
	var suffix strings.Builder

	// Close string if we're in one
	if inString {
		suffix.WriteRune('"')
	}

	// Close brackets/braces in reverse order
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == '{' {
			suffix.WriteRune('}')
		} else {
			suffix.WriteRune(']')
		}
	}

	return s + suffix.String()
}

// parseJsonFields parses JSON-escaped strings in configured fields
// Returns a new map with parsed fields
func parseJsonFields(data map[string]any, columns []nt.Column) (map[string]any, error) {

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

		// Try to parse as JSON, repairing if truncated
		str = repairTruncatedJSON(str)
		var parsed any
		err := json.Unmarshal([]byte(str), &parsed)
		if err == nil {
			result[key] = parsed
		}
		// If parsing fails, keep original string value
	}

	return result, nil
}
