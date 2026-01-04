package piece

import (
	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
)

// Value displays a formatted value while preserving the raw value for filtering
type Value struct {
	value     nt.Value
	formatter func(nt.Value) string
}

func NewValue(raw nt.Value, formatter func(nt.Value) string) Value {
	if formatter == nil {
		formatter = func(v nt.Value) string { return v.String() }
	}
	return Value{value: raw, formatter: formatter}
}

func (v Value) Init() tea.Cmd { return nil }

func (v Value) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return v, nil
}

func (v Value) View() tea.View {
	return tea.NewView(v.formatter(v.value))
}

func (v Value) Render() string {
	return v.formatter(v.value)
}

func (v Value) Value() string {
	// Todo: b-but it just returns string, do we ever want num,time,etc or is ducksql cool with string?
	return v.value.String()
}
