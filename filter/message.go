package filter

type FilterMsg interface {
	isFilterMsg()
}

func (SizeMsg) isFilterMsg() {}

type SizeMsg struct {
	Width  int
	Height int
}
