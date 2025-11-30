package message

// lineMsg contains a full line
// Todo: disambiguate line from lines elsewhere (thisn is full/raw)
type LineMsg struct {
	Line map[string]any
}

// ErrorMsg contains an error
type ErrorMsg struct {
	Err error
}

// GetPageMsg signals to load a page of lines
type GetPageMsg struct {
	Offset int
	Size   int
}
