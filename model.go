package parcours

import (
	"context"
	"encoding/json"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	footerHeight = 2
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
	Store       Store
	Layout      *Layout
	logger      Logger
	ctx         context.Context
	errorString string

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
func NewModel(ctx context.Context, store Store, lgr Logger) (model Model, err error) {

	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		return
	}

	// Promote fields from layout
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

	// Apply filter from layout (SetView handles nil)
	err = store.SetView(layout.Filter, nil)
	if err != nil {
		return
	}

	model = Model{
		Store:         store,
		Layout:        layout,
		logger:        lgr,
		CurrentScreen: TableScreen,
		TablePane:     NewTablePane(layout),
		DetailPane:    NewDetailPane(),
	}

	return
}

func (m Model) Init() tea.Cmd {
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case pageMsg:
		m.Fields = msg.fields
		m.Lines = msg.lines
		m.TotalLines = msg.count
		m.TablePane.TotalLines = msg.count
		return m, nil

	case lineMsg:
		m.FullRecord = msg.data
		return m, nil

	case getPageMsg:
		// Todo: msg relay, can we make do with in or out?
		return m, m.getPage(msg.offset, msg.size)

	case errorMsg:
		m.logger.Error(m.ctx, "error msg", msg.err)
		m.errorString = msg.err.Error()
		//m = m.ready()
		//return m.refocus(alert)
		return m, nil

	case tea.KeyPressMsg:
		if m.errorString != "" {
			m.errorString = "" //Todo: find home for clear error
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			if m.CurrentScreen != TableScreen {
				m.switchToTable()
				return m, nil
			}
			return m, tea.Quit

		case "right", "l":
			if m.CurrentScreen == TableScreen {
				return m, m.switchToDetail()
			}

		case "left", "h":
			if m.CurrentScreen == DetailScreen {
				return m, m.switchToTable()
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		adjustedMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - footerHeight,
		}
		// Todo: just loop these thru again?? (just below)
		var cmd1, cmd2 tea.Cmd
		m.TablePane, cmd1 = m.TablePane.Update(adjustedMsg)
		m.DetailPane, cmd2 = m.DetailPane.Update(adjustedMsg)

		return m, tea.Sequence(cmd1, cmd2)
	}

	// Broadcast to all child components
	// Todo: icanhaz slice of interface?
	var cmd1, cmd2 tea.Cmd
	m.TablePane, cmd1 = m.TablePane.Update(msg)
	m.DetailPane, cmd2 = m.DetailPane.Update(msg)
	return m, tea.Sequence(cmd1, cmd2)
}

func (m Model) View() tea.View {
	if m.Width == 0 { // Todo: use m.intialized
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
		screenContent = "Unknown screen" // Todo: error plz
	}

	// Create screen layer at origin (0, 0)
	screenLayer := lipgloss.NewLayer("screen", screenContent)

	// Create footer content and layer positioned at bottom
	current := m.TablePane.SelectedLine + 1
	total := m.TablePane.TotalLines
	footerContent := RenderFooter(current, total, m.Store.Name(), m.Width)
	if m.errorString != "" {
		footerContent = m.errorString // Todo: find a home for error string
	}
	footerLayer := lipgloss.NewLayer("footer", footerContent).Y(m.Height - footerHeight)

	// Compose layers on canvas
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	canvas.Compose(screenLayer)
	canvas.Compose(footerLayer)

	view := tea.NewView(canvas)
	view.AltScreen = true
	return view
}
