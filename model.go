package parcours

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"parcours/detail"
	nt "parcours/entity"
	"parcours/filter"
	"parcours/message"
	"parcours/style"
	"parcours/table"
)

// Todo: why is pageup/down broken after running parcours?

const (
	layoutFile   = "layout.yaml"
	footerHeight = 1
)

type active int

const (
	tableActive active = iota
	detailActive
	filterActive
)

// Model is the bubbletea model for the log viewer TUI.
type Model struct {
	Store       Store
	logger      nt.Logger
	ctx         context.Context
	errorString string

	tablePanel  tea.Model
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

// NewModel creates a new bt model.
func NewModel(ctx context.Context, store Store, lgr nt.Logger) (model Model, err error) {

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

	tblPanel, err := table.NewTablePanel(ctx, layout.Columns, fields, count, lgr)
	if err != nil {
		return
	}

	model = Model{
		Store:       store,
		ctx:         ctx,
		logger:      lgr,
		tablePanel:  tblPanel,
		detailPanel: detail.NewDetailPanel(ctx, layout.Columns, lgr),
		filterPanel: filter.NewFilterPanel(ctx, lgr),
		active:      tableActive,
	}

	return
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	m.logger.Info(m.ctx, "received", "message", msg, "type", fmt.Sprintf("%T", msg))

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case table.TableMsg:
		m.tablePanel, cmd = m.tablePanel.Update(msg)
		return m, cmd

	case detail.DetailMsg:
		m.detailPanel, cmd = m.detailPanel.Update(msg)
		return m, cmd

	case filter.FilterMsg:
		m.filterPanel, cmd = m.filterPanel.Update(msg)
		return m, cmd

	case message.SetFilterMsg:
		// Apply the filter and reload data
		err := m.Store.SetView(msg.Filter, nil)
		if err != nil {
			m.errorString = err.Error()
			m.active = tableActive
			return m, nil
		}
		// Switch back to table and reset to reload with new filter
		m.active = tableActive
		return m, func() tea.Msg { return table.ResetMsg{} }

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
			if m.active != tableActive {
				m.active = tableActive
				return m, nil
			}
			return m, tea.Quit

		case "r":
			return m, m.reloadColumns()

		case "f":
			return m, m.reloadFilter()

		case "enter":
			if m.active == tableActive {
				m.active = detailActive
				return m, m.getLine(m.selectedId)
			}

		default:
			switch m.active {
			case tableActive:
				m.tablePanel, cmd = m.tablePanel.Update(msg)
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

		panelHeight := msg.Height - footerHeight

		var cmds []tea.Cmd
		m.tablePanel, cmd = m.tablePanel.Update(table.SizeMsg{
			Width:  msg.Width,
			Height: panelHeight,
		})
		cmds = append(cmds, cmd)

		m.detailPanel, cmd = m.detailPanel.Update(detail.SizeMsg{
			Width:  msg.Width,
			Height: panelHeight,
		})
		cmds = append(cmds, cmd)

		// Todo: use filter size, or lose
		m.filterPanel, cmd = m.filterPanel.Update(filter.SizeMsg{
			Width:  msg.Width,
			Height: panelHeight,
		})
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
	case tableActive:
		activeView = m.tablePanel.View()
	case detailActive:
		activeView = m.detailPanel.View()
	case filterActive:
		// Show filter dialog over table
		activeView = m.tablePanel.View()
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
