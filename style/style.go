package style

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

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

// StyleTable applies consistent table styling for borders and separators
func StyleTable(tbl *table.Table) {
	tbl.Border(lipgloss.Border{
		Top:         "─", // Horizontal parts of separator
		Middle:      "─", // Between columns in separator
		MiddleLeft:  "─", // Left edge of separator
		MiddleRight: "─", // Right edge of separator
	}).
		BorderTop(false).    // Disable top border
		BorderBottom(false). // Disable bottom border
		BorderLeft(false).   // Disable left border
		BorderRight(false).  // Disable right border
		BorderColumn(false). // Disable column separators
		BorderStyle(TableBorderStyle)
}
