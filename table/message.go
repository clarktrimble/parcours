package table

import nt "parcours/entity"

type TableMsg interface {
	isTableMsg()
}

func (SizeMsg) isTableMsg()    {}
func (PageMsg) isTableMsg()    {}
func (ColumnsMsg) isTableMsg() {}
func (ResetMsg) isTableMsg()   {}

type SizeMsg struct {
	Width  int
	Height int
}

type PageMsg struct {
	Fields []nt.Field
	Lines  []nt.Line
	Count  int
}

type ColumnsMsg struct {
	Columns []nt.Column
	Fields  []nt.Field
}

type ResetMsg struct{}
