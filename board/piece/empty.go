package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

var _ board.Piece = Empty{}

// Empty is a placeholder piece that renders as empty space
type Empty struct{}

func (empty Empty) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
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
