package intake

import tea "charm.land/bubbletea/v2"

// FileSelectedMsg signals that a file was selected for loading
type FileSelectedMsg struct {
	Path string
}

// Msg is a msg that can be routed as intake.Msg.
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
