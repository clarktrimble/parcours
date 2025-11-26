package parcours

// pageMsg contains loaded page data
type pageMsg struct {
	fields []Field
	lines  []Line
	count  int
	err    error
}

// lineMsg contains loaded detail record data
type lineMsg struct {
	data map[string]any
	err  error
}
