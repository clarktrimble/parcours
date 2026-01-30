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

// jsonScanner tracks state while scanning truncated JSON.
type jsonScanner struct {
	stack      []rune   // '{' or '['
	path       []string // keys as we descend into objects
	pathDepths []int    // stack depth when each path entry was added
	currentKey string
	keyBuf     strings.Builder

	inString   bool
	inKey      bool
	afterColon bool
	escaped    bool
}

// scan processes the JSON string character by character.
func (js *jsonScanner) scan(s string) {
	for _, r := range s {
		js.processRune(r)
	}
}

// processRune handles a single character.
func (js *jsonScanner) processRune(r rune) {
	if js.escaped {
		if js.inKey {
			js.keyBuf.WriteRune(r)
		}
		js.escaped = false
		return
	}

	if r == '\\' && js.inString {
		js.escaped = true
		if js.inKey {
			js.keyBuf.WriteRune(r)
		}
		return
	}

	if r == '"' {
		js.handleQuote()
		return
	}

	if js.inString {
		if js.inKey {
			js.keyBuf.WriteRune(r)
		}
		return
	}

	js.handleStructural(r)
}

// handleQuote processes opening and closing quotes.
func (js *jsonScanner) handleQuote() {
	if js.inString {
		if js.inKey {
			js.currentKey = js.keyBuf.String()
			js.keyBuf.Reset()
			js.inKey = false
		}
		js.inString = false
	} else {
		js.inString = true
		// Starting a key if in object context and not after colon
		if js.inObject() && !js.afterColon {
			js.inKey = true
			js.keyBuf.Reset()
		}
	}
}

// handleStructural processes brackets, braces, colons, and commas.
func (js *jsonScanner) handleStructural(r rune) {
	switch r {
	case '{':
		js.stack = append(js.stack, r)
		if js.currentKey != "" {
			js.path = append(js.path, js.currentKey)
			js.pathDepths = append(js.pathDepths, len(js.stack))
			js.currentKey = ""
		}
		js.afterColon = false
	case '[':
		js.stack = append(js.stack, r)
		js.afterColon = false
	case '}':
		if js.inObject() {
			if len(js.pathDepths) > 0 && js.pathDepths[len(js.pathDepths)-1] == len(js.stack) {
				js.path = js.path[:len(js.path)-1]
				js.pathDepths = js.pathDepths[:len(js.pathDepths)-1]
			}
			js.stack = js.stack[:len(js.stack)-1]
		}
		js.afterColon = false
	case ']':
		if js.inArray() {
			js.stack = js.stack[:len(js.stack)-1]
		}
		js.afterColon = false
	case ':':
		js.afterColon = true
	case ',':
		// Only reset currentKey inside objects, not arrays
		if js.inObject() {
			js.currentKey = ""
		}
		js.afterColon = false
	}
}

// inObject returns true if currently inside an object.
func (js *jsonScanner) inObject() bool {
	return len(js.stack) > 0 && js.stack[len(js.stack)-1] == '{'
}

// inArray returns true if currently inside an array.
func (js *jsonScanner) inArray() bool {
	return len(js.stack) > 0 && js.stack[len(js.stack)-1] == '['
}

// repairPartialPrimitive checks for and repairs truncated true/false/null.
// Returns the trimmed string and replacement value (if any).
func (js *jsonScanner) repairPartialPrimitive(s string) (string, string) {
	if js.inString {
		return s, ""
	}

	s = strings.TrimRight(s, " \t\n\r")
	partials := []string{"true", "tru", "tr", "false", "fals", "fal", "fa", "null", "nul", "nu"}

	for _, p := range partials {
		if strings.HasSuffix(s, p) {
			return strings.TrimSuffix(s, p), "null"
		}
	}

	return s, ""
}

// buildSuffix creates the string needed to close all open structures.
func (js *jsonScanner) buildSuffix(s string) string {
	var suffix strings.Builder

	if js.inString {
		suffix.WriteRune('"')
		if js.inKey {
			suffix.WriteString(": null")
		}
	} else {
		// Check for trailing comma
		trimmed := strings.TrimRight(s, " \t\n\r")
		if strings.HasSuffix(trimmed, ",") {
			if js.inObject() {
				suffix.WriteString(`"_": null`)
			} else {
				suffix.WriteString("null")
			}
		}
	}

	// Close brackets/braces in reverse order
	for i := len(js.stack) - 1; i >= 0; i-- {
		if js.stack[i] == '{' {
			suffix.WriteRune('}')
		} else {
			suffix.WriteRune(']')
		}
	}

	return suffix.String()
}

// buildPath assembles the path to where truncation occurred.
func (js *jsonScanner) buildPath() string {
	truncPath := strings.Join(js.path, ".")

	if js.currentKey != "" {
		if truncPath != "" {
			truncPath += "."
		}
		truncPath += js.currentKey
	} else if js.inKey {
		if truncPath != "" {
			truncPath += "."
		}
		truncPath += js.keyBuf.String()
	}

	return truncPath
}

// repairTruncatedJSON attempts to make truncated JSON valid by closing
// unclosed strings, arrays, and objects. Returns the repaired JSON and
// the path where truncation occurred (e.g. "payload.fields.tag_number").
func repairTruncatedJSON(s string) (string, string) {
	const marker = "--truncated--"
	if !strings.HasSuffix(s, marker) {
		return s, ""
	}

	s = strings.TrimSuffix(s, marker)

	js := &jsonScanner{}
	js.scan(s)

	s, primitive := js.repairPartialPrimitive(s)

	return s + primitive + js.buildSuffix(s), js.buildPath()
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
		repaired, truncPath := repairTruncatedJSON(str)
		var parsed any
		err := json.Unmarshal([]byte(repaired), &parsed)
		if err == nil {
			result[key] = parsed
			if truncPath != "" {
				result["_truncated_at"] = key + "." + truncPath
			}
		}
		// If parsing fails, keep original string value
	}

	return result, nil
}
