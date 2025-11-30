package parcours

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"parcours/message"
	"parcours/table"
)

// Todo: push store into table and detail derp

const (
	layoutFile   = "layout.yaml"
	footerHeight = 1
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
	Store       Store
	logger      Logger
	ctx         context.Context
	errorString string

	CurrentScreen Screen

	//Lines []nt.Line

	TablePanel  table.TablePanel
	DetailPanel DetailPanel

	initialized bool // Set to true after first WindowSizeMsg
	Width       int
	Height      int
}

// NewModel creates a new bt model.
func NewModel(ctx context.Context, store Store, lgr Logger) (model Model, err error) {

	layout, err := loadLayout("layout.yaml")
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
		logger:        lgr,
		CurrentScreen: TableScreen,
		TablePanel:    table.NewTablePanel(layout.Columns, fields, count),
		DetailPanel:   NewDetailPanel(layout.Columns),
	}

	return
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case table.TableMsg:
		m.TablePanel, cmd = m.TablePanel.Update(msg)
		return m, cmd

	case message.GetPageMsg:
		return m, m.getPage(msg.Offset, msg.Size)

	case message.ErrorMsg:
		m.logger.Error(m.ctx, "error msg", msg.Err)
		m.errorString = msg.Err.Error()
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
			return m, m.reloadColumns()

		case "f":
			// Reload filter from layout
			return m, m.reloadFilter()

		case "right", "l":
			if m.CurrentScreen == TableScreen {
				return m.switchToDetail()
			}

		case "left", "h":
			if m.CurrentScreen == DetailScreen {
				return m.switchToTable()
			}
		default:
			// unmatched keys to children
			m.TablePanel, cmd = m.TablePanel.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if !m.initialized {
			m.initialized = true
		}

		return m, func() tea.Msg {
			return table.SizeMsg{
				Width:  msg.Width,
				Height: msg.Height - footerHeight,
			}
		}

	}

	return m, nil
}

func (m Model) View() tea.View {
	if !m.initialized {
		return tea.NewView("Loading...")
	}

	var mainView tea.View
	switch m.CurrentScreen {
	case DetailScreen:
		mainView = tea.NewView(m.DetailPanel.Render())
	case TableScreen:
		mainView = m.TablePanel.View()
	default:
		mainView = tea.NewView("Unknown screen") // Todo: error plz
	}

	// Create footer content and layer positioned at bottom
	selectedLine := m.TablePanel.Selected + 1
	total := m.TablePanel.Total

	footerContent := RenderFooter(selectedLine, total, m.Store.Name(), m.Width)
	if m.errorString != "" {
		footerContent = m.errorString // Todo: find a home for error string
	}
	footerLayer := lipgloss.NewLayer("footer", footerContent).Y(m.Height - footerHeight)
	// end footer

	// Compose layers on canvas
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	canvas.Compose(mainView.Content)
	canvas.Compose(footerLayer)

	view := tea.NewView(canvas)
	view.AltScreen = true
	return view
}
