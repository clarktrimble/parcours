package detail

import nt "parcours/entity"

type DetailMsg interface {
	isDetailMsg()
}

func (SizeMsg) isDetailMsg()    {}
func (LineMsg) isDetailMsg()    {}
func (ColumnsMsg) isDetailMsg() {}

type SizeMsg struct {
	Width  int
	Height int
}

type LineMsg struct {
	Line map[string]any
}

type ColumnsMsg struct {
	Columns []nt.Column
}
