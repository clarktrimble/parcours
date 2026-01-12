package detail

import nt "parcours/entity"

// LineMsg delivers full line data for detail view.
type LineMsg struct {
	Line map[string]any
}

// ColumnsMsg updates the column configuration.
type ColumnsMsg struct {
	Columns []nt.Column
}

// DismissedMsg signals the detail panel wants to close.
type DismissedMsg struct{}

// OpenDetailMsg requests opening detail view for a line.
type OpenDetailMsg struct {
	Id string
}
