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

// ReloadColumnsMsg requests reloading columns from layout file.
type ReloadColumnsMsg struct{}

// OpenDetailMsg requests opening detail view for a line.
type OpenDetailMsg struct {
	Id      string
	Columns []nt.Column
}

// OpenFilterMsg requests opening filter panel.
// If Field is set, a new filter is added; otherwise just view existing.
type OpenFilterMsg struct {
	Field string
	Value nt.Value
}

// OpenIntakeMsg requests opening intake panel to select a new file.
type OpenIntakeMsg struct{}

// CloseMsg signals linepanel wants to close (esc pressed).
type CloseMsg struct{}
