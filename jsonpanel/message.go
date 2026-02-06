package jsonpanel

import nt "parcours/entity"

// LineMsg delivers full line data for detail view.
type LineMsg struct {
	Line map[string]any
}

// ColumnsMsg updates the column configuration.
type ColumnsMsg struct {
	Columns []nt.Column
}
