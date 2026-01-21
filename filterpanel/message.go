package filterpanel

import (
	tea "charm.land/bubbletea/v2"

	"parcours/message"
)

// Msg wraps a msg as originating from filterpanel.
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
