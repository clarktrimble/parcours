package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	choices []string // Example data
	cursor  int      // UI state
	width   int      // For responsive rendering
	height  int
}

func initialModel() model {
	return model{
		choices: []string{"Option 1", "Option 2", "Quit"},
		cursor:  0,
	}
}

func (m model) Init() tea.Cmd {
	// Optional: Return a command here (e.g., tea.Tick for timers)
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			// Handle selection (e.g., quit if "Quit")
			if m.cursor == len(m.choices)-1 {
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, cmd
}

func (m model) View() tea.View {
	// Use Lip Gloss for styling (v2 improves IO sharing)
	var b strings.Builder
	for i, choice := range m.choices {
		cursor := " " // Default
		if m.cursor == i {
			cursor = ">" // Selected
		}
		// Style with Lip Gloss
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		if i == m.cursor {
			style = style.Background(lipgloss.Color("63"))
		}
		b.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(choice)))
	}

	// Footer with responsive width
	footer := lipgloss.NewStyle().
		MarginTop(1).
		Width(m.width).
		Align(lipgloss.Center).
		Render("↑/↓ to navigate | Enter to select | Q to quit")

	content := lipgloss.JoinVertical(lipgloss.Left, b.String(), footer)
	return tea.NewView(content)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
