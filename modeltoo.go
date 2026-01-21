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
const (
	layoutFile   = "layout.yaml"
	footerHeight = 1
)

// ModelToo is the model using stack-based routing
type ModelToo struct {
	Store  Store
	logger nt.Logger
	ctx    context.Context

	stack       []tea.Model // focus stack - top receives messages
	errorString string
	filters     []nt.Filter // for recreating filter panel

	initialized bool
	Width       int
	Height      int
	panelSize   tea.WindowSizeMsg
	total       int

	selectedRow int
	selectedId  string
}

// NewModelToo creates a new stack-based model
func NewModelToo(ctx context.Context, store Store, lgr nt.Logger) (model ModelToo, err error) {
	intakePanel, err := intake.New(ctx, lgr, tea.WindowSizeMsg{})
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

// withError sets the error string and returns the model
func (m ModelToo) withError(err error) (tea.Model, tea.Cmd) {
	m.errorString = err.Error()
	return m, nil
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
	key, ok := msg.(tea.KeyPressMsg)
	if ok {
		if m.errorString != "" {
			m.errorString = ""
		}
		if key.String() == "ctrl+c" || key.String() == "q" {
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {

	// Lifecycle
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.panelSize = tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - footerHeight}
		m.initialized = true

		var cmds []tea.Cmd
		for i := range m.stack {
			m.stack[i], cmd = m.stack[i].Update(m.panelSize)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// Todo: loadFile is broken in Store; when fixed, consider having loadFile
	// accept panelSize and set it on the linepanel it creates (like New funcs)
	case message.FileSelectedMsg:
		err := m.loadFile(msg.Path)
		if err != nil {
			return m.withError(err)
		}
		m.stack[m.top()], cmd = m.stack[m.top()].Update(m.panelSize)
		return m, cmd

	// Todo: think thru error handling moar
	case error:
		return m.withError(msg)

	// Push
	case message.OpenFilterMsg:
		panel, err := filterpanel.New(m.ctx, m.logger, m.filters, msg.Field, msg.Value, m.panelSize)
		if err != nil {
			return m.withError(err)
		}
		m.push(panel)
		return m, nil

	case message.OpenDetailMsg:
		m.push(detail.New(m.ctx, msg.Columns, m.logger, m.panelSize))
		return m, m.getLineToo(msg.Id)

	case message.OpenIntakeMsg:
		intakePanel, err := intake.New(m.ctx, m.logger, m.panelSize)
		if err != nil {
			return m.withError(err)
		}
		m.push(intakePanel)
		return m, nil

	// Pop
	case message.SetFilterMsg:
		m.pop()
		err := m.Store.SetView(msg.Filter, nil)
		if err != nil {
			return m.withError(err)
		}
		m.filters = msg.Filters
		return m, func() tea.Msg { return linepanel.ResetMsg{} }

	case message.CloseMsg:
		if len(m.stack) == 1 {
			return m, tea.Quit
		}
		m.pop()
		return m, nil

	// State
	case message.CountMsg:
		m.total = msg.Count
		return m, nil

	case message.SelectedMsg:
		m.selectedRow = msg.Row
		m.selectedId = msg.Id
		return m, nil

	// Data
	case message.GetPageMsg:
		return m, m.getPageToo(msg.Offset, msg.Size)

	case message.ReloadColumnsMsg:
		return m, m.reloadColumnsToo()

	default:
		m.stack[m.top()], cmd = m.stack[m.top()].Update(msg)
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

// reloadColumnsToo loads layout from file and sends column updates
func (m ModelToo) reloadColumnsToo() tea.Cmd {
	layout, err := loadLayout(layoutFile)
	if err != nil {
		return func() tea.Msg { return err }
	}

	err = layout.promote(m.Store)
	if err != nil {
		return func() tea.Msg { return err }
	}

	fields, _, err := m.Store.GetView()
	if err != nil {
		return func() tea.Msg { return err }
	}

	return func() tea.Msg {
		return linepanel.ColumnsMsg{Columns: layout.Columns, Fields: fields}
	}
}
