package linepanel

import (
	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
)

// Msg is a msg that can be routed as linepanel.Msg.
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
