package parcours

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	footerHeight = 2
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
	Store  Store
	Layout *Layout
	logger Logger

	// Current screen
	CurrentScreen Screen

	// Data (loaded from Store)
	Fields     []Field
	Lines      []Line
	TotalLines int
	FullRecord map[string]any

	// Child panes (display state only)
	TablePane  *TablePane
	DetailPane *DetailPane

	// Terminal dimensions
	Width  int
	Height int
}

// NewModel creates a new TUI model with the given store.
func NewModel(store Store, lgr Logger) (model Model, err error) {

	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		return
	}

	// Promote fields from layout
	// TODO: improve error handling/logging so we can see promotion failures
	for _, col := range layout.Columns {

		if col.Demote {
			continue
		}
		// Todo: fix to allow in impl
		if col.Field == "timestamp" || col.Field == "message" {
			continue
		}

		err = store.Promote(col.Field)
		if err != nil {
			return
		}
	}

	model = Model{
		Store:         store,
		Layout:        layout,
		logger:        lgr,
		CurrentScreen: TableScreen,
		TablePane:     NewTablePane(),
		DetailPane:    NewDetailPane(),
	}

	return
}

func (m Model) Init() tea.Cmd {
	return nil
}

// getPage loads a page of data from the store
func (m Model) getPage(offset, size int) tea.Cmd {
	return func() tea.Msg {
		fields, count, err := m.Store.GetView()
		if err != nil {
			return pageMsg{err: err}
		}

		lines, err := m.Store.GetPage(offset, size)
		if err != nil {
			return pageMsg{err: err}
		}

		return pageMsg{
			fields: fields,
			lines:  lines,
			count:  count,
		}
	}
}

// getLine loads a full record from the store
func (m Model) getLine(id string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.Store.GetLine(id)
		if err != nil {
			return lineMsg{err: err}
		}

		parsed := parseJsonFields(data, m.Layout)
		return lineMsg{data: parsed}
	}
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

// switchToTable switches to the table screen and manages focus
func (m *Model) switchToTable() {
	m.CurrentScreen = TableScreen
	m.TablePane.Focused = true
	m.DetailPane.Focused = false
}

// switchToDetail switches to the detail screen and manages focus
func (m *Model) switchToDetail() {
	m.CurrentScreen = DetailScreen
	m.TablePane.Focused = false
	m.DetailPane.Focused = true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case pageMsg:
		if msg.err != nil {
			// TODO: handle error
			return m, nil
		}
		m.Fields = msg.fields
		m.Lines = msg.lines
		m.TotalLines = msg.count
		// Update TablePane's line counts
		m.TablePane.CurrentLines = len(msg.lines)
		m.TablePane.TotalLines = msg.count
		return m, nil

	case lineMsg:
		if msg.err != nil {
			// TODO: handle error - maybe show error in detail view
			m.FullRecord = map[string]any{"error": msg.err.Error()}
			return m, nil
		}
		m.FullRecord = msg.data
		return m, nil

	case getPageMsg:
		// TablePane scrolled and needs new data
		return m, m.getPage(msg.Offset, msg.Size)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			if m.CurrentScreen != TableScreen {
				// Esc from detail goes back to table
				m.switchToTable()
				return m, nil
			}
			// Esc from table quits
			return m, tea.Quit

		case "right", "l":
			// Navigate right: table → detail
			if m.CurrentScreen == TableScreen {
				m.switchToDetail()
				// Reset detail scroll when entering
				m.DetailPane.ScrollOffset = 0
				// Load detail for currently selected row
				id := m.TablePane.GetSelectedID(m.Lines)
				if id != "" {
					return m, m.getLine(id)
				}
				return m, nil
			}

		case "left", "h":
			// Navigate left: detail → table
			if m.CurrentScreen == DetailScreen {
				m.switchToTable()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		adjustedMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - footerHeight,
		}
		var cmd1, cmd2 tea.Cmd
		m.TablePane, cmd1 = m.TablePane.Update(adjustedMsg)
		m.DetailPane, cmd2 = m.DetailPane.Update(adjustedMsg)

		return m, tea.Sequence(cmd1, cmd2)
	}

	// Broadcast to all child components
	var cmd1, cmd2 tea.Cmd
	m.TablePane, cmd1 = m.TablePane.Update(msg)
	m.DetailPane, cmd2 = m.DetailPane.Update(msg)
	return m, tea.Sequence(cmd1, cmd2)
}

func (m Model) View() tea.View {
	if m.Width == 0 {
		return tea.NewView("Loading...")
	}

	// Get current screen's content from child panes (pass data to them)
	var screenContent string
	switch m.CurrentScreen {
	case DetailScreen:
		screenContent = m.DetailPane.Render(m.FullRecord)
	case TableScreen:
		screenContent = m.TablePane.Render(m.Fields, m.Lines, m.Layout)
	default:
		screenContent = "Unknown screen"
	}

	// Create screen layer at origin (0, 0)
	screenLayer := lipgloss.NewLayer("screen", screenContent)

	// Create footer content and layer positioned at bottom
	current := m.TablePane.ScrollOffset + m.TablePane.SelectedRow + 1
	total := m.TablePane.TotalLines
	footerContent := RenderFooter(current, total, m.Store.Name(), m.Width)
	footerLayer := lipgloss.NewLayer("footer", footerContent).Y(m.Height - footerHeight)

	// Compose layers on canvas
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	canvas.Compose(screenLayer)
	canvas.Compose(footerLayer)

	v := tea.NewView(canvas)
	v.AltScreen = true
	return v
}
