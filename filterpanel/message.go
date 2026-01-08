package filterpanel

import tea "charm.land/bubbletea/v2"

// Msg is a msg that can be routed as filterpanel.Msg.
type Msg struct {
	Wrapped tea.Msg
}

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
