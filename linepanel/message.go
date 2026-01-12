package linepanel

import (
	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
)

// Msg is a msg that can be routed as linepanel.Msg.
type Msg struct {
	Wrapped tea.Msg
}

// Unwrap returns the wrapped message (implements Unwrapper)
func (m Msg) Unwrap() tea.Msg {
	return m.Wrapped
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

		// Todo: handle batch in intake and fp too plz
		cmds, ok := msg.(tea.BatchMsg)
		if !ok {
			return Msg{Wrapped: msg}
		}

		wrapped := make([]tea.Cmd, len(cmds))
		for i, cmd := range cmds {
			wrapped[i] = Wrap(cmd)
		}
		return tea.BatchMsg(wrapped)
	}
}

// PageMsg delivers a page of line data.
type PageMsg struct {
	Lines []nt.Line
	Count int // Total count after filtering
}

// ColumnsMsg updates the column configuration.
type ColumnsMsg struct {
	Columns []nt.Column
	Fields  []nt.Field
}

// ResetMsg resets the panel to initial state.
type ResetMsg struct{}

// OpenDetailMsg requests opening detail view for a line.
type OpenDetailMsg struct {
	Id      string
	Columns []nt.Column
}

// DismissedMsg signals linepanel wants to close (esc pressed).
type DismissedMsg struct{}
