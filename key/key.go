package key

import tea "charm.land/bubbletea/v2"

// Binding represents a key binding with help text.
type Binding struct {
	keys    []string
	help    Help
	enabled bool
}

// Help describes a binding for display.
type Help struct {
	Key  string
	Desc string
}

// New creates an enabled binding.
func New(keys ...string) Binding {
	return Binding{
		keys:    keys,
		enabled: true,
	}
}

// WithHelp sets the help text.
func (b Binding) WithHelp(key, desc string) Binding {
	b.help = Help{Key: key, Desc: desc}
	return b
}

// SetEnabled enables or disables the binding.
func (b Binding) SetEnabled(v bool) Binding {
	b.enabled = v
	return b
}

// Enabled returns whether the binding is active.
func (b Binding) Enabled() bool {
	return b.enabled
}

// Help returns the help text.
func (b Binding) Help() Help {
	return b.help
}

// Matches returns true if the key message matches this binding.
func Matches(msg tea.KeyPressMsg, b Binding) bool {
	if !b.enabled {
		return false
	}
	k := msg.String()
	for _, key := range b.keys {
		if key == k {
			return true
		}
	}
	return false
}
