package parcours

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"parcours/documentpanel"
	nt "parcours/entity"
	"parcours/filterpanel"
	"parcours/intake"
	"parcours/jsonpanel"
	"parcours/linepanel"
	"parcours/message"
	"parcours/style"
)

const (
	layoutFile   = "layout.yaml"
	footerHeight = 2
)

// Model is the root bt model.
type Model struct {
	Store    Store
	lastFile string
	logger   nt.Logger
	ctx      context.Context

	stack   []tea.Model
	hints   []message.Hint
	filters []nt.Filter
	errorString string

	initialized bool
	Width       int
	Height      int
	panelSize   tea.WindowSizeMsg
	total       int

	selectedRow int
	selectedId  string
}

// NewModel creates a model with intake panel.
func NewModel(ctx context.Context, store Store, lgr nt.Logger, lastFile string) (model tea.Model, err error) {

	intakePanel, err := intake.New(ctx, lgr, tea.WindowSizeMsg{}, lastFile)
	if err != nil {
		return
	}

	model = Model{
		Store:    store,
		lastFile: lastFile,
		ctx:      ctx,
		logger:   lgr,
		stack:    []tea.Model{intakePanel},
	}

	return
}

func (m Model) top() int {
	return len(m.stack) - 1
}

func (m Model) push(panel tea.Model) Model {
	m.stack = append(m.stack, panel)
	return m
}

func (m Model) pop() Model {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
	return m
}

// collectHints triggers hint collection from the stack.
func (m Model) collectHints() tea.Cmd {
	return func() tea.Msg {
		return message.ReportHintsMsg{}
	}
}

func (m Model) withError(err error) (tea.Model, tea.Cmd) {
	m.errorString = err.Error()
	return m, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

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

	case message.FileSelectedMsg:
		linePanel := linepanel.New(m.ctx, m.logger, m.panelSize)
		m.stack = []tea.Model{linePanel}
		return m, tea.Batch(linePanel.Init(), m.loadStore(msg.Path), m.collectHints())

	// Todo: loadFile is broken in Store; when fixed, consider having loadFile
	// accept panelSize and set it on the linepanel it creates (like New funcs)
	case message.FileLoadedMsg:
		m.lastFile = msg.Path
		return m, func() tea.Msg {
			return linepanel.ColumnsMsg{Columns: msg.Columns, Fields: msg.Fields}
		}

	case error:
		// Todo: think thru error handling moar
		return m.withError(msg)

	// Push
	case message.OpenFilterMsg:
		panel, err := filterpanel.New(m.ctx, m.logger, m.filters, msg.Field, msg.Value, m.panelSize)
		if err != nil {
			return m.withError(err)
		}
		return m.push(panel), m.collectHints()

	case message.OpenDetailMsg:
		m = m.push(documentpanel.New(m.ctx, msg.Columns, m.logger, m.panelSize))
		return m, tea.Batch(m.getLine(msg.Id), m.collectHints())

	case message.OpenJsonDetailMsg:
		m = m.push(jsonpanel.New(m.ctx, msg.Columns, m.logger, m.panelSize))
		return m, tea.Batch(
			func() tea.Msg { return jsonpanel.LineMsg{Line: msg.Line} },
			m.collectHints(),
		)

	case message.OpenIntakeMsg:
		intakePanel, err := intake.New(m.ctx, m.logger, m.panelSize, m.lastFile)
		if err != nil {
			return m.withError(err)
		}
		return m.push(intakePanel), m.collectHints()

	// Pop
	case message.SetFilterMsg:
		// close filter panel
		m = m.pop()

		// apply new filter
		err := m.Store.SetView(msg.Filter, nil)
		if err != nil {
			return m.withError(err)
		}
		m.filters = msg.Filters

		// reset line panel and collect hints
		return m, tea.Batch(
			func() tea.Msg { return linepanel.ResetMsg{} },
			m.collectHints(),
		)

	case message.CloseMsg:
		if len(m.stack) == 1 {
			return m, tea.Quit
		}
		m = m.pop()
		return m, m.collectHints()

	// State
	case message.CountMsg:
		m.total = msg.Count
		return m, nil

	case message.SelectedMsg:
		m.selectedRow = msg.Row
		m.selectedId = msg.Id
		return m, nil

	// Help
	case message.HintsChangedMsg:
		return m, m.collectHints()

	case message.ReportHintsMsg:
		m.hints = nil
		m.stack[m.top()], cmd = m.stack[m.top()].Update(msg)
		return m, cmd

	case message.HintsMsg:
		m.hints = append(m.hints, msg.Hints...)
		return m, nil

	// Data
	case message.GetPageMsg:
		return m, m.getPage(msg.Offset, msg.Size)

	case message.ReloadColumnsMsg:
		return m, m.reloadColumns()

	default:
		m.stack[m.top()], cmd = m.stack[m.top()].Update(msg)
		return m, cmd
	}
}

func (m Model) View() tea.View {
	if !m.initialized {
		return tea.NewView("Loading...")
	}

	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	for _, panel := range m.stack {
		canvas.Compose(panel.View().Content)
	}

	// Footer
	footerContent := RenderFooter(m.selectedRow, m.total, m.Store.Name(), m.hints, m.Width)
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

func (m Model) getPage(offset, size int) tea.Cmd {

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

func (m Model) getLine(id string) tea.Cmd {

	return func() tea.Msg {
		line, err := m.Store.GetLine(id)
		if err != nil {
			return err
		}
		return documentpanel.LineMsg{Line: line}
	}
}

// reloadColumns loads layout from file and sends column updates
func (m Model) reloadColumns() tea.Cmd {

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

// loadStore loads a file into the store asynchronously.
// Todo: generally straighten: dry with reloadColumns, unfuzzle store View stuff
func (m Model) loadStore(path string) tea.Cmd {

	return func() tea.Msg {
		err := m.Store.Load(path, 0)
		if err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
		// Todo: ignore input (except ctrl-c) during load

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

		fields, _, err := m.Store.GetView()
		if err != nil {
			return err
		}

		return message.FileLoadedMsg{
			Path:    path,
			Columns: layout.Columns,
			Fields:  fields,
		}
	}
}
