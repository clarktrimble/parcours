package filterpanel

import tea "charm.land/bubbletea/v2"

// Msg is a msg that can be routed as filterpanel.Msg.
type Msg struct {
	Wrapped tea.Msg
}

// Unwrap returns the wrapped message (implements Unwrapper)
func (m Msg) Unwrap() tea.Msg {
	return m.Wrapped
}

// CanceledMsg signals the filter panel was dismissed without applying.
type CanceledMsg struct{}

// Wrap wraps a cmd.
func Wrap(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}

	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}

		return Msg{Wrapped: msg}
	}
}
