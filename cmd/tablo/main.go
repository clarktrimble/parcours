package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"parcours"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Simple logger that prints to stdout
type simpleLogger struct{}

func (l *simpleLogger) Info(ctx context.Context, msg string, kv ...any) {
	fmt.Printf("[INFO] %s %v\n", msg, kv)
}

func (l *simpleLogger) Error(ctx context.Context, msg string, err error, kv ...any) {
	fmt.Printf("[ERROR] %s: %v %v\n", msg, err, kv)
}

type model struct {
	store parcours.Store

	// View data
	fields     []parcours.Field
	lines      []parcours.Line
	totalLines int

	// Display state
	scrollOffset int
	selectedRow  int
	width        int
	height       int
}

type loadDataMsg struct {
	fields []parcours.Field
	lines  []parcours.Line
	count  int
	err    error
}

func initialModel(store parcours.Store) model {
	return model{
		store: store,
	}
}

func (m model) Init() tea.Cmd {
	return m.loadData()
}

func (m model) loadData() tea.Cmd {
	return func() tea.Msg {
		fields, count, err := m.store.GetView()
		if err != nil {
			return loadDataMsg{err: err}
		}

		lines, err := m.store.GetPage(m.scrollOffset, 20)
		if err != nil {
			return loadDataMsg{err: err}
		}

		return loadDataMsg{
			fields: fields,
			lines:  lines,
			count:  count,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadDataMsg:
		if msg.err != nil {
			// TODO: handle error
			return m, nil
		}
		m.fields = msg.fields
		m.lines = msg.lines
		m.totalLines = msg.count
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.selectedRow > 0 {
				m.selectedRow--
			} else if m.scrollOffset > 0 {
				m.scrollOffset--
				return m, m.loadData()
			}
		case "down", "j":
			if m.selectedRow < len(m.lines)-1 {
				m.selectedRow++
			} else if m.scrollOffset+len(m.lines) < m.totalLines {
				m.scrollOffset++
				return m, m.loadData()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder

	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	// Calculate column widths (simple equal distribution for now)
	colWidths := calculateColumnWidths(m.fields, m.width)

	// Header row
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	var headerCols []string
	for i, field := range m.fields {
		col := headerStyle.Width(colWidths[i]).Render(field.Name)
		headerCols = append(headerCols, col)
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, headerCols...))
	b.WriteString("\n")

	// Separator
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	b.WriteString(sepStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	// Data rows
	for i, line := range m.lines {
		var rowCols []string
		for j, val := range line {
			cellStyle := lipgloss.NewStyle()

			// Highlight selected row
			if i == m.selectedRow {
				cellStyle = cellStyle.Background(lipgloss.Color("63"))
			}

			formatted := formatValue(val, m.fields[j].Type)
			col := cellStyle.Width(colWidths[j]).Render(formatted)
			rowCols = append(rowCols, col)
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, rowCols...))
		b.WriteString("\n")
	}

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("Lines: %d | ↑/↓ navigate | q quit", m.totalLines))
	b.WriteString("\n")
	b.WriteString(footer)

	return tea.NewView(b.String())
}

func calculateColumnWidths(fields []parcours.Field, totalWidth int) []int {
	if len(fields) == 0 {
		return nil
	}

	// Simple strategy: divide equally
	// TODO: Make smarter based on field type
	widths := make([]int, len(fields))
	colWidth := totalWidth / len(fields)

	for i := range fields {
		widths[i] = colWidth
	}

	// Give any remainder to the last column (usually message)
	widths[len(widths)-1] += totalWidth % len(fields)

	return widths
}

func formatValue(val parcours.Value, fieldType string) string {
	switch fieldType {
	case "timestamp":
		if t, err := val.Time(); err == nil {
			return t.Format("15:04:05")
		}
	}
	return val.String()
}

func main() {

	logger := &simpleLogger{}
	dk, err := duck.New(logger)
	if err != nil {
		panic(err)
	}
	defer dk.Close()

	logFile := "test/data/blah.log"
	if err := dk.Load(logFile, 0); err != nil {
		panic(err)
	}

	if err := dk.SetView(parcours.Filter{}, nil); err != nil {
		panic(err)
	}

	var store parcours.Store = nil
	p := tea.NewProgram(initialModel(store))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
