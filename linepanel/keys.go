package linepanel

import (
	"parcours/key"
	"parcours/message"
)

// KeyMap defines linepanel's key bindings.
type KeyMap struct {
	Close      key.Binding
	FilterCell key.Binding
	Filters    key.Binding
	Reload     key.Binding
	OpenFile   key.Binding
	Detail     key.Binding
}

// DefaultKeyMap returns the default linepanel key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Close:      key.New("esc").WithHelp("esc", "close"),
		FilterCell: key.New("c").WithHelp("c", "filter cell"),
		Filters:    key.New("f").WithHelp("f", "filters"),
		Reload:     key.New("r").WithHelp("r", "reload"),
		OpenFile:   key.New("o").WithHelp("o", "open file"),
		Detail:     key.New("enter").WithHelp("enter", "detail"),
	}
}

// Hints returns linepanel's hints.
func (km KeyMap) Hints() []message.Hint {
	bindings := []key.Binding{
		km.Detail, km.FilterCell, km.Filters, km.OpenFile, km.Close,
	}
	var hints []message.Hint
	for _, b := range bindings {
		if b.Enabled() {
			h := b.Help()
			hints = append(hints, message.Hint{Key: h.Key, Desc: h.Desc})
		}
	}
	return hints
}
