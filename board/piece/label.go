package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

// Label is a read-only text cell
type Label struct {
	text string
}

func NewLabel(text string) Label {
	return Label{text: text}
}

func (l Label) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
	return l, nil
}

func (l Label) Text() string {
	return l.text
}

func (l Label) Render() string {
	return l.text
}

// Todo: dehack, this is here so Label can act as Field
func (l Label) String() string {
	return l.text
}
