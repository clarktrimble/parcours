package parcours

// Todo: deal with blank line at bottom of app

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	footerHeight = 2
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
	Store       Store
	Layout      *Layout // Todo: obviate?
	logger      Logger
	ctx         context.Context
	errorString string

	CurrentScreen Screen

	Lines []Line

	TablePanel  TablePanel
	DetailPanel DetailPanel

	Width  int
	Height int
}

// NewModel creates a new bt model.
func NewModel(ctx context.Context, store Store, lgr Logger) (model Model, err error) {

	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		return
	}

	// Promote fields from layout
	err = layout.promote(store)
	if err != nil {
		return
	}

	// Apply filter from layout (SetView handles nil)
	err = store.SetView(layout.Filter, nil)
	if err != nil {
		return
	}

	// Get fields from store
	fields, count, err := store.GetView()
	if err != nil {
		return
	}

	model = Model{
		Store:         store,
		Layout:        layout,
		logger:        lgr,
		CurrentScreen: TableScreen,
		TablePanel:    NewTablePanel(layout.Columns, fields, count),
		DetailPanel:   NewDetailPanel(layout.Columns),
	}

	return
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case pageMsg:
		m.Lines = msg.lines
		m.TablePanel.TotalLines = msg.count
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
				return m.switchToTable()
			}
			return m, tea.Quit

		case "r":
			// Reload columns from layout
			return m.reloadColumns()

		case "f":
			// Reload filter from layout
			return m.reloadFilter()

		case "right", "l":
			if m.CurrentScreen == TableScreen {
				return m.switchToDetail()
			}

		case "left", "h":
			if m.CurrentScreen == DetailScreen {
				return m.switchToTable()
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
		m.TablePanel, cmd1 = m.TablePanel.Update(adjustedMsg)
		m.DetailPanel, cmd2 = m.DetailPanel.Update(adjustedMsg)

		return m, tea.Sequence(cmd1, cmd2)
	}

	// Broadcast to all child components
	// Todo: icanhaz slice of interface?
	var cmd1, cmd2 tea.Cmd
	m.TablePanel, cmd1 = m.TablePanel.Update(msg)
	m.DetailPanel, cmd2 = m.DetailPanel.Update(msg)
	return m, tea.Sequence(cmd1, cmd2)
}

func (m Model) View() tea.View {
	if m.Width == 0 { // Todo: use m.intialized
		return tea.NewView("Loading...")
	}

	// Get current screen's content from child panes
	var screenContent string
	switch m.CurrentScreen {
	case DetailScreen:
		screenContent = m.DetailPanel.Render()
	case TableScreen:
		screenContent = m.TablePanel.Render(m.Lines)
	default:
		screenContent = "Unknown screen" // Todo: error plz
	}

	// Create screen layer at origin (0, 0)
	screenLayer := lipgloss.NewLayer("screen", screenContent)

	// Create footer content and layer positioned at bottom
	current := m.TablePanel.SelectedLine + 1
	total := m.TablePanel.TotalLines
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
