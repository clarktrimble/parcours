package parcours

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"parcours/board"
	"parcours/detail"
	nt "parcours/entity"
	"parcours/filterpanel"
	"parcours/intake"
	"parcours/linepanel"
	"parcours/message"
	"parcours/style"
)

// Todo: why is pageup/down broken after running parcours?

const (
	layoutFile   = "layout.yaml"
	footerHeight = 1
)

type active int

const (
	intakeActive active = iota
	lineActive
	detailActive
	filterActive
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
	Store       Store
	logger      nt.Logger
	ctx         context.Context
	errorString string

	intakePanel tea.Model
	linePanel   tea.Model
	detailPanel tea.Model
	filterPanel tea.Model
	active      active

	initialized bool
	Width       int
	Height      int
	total       int

	selectedRow int
	selectedId  string
}

// NewModel creates a new bt model starting with the intake panel.
func NewModel(ctx context.Context, store Store, lgr nt.Logger) (model Model, err error) {

	intakePanel, err := intake.New(ctx, lgr)
	if err != nil {
		return
	}

	model = Model{
		Store:       store,
		ctx:         ctx,
		logger:      lgr,
		intakePanel: intakePanel,
		linePanel:   board.NewPlaceholder("No file loaded"),
		detailPanel: board.NewPlaceholder("No file loaded"),
		filterPanel: filterpanel.New(ctx, lgr),
		active:      intakeActive,
	}

	return
}

// loadFile loads a file and initializes the data panels.
func (m *Model) loadFile(path string) error {
	err := m.Store.Load(path, 0)
	if err != nil {
		return err
	}

	layout, err := loadLayout(layoutFile)
	if err != nil {
		return err
	}

	// Promote fields from layout
	err = layout.promote(m.Store)
	if err != nil {
		return err
	}

	// Apply filter from layout (SetView handles nil)
	err = m.Store.SetView(layout.Filter, nil)
	if err != nil {
		return err
	}

	// Get fields from store
	fields, count, err := m.Store.GetView()
	if err != nil {
		return err
	}

	m.linePanel = linepanel.New(m.ctx, layout.Columns, fields, count, m.logger)
	m.detailPanel = detail.NewDetailPanel(m.ctx, layout.Columns, m.logger)

	return nil
}

func (m Model) Init() tea.Cmd {
	// Todo: call children for good form
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	m.logger.Info(m.ctx, "received", "message", msg, "type", fmt.Sprintf("%T", msg))

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case linepanel.PageMsg, linepanel.ColumnsMsg, linepanel.ResetMsg:
		m.linePanel, cmd = m.linePanel.Update(msg)
		return m, cmd

	case intake.FileSelectedMsg:
		// Load the selected file and switch to table view
		err := m.loadFile(msg.Path)
		if err != nil {
			m.errorString = err.Error()
			return m, nil
		}
		m.active = lineActive
		// Send size to newly created panels - must return cmd from tablePanel
		// as it triggers the initial data fetch
		panelSize := tea.WindowSizeMsg{Width: m.Width, Height: m.Height - footerHeight}
		m.linePanel, cmd = m.linePanel.Update(panelSize)
		m.detailPanel, _ = m.detailPanel.Update(panelSize)
		return m, cmd

	case intake.Msg:
		m.intakePanel, cmd = m.intakePanel.Update(msg.Wrapped)
		return m, cmd

	case linepanel.Msg:
		m.linePanel, cmd = m.linePanel.Update(msg.Wrapped)
		return m, cmd

	case detail.LineMsg, detail.ColumnsMsg:
		m.detailPanel, cmd = m.detailPanel.Update(msg)
		return m, cmd

	case filterpanel.Msg:
		m.filterPanel, cmd = m.filterPanel.Update(msg.Wrapped)
		return m, cmd

	case message.SetFilterMsg:
		// Apply the filter and reload data
		err := m.Store.SetView(msg.Filter, nil)
		if err != nil {
			m.errorString = err.Error()
			m.active = lineActive
			return m, nil
		}
		// Switch back to table and reset to reload with new filter
		m.active = lineActive
		return m, func() tea.Msg { return linepanel.ResetMsg{} }

	case message.OpenFilterMsg:
		// Open filter dialog with cell data
		m.active = filterActive
		m.filterPanel, cmd = m.filterPanel.Update(msg)
		return m, cmd

	case message.GetPageMsg:
		return m, m.getPage(msg.Offset, msg.Size)

	case message.CountMsg:
		m.total = msg.Count
		return m, nil

	case message.SelectedMsg:
		m.selectedRow = msg.Row
		m.selectedId = msg.Id
		return m, nil

	case error:
		m.logger.Error(m.ctx, "error", msg)
		m.errorString = msg.Error()
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
			switch m.active {
			case intakeActive:
				// Only quit from intake if a file is loaded
				if m.Store.Name() != "" {
					m.active = lineActive
					return m, nil
				}
				return m, tea.Quit
			case lineActive:
				return m, tea.Quit
			default:
				m.active = lineActive
				return m, nil
			}

		case "o":
			// Open intake panel
			if m.active != intakeActive {
				m.active = intakeActive
				return m, nil
			}

		case "r":
			if m.active == lineActive {
				return m, m.reloadColumns()
			}

		case "f":
			if m.active == lineActive {
				return m, m.reloadFilter()
			}

		case "enter":
			if m.active == lineActive {
				m.active = detailActive
				return m, m.getLine(m.selectedId)
			}
			fallthrough // Todo: restructure properly, maybe?

		default:
			switch m.active {
			case intakeActive:
				m.intakePanel, cmd = m.intakePanel.Update(msg)
			case lineActive:
				m.linePanel, cmd = m.linePanel.Update(msg)
			case detailActive:
				m.detailPanel, cmd = m.detailPanel.Update(msg)
			case filterActive:
				m.filterPanel, cmd = m.filterPanel.Update(msg)
			}
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if !m.initialized {
			m.initialized = true
		}

		panelSize := tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - footerHeight}

		var cmds []tea.Cmd
		m.intakePanel, cmd = m.intakePanel.Update(panelSize)
		cmds = append(cmds, cmd)

		m.linePanel, cmd = m.linePanel.Update(panelSize)
		cmds = append(cmds, cmd)

		m.detailPanel, cmd = m.detailPanel.Update(panelSize)
		cmds = append(cmds, cmd)

		m.filterPanel, cmd = m.filterPanel.Update(panelSize)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) View() tea.View {
	if !m.initialized {
		return tea.NewView("Loading...")
	}

	var activeView tea.View
	switch m.active {
	case intakeActive:
		activeView = m.intakePanel.View()
	case lineActive:
		activeView = m.linePanel.View()
	case detailActive:
		activeView = m.detailPanel.View()
	case filterActive:
		// Show filter dialog over table
		activeView = m.linePanel.View()
	}

	// Create footer content and layer positioned at bottom
	footerContent := RenderFooter(m.selectedRow, m.total, m.Store.Name(), m.Width)
	if m.errorString != "" {
		footerContent = m.errorString // Todo: find a home for error string
	}
	footerLayer := lipgloss.NewLayer("footer", footerContent).Y(m.Height - footerHeight)
	// end footer

	// Compose layers on canvas
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	canvas.Compose(activeView.Content)
	canvas.Compose(footerLayer)

	// Overlay filter dialog if active
	if m.active == filterActive {
		canvas.Compose(m.filterPanel.View().Content)
	}

	view := tea.NewView(canvas)
	view.BackgroundColor = style.BackgroundColor
	view.AltScreen = true
	return view
}
