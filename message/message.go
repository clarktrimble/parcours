package message

import (
	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
)

// WrapCmd wraps a command or batch using a given wrapFunc.
func WrapCmd(cmd tea.Cmd, wrapFunc func(tea.Msg) tea.Msg) tea.Cmd {
	if cmd == nil {
		return nil
	}

	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}

		cmds, ok := msg.(tea.BatchMsg)
		if !ok {
			return wrapFunc(msg)
		}

		// recurse for batch
		wrapped := make([]tea.Cmd, len(cmds))
		for i, c := range cmds {
			wrapped[i] = WrapCmd(c, wrapFunc)
		}
		return tea.BatchMsg(wrapped)
	}
}

// Todo: can message pkg be obviated? or is like entity?

// GetPageMsg signals to load a page of lines
type GetPageMsg struct {
	Offset int
	Size   int
}

// CountMsg contains the total count from store
type CountMsg struct {
	Count int
}

// SelectedMsg contains row and id of selected line
type SelectedMsg struct {
	Row int
	Id  string
}

// SetFilterMsg signals to apply a filter to the data
// Todo: unify Filter and Filters - currently Filter is combined for Store,
// Filters is full list for UI state
type SetFilterMsg struct {
	Filter  nt.Filter   // combined filter for Store
	Filters []nt.Filter // full list for saving
}

// CloseMsg signals a panel wants to close.
type CloseMsg struct{}

// OpenDetailMsg requests opening detail view for a line.
type OpenDetailMsg struct {
	Id      string
	Columns []nt.Column
}

// OpenFilterMsg requests opening filter panel.
type OpenFilterMsg struct {
	Field string
	Value nt.Value
}

// OpenIntakeMsg requests opening intake panel.
type OpenIntakeMsg struct{}

// ReloadColumnsMsg requests reloading columns from layout file.
type ReloadColumnsMsg struct{}

// FileSelectedMsg signals a file was selected for loading.
type FileSelectedMsg struct {
	Path string
}
