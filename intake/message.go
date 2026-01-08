package intake

// SizeMsg tells the intake panel its display size
//
// Todo: does intake, etc need own SizeMsg really?
// -     these flow naturally thru direct Update to children?
type SizeMsg struct {
	Width  int
	Height int
}

// FileSelectedMsg signals that a file was selected for loading
type FileSelectedMsg struct {
	Path string
}
