package linespanel

import (
	nt "parcours/entity"
)

// LinesMsg is a marker interface for messages destined for LinesPanel
type LinesMsg interface {
	linesMsg()
}

// SizeMsg tells the panel its display size
type SizeMsg struct {
	Width  int
	Height int
}

func (SizeMsg) linesMsg() {}

// PageMsg delivers a page of line data
type PageMsg struct {
	Lines []nt.Line
	Count int // Total count after filtering
}

func (PageMsg) linesMsg() {}

// ColumnsMsg updates the column configuration
type ColumnsMsg struct {
	Columns []nt.Column
	Fields  []nt.Field
}

func (ColumnsMsg) linesMsg() {}

// ResetMsg resets the panel to initial state
type ResetMsg struct{}

func (ResetMsg) linesMsg() {}
