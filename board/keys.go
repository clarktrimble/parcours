package board

import (
	"parcours/key"
	"parcours/message"
)

// Todo: look at:
/*
  type KeyBind struct {
      Keys   []string  // what triggers it
      Action string    // what it does: "close", "navigate up"
  }
*/

// KeyMap defines board's key bindings.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Edit     key.Binding
	ExitEdit key.Binding
}

// DefaultKeyMap returns the default board key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       key.New("up", "k").WithHelp("k", "up"),
		Down:     key.New("down", "j").WithHelp("j", "down"),
		Left:     key.New("left", "h").WithHelp("h", "left"),
		Right:    key.New("right", "l").WithHelp("l", "right"),
		PageUp:   key.New("pgup").WithHelp("pgup", "page up"),
		PageDown: key.New("pgdown").WithHelp("pgdn", "page down"),
		Top:      key.New("g", "alt+pgup").WithHelp("g", "top"),
		Bottom:   key.New("G", "alt+pgdown").WithHelp("G", "bottom"),
		Edit:     key.New("i").WithHelp("i", "edit"),
		ExitEdit: key.New("enter").WithHelp("enter", "exit edit"),
	}
}

// Hints returns the current hints based on edit mode.
func (km KeyMap) Hints(editMode bool) []message.Hint {
	if editMode {
		h := km.ExitEdit.Help()
		return []message.Hint{{Key: h.Key, Desc: h.Desc}}
	}
	bindings := []key.Binding{
		km.Up, km.Down, km.Left, km.Right, km.Top, km.Bottom,
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
