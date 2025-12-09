package style

import "charm.land/lipgloss/v2"

var (
	BackgroundColor  = lipgloss.Color("234")                                 // Dark warm grey
	TableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Subtle warm grey border
	HlRowStyle       = lipgloss.NewStyle().Background(lipgloss.Color("235")) // Very subtle warm grey row
	HlColStyle       = lipgloss.NewStyle().Background(lipgloss.Color("234")) // Twice as subtle - barely visible
	HlCellStyle      = lipgloss.NewStyle().Background(lipgloss.Color("237")) // Slightly warmer cell
	MutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("246")) // Warm muted grey text
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
