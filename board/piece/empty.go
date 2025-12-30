package piece

import (
	tea "charm.land/bubbletea/v2"
)

// Empty is a placeholder piece that renders as empty space
type Empty struct{}

func (empty Empty) Init() tea.Cmd { return nil }

func (empty Empty) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return empty, nil
}

func (empty Empty) View() tea.View {
	return tea.NewView("")
}

func (empty Empty) Render() string {
	return ""
}

func (empty Empty) Value() string {
	return ""
}
