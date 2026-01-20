package intake

import (
	tea "charm.land/bubbletea/v2"

	"parcours/message"
)

// FileSelectedMsg signals that a file was selected for loading
type FileSelectedMsg struct {
	Path string
}

// CloseMsg signals intake wants to close (esc pressed).
type CloseMsg struct{}

// Msg wraps a msg as originating from intake.
type Msg struct {
	Wrapped tea.Msg
}

// Unwrap unwraps the wrapped msg.
func (msg Msg) Unwrap() tea.Msg {
	return msg.Wrapped
}

// Wrap produces a cmd that will wrap its msg.
func Wrap(cmd tea.Cmd) tea.Cmd {
	return message.WrapCmd(cmd, func(msg tea.Msg) tea.Msg {
		return Msg{Wrapped: msg}
	})
}
