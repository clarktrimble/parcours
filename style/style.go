package style

import "charm.land/lipgloss/v2"

var (
	BackgroundColor  = lipgloss.Color("232")
	TableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	HlRowStyle       = lipgloss.NewStyle().Background(lipgloss.Color("234")) // Subtle row highlight
	HlColStyle       = lipgloss.NewStyle().Background(lipgloss.Color("234")) // Subtle column highlight
	HlCellStyle      = lipgloss.NewStyle().Background(lipgloss.Color("236")) // Combined effect - slightly brighter
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

// CellStyler returns a StyleFunc that highlights the selected cell, row, and column
func CellStyler(selectedRow, selectedCol int) func(row, col int) lipgloss.Style {
	return func(row, col int) lipgloss.Style {
		rowMatch := row == selectedRow
		colMatch := col == selectedCol

		if rowMatch && colMatch {
			return HlCellStyle // Brightest - the selected cell
		} else if rowMatch {
			return HlRowStyle // Medium - selected row
		} else if colMatch {
			return HlColStyle // Medium - selected column
		}
		return UnStyle
	}
}
