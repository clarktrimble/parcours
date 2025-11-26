package parcours

import (
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

	// Current screen
	CurrentScreen Screen

	// Models
	TableModel  *TableModel
	DetailModel *DetailModel

	// Terminal dimensions
	Width  int
	Height int
}

// NewModel creates a new TUI model with the given store.
func NewModel(store Store) Model {
	layout, err := LoadLayout("layout.yaml")
	if err != nil {
		panic(err) // TODO: handle better
	}

	// Promote fields from layout
	// TODO: improve error handling/logging so we can see promotion failures
	for _, col := range layout.Columns {
		// Skip demoted fields
		if col.Demote {
			continue
		}
		// Skip base fields that already exist
		if col.Field == "timestamp" || col.Field == "message" {
			continue
		}
		if err := store.Promote(col.Field); err != nil {
			// TODO: log error instead of panicking
			panic(err)
		}
	}

	tableModel := NewTableModel(store, layout)
	tableModel.Focused = true // Start with table focused

	detailModel := NewDetailModel(store, layout, tableModel)

	return Model{
		Store:         store,
		Layout:        layout,
		CurrentScreen: TableScreen,
		TableModel:    tableModel,
		DetailModel:   detailModel,
	}
}

func (m Model) Init() tea.Cmd {
	return m.TableModel.Init()
}

// switchToTable switches to the table screen and manages focus
func (m *Model) switchToTable() {
	m.CurrentScreen = TableScreen
	m.TableModel.Focused = true
	m.DetailModel.Focused = false
}

// switchToDetail switches to the detail screen and manages focus
func (m *Model) switchToDetail() {
	m.CurrentScreen = DetailScreen
	m.TableModel.Focused = false
	m.DetailModel.Focused = true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
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
				return m, m.DetailModel.LoadCurrentRecord()
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
		m.TableModel, cmd1 = m.TableModel.Update(adjustedMsg)
		m.DetailModel, cmd2 = m.DetailModel.Update(adjustedMsg)

		return m, tea.Sequence(cmd1, cmd2)
	}

	var cmd1, cmd2 tea.Cmd
	m.TableModel, cmd1 = m.TableModel.Update(msg)
	m.DetailModel, cmd2 = m.DetailModel.Update(msg)
	return m, tea.Sequence(cmd1, cmd2)
}

func (m Model) View() tea.View {
	if m.Width == 0 {
		return tea.NewView("Loading...")
	}

	// Get current screen's content from child models
	var screenContent string
	switch m.CurrentScreen {
	case DetailScreen:
		screenContent = m.DetailModel.View()
	case TableScreen:
		screenContent = m.TableModel.View()
	default:
		screenContent = "Unknown screen"
	}

	// Create screen layer at origin (0, 0)
	screenLayer := lipgloss.NewLayer("screen", screenContent)

	// Create footer content and layer positioned at bottom
	current := m.TableModel.ScrollOffset + m.TableModel.SelectedRow + 1
	total := m.TableModel.TotalLines
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
