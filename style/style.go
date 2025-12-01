package style

import "charm.land/lipgloss/v2"

var (
	TableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	HlRowStyle       = lipgloss.NewStyle().Background(lipgloss.Color("63"))
	MutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	UnStyle          = lipgloss.NewStyle()
)

// RowStyler returns a StyleFunc that highlights the selected row
func RowStyler(selectedRow int) func(row, col int) lipgloss.Style {
	return func(row, col int) lipgloss.Style {
		if row == selectedRow {
			return HlRowStyle
		}
		return UnStyle
	}
}
