package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
)

// TextInput is an editable text field
type TextInput struct {
	value     string
	cursor    int
	maxLength int
}

func NewTextInput(value string, maxLength int) TextInput {
	if maxLength <= 0 {
		maxLength = 100 // Default max length
	}
	return TextInput{
		value:     value,
		cursor:    len(value),
		maxLength: maxLength,
	}
}

func (t TextInput) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
	oldValue := t.value
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "backspace":
			if t.cursor > 0 {
				t.value = t.value[:t.cursor-1] + t.value[t.cursor:]
				t.cursor--
			}
		case "delete":
			if t.cursor < len(t.value) {
				t.value = t.value[:t.cursor] + t.value[t.cursor+1:]
			}
		case "left":
			if t.cursor > 0 {
				t.cursor--
			}
		case "right":
			if t.cursor < len(t.value) {
				t.cursor++
			}
		case "home", "ctrl+a":
			t.cursor = 0
		case "end", "ctrl+e":
			t.cursor = len(t.value)
		default:
			// Insert character if it's a single rune and under max length
			if len(msg.String()) == 1 && len(t.value) < t.maxLength {
				t.value = t.value[:t.cursor] + msg.String() + t.value[t.cursor:]
				t.cursor++
			}
		}
	}
	// Only send message if value changed
	if t.value != oldValue {
		return t, func() tea.Msg {
			return &ValueChangedMsg{Value: t.value}
		}
	}
	return t, nil
}


func (t TextInput) Value() string {
	return t.value
}

func (t TextInput) Cursor() int {
	return t.cursor
}

func (t TextInput) Render() string {
	return t.value
}
