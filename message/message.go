package message

import (
	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
)

// lineMsg contains a full line
// Todo: disambiguate line from lines elsewhere (thisn is full/raw)
type LineMsg struct {
	Line map[string]any
}

// ErrorMsg contains an error
type ErrorMsg struct {
	Err error
}

// ErrorCmd returns an error cmd
func ErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Err: err}
	}
}

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

// OpenFilterMsg signals to open filter dialog with cell data
type OpenFilterMsg struct {
	Field string // Field name from column
	Value string // Value from cell
}

// SetFilterMsg signals to apply a filter to the data
type SetFilterMsg struct {
	Filter nt.Filter
}
