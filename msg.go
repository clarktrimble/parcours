package parcours

// pageMsg contains a page of lines and their fields
type pageMsg struct {
	fields []Field
	lines  []Line
	count  int
}

// lineMsg contains a full line
// Todo: disambiguate line from lines above
type lineMsg struct {
	line map[string]any
}

// errorMsg contains an error
type errorMsg struct {
	err error
}

// getPageMsg signals to load a page of lines
type getPageMsg struct {
	offset int
	size   int
}
