package piece

import tea "charm.land/bubbletea/v2"

// Label displays static text
type Label struct {
	text string
}

func NewLabel(text string) Label {
	return Label{text: text}
}

func (l Label) Init() tea.Cmd { return nil }

func (l Label) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return l, nil
}

func (l Label) View() tea.View {
	return tea.NewView(l.text)
}
