package parcours

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"parcours/detail"
	nt "parcours/entity"
	"parcours/filterpanel"
	"parcours/intake"
	"parcours/linepanel"
	"parcours/message"
	"parcours/style"
)

/*
  Needs attention:
  1. o key - open intake from linepanel
  2. r / f keys - reload columns/filter (model methods)
*/

// Unwrapper is implemented by wrapped messages for unified handling
type Unwrapper interface {
	Unwrap() tea.Msg
}

// ModelToo is an experimental model using stack-based routing
type ModelToo struct {
	Store  Store
	logger nt.Logger
	ctx    context.Context

	stack       []tea.Model  // focus stack - top receives messages
	errorString string
	filters     []nt.Filter // for recreating filter panel

	initialized bool
	Width       int
	Height      int
	total       int

	selectedRow int
	selectedId  string
}

// NewModelToo creates a new stack-based model
func NewModelToo(ctx context.Context, store Store, lgr nt.Logger) (model ModelToo, err error) {
	intakePanel, err := intake.New(ctx, lgr)
	if err != nil {
		return
	}

	model = ModelToo{
		Store:  store,
		ctx:    ctx,
		logger: lgr,
		stack:  []tea.Model{intakePanel},
	}

	return
}

// top returns the index of the top of the stack
func (m ModelToo) top() int {
	return len(m.stack) - 1
}

// push adds a panel to the stack
func (m *ModelToo) push(panel tea.Model) {
	m.stack = append(m.stack, panel)
}

// pop removes the top panel from the stack
func (m *ModelToo) pop() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
}

// loadFile loads a file and pushes linepanel onto stack
func (m *ModelToo) loadFile(path string) error {
	err := m.Store.Load(path, 0)
	if err != nil {
		return err
	}

	layout, err := loadLayout(layoutFile)
	if err != nil {
		return err
	}

	err = layout.promote(m.Store)
	if err != nil {
		return err
	}

	err = m.Store.SetView(layout.Filter, nil)
	if err != nil {
		return err
	}

	fields, count, err := m.Store.GetView()
	if err != nil {
		return err
	}

	linePanel := linepanel.New(m.ctx, layout.Columns, fields, count, m.logger)

	// Replace stack with just linepanel (discard intake)
	m.stack = []tea.Model{linePanel}

	return nil
}

func (m ModelToo) Init() tea.Cmd {
	return nil
}

func (m ModelToo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	top := m.top()

	switch msg := msg.(type) {

	// Broadcast to all panels in stack
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if !m.initialized {
			m.initialized = true
		}

		panelSize := tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - footerHeight}

		var cmds []tea.Cmd
		for i := range m.stack {
			m.stack[i], cmd = m.stack[i].Update(panelSize)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// Unwrap and send to top
	case Unwrapper:
		m.stack[top], cmd = m.stack[top].Update(msg.Unwrap())
		return m, cmd

	// Messages that model handles (mediates Store access)
	case message.GetPageMsg:
		return m, m.getPageToo(msg.Offset, msg.Size)

	case message.CountMsg:
		m.total = msg.Count
		return m, nil

	case message.SelectedMsg:
		m.selectedRow = msg.Row
		m.selectedId = msg.Id
		return m, nil

	case intake.FileSelectedMsg:
		err := m.loadFile(msg.Path)
		if err != nil {
			m.errorString = err.Error()
			return m, nil
		}
		// Send size to new linepanel
		panelSize := tea.WindowSizeMsg{Width: m.Width, Height: m.Height - footerHeight}
		m.stack[m.top()], cmd = m.stack[m.top()].Update(panelSize)
		return m, cmd

	case message.SetFilterMsg:
		err := m.Store.SetView(msg.Filter, nil)
		if err != nil {
			m.errorString = err.Error()
			m.pop() // back to line
			return m, nil
		}
		m.filters = msg.Filters // save for next time
		m.pop()                 // back to line
		return m, func() tea.Msg { return linepanel.ResetMsg{} }

	case message.OpenFilterMsg:
		filterPanel := filterpanel.New(m.ctx, m.logger, m.filters)
		m.push(filterPanel)
		// Send size then the OpenFilterMsg to set up the filter
		panelSize := tea.WindowSizeMsg{Width: m.Width, Height: m.Height - footerHeight}
		m.stack[m.top()], _ = m.stack[m.top()].Update(panelSize)
		m.stack[m.top()], cmd = m.stack[m.top()].Update(msg)
		return m, cmd

	case linepanel.OpenDetailMsg:
		// Push detail panel and get line data
		detailPanel := detail.NewDetailPanel(m.ctx, msg.Columns, m.logger)
		m.push(detailPanel)
		// Send size to new panel
		panelSize := tea.WindowSizeMsg{Width: m.Width, Height: m.Height - footerHeight}
		m.stack[m.top()], _ = m.stack[m.top()].Update(panelSize)
		return m, m.getLineToo(msg.Id)

	case detail.DismissedMsg:
		m.pop()
		return m, nil

	case filterpanel.CanceledMsg:
		m.pop()
		return m, nil

	case linepanel.DismissedMsg:
		return m, tea.Quit

	case intake.DismissedMsg:
		if len(m.stack) == 1 {
			return m, tea.Quit
		}
		m.pop()
		return m, nil

	case error:
		m.errorString = msg.Error()
		return m, nil

	// Global keys
	case tea.KeyPressMsg:
		if m.errorString != "" {
			m.errorString = ""
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		// Everything else to top of stack
		m.stack[top], cmd = m.stack[top].Update(msg)
		return m, cmd

	// Everything else to top of stack
	default:
		m.stack[top], cmd = m.stack[top].Update(msg)
		return m, cmd
	}
}

func (m ModelToo) View() tea.View {
	if !m.initialized {
		return tea.NewView("Loading...")
	}

	// Render all panels in stack (bottom to top)
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	for _, panel := range m.stack {
		canvas.Compose(panel.View().Content)
	}

	// Footer
	footerContent := RenderFooter(m.selectedRow, m.total, m.Store.Name(), m.Width)
	if m.errorString != "" {
		footerContent = m.errorString
	}
	footerLayer := lipgloss.NewLayer("footer", footerContent).Y(m.Height - footerHeight)
	canvas.Compose(footerLayer)

	view := tea.NewView(canvas)
	view.BackgroundColor = style.BackgroundColor
	view.AltScreen = true
	return view
}

// getPageToo fetches a page of lines from the store
func (m ModelToo) getPageToo(offset, size int) tea.Cmd {
	return func() tea.Msg {
		_, count, err := m.Store.GetView()
		if err != nil {
			return err
		}

		linesData, err := m.Store.GetPage(offset, size)
		if err != nil {
			return err
		}

		return linepanel.PageMsg{
			Lines: linesData,
			Count: count,
		}
	}
}

// getLineToo fetches a full line from the store
func (m ModelToo) getLineToo(id string) tea.Cmd {
	return func() tea.Msg {
		line, err := m.Store.GetLine(id)
		if err != nil {
			return err
		}
		return detail.LineMsg{Line: line}
	}
}
