package linepanel

import (
	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
	"parcours/message"
)

// Msg wraps a msg as originating from linepanel.
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
