package piece

import (
	tea "charm.land/bubbletea/v2"

	"parcours/board"
	nt "parcours/entity"
)

// Value displays a formatted value while preserving the raw value for filtering
type Value struct {
	raw       nt.Value
	formatter func(nt.Value) string
}

func NewValue(raw nt.Value, formatter func(nt.Value) string) Value {
	if formatter == nil {
		formatter = func(v nt.Value) string { return v.String() }
	}
	return Value{raw: raw, formatter: formatter}
}

func (v Value) Update(msg tea.Msg) (board.Piece, tea.Cmd) {
	return v, nil
}

func (v Value) Render() string {
	return v.formatter(v.raw)
}

func (v Value) Value() string {
	return v.raw.String()
}
