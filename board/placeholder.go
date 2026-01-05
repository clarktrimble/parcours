package board

import tea "charm.land/bubbletea/v2"

// Placeholder is a tea.Model that shows a message
type Placeholder struct {
	message string
}

func NewPlaceholder(message string) Placeholder {
	return Placeholder{message: message}
}

func (p Placeholder) Init() tea.Cmd {
	return nil
}

func (p Placeholder) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return p, nil
}

func (p Placeholder) View() tea.View {
	return tea.NewView(p.message)
}
