package message

// pageMsg contains a page of lines and their fields
//type pageMsg struct {
//fields []Field
//lines  []Line
//count  int
//}

// lineMsg contains a full line
// Todo: disambiguate line from lines above
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

// resetMsg signals to reset table position to start
//type resetMsg struct{}

// panelSizeMsg signals panel size computed by Model's layout manager
//type panelSizeMsg struct {
//width  int
//height int
//}
