package parcours

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

func RenderTable(t *table.Table, fields []Field, lines []Line, selectedRow, width int, layout *Layout) string {
	// Build field lookup map
	fieldIndex := make(map[string]int)
	for i, f := range fields {
		fieldIndex[f.Name] = i
	}

	// Clear existing rows and add new data
	t.ClearRows()

	// Add data rows
	for _, line := range lines {
		var row []string
		for _, col := range layout.Columns {
			if col.Hidden || col.Demote {
				continue
			}

			// Get field and format value
			field := fields[fieldIndex[col.Field]]
			idx := fieldIndex[col.Field]
			formatted := formatValue(line[idx], field.Type, col.Format)

			// Pad/truncate to exact width
			padded := fmt.Sprintf("%-*.*s", col.Width, col.Width, formatted)
			row = append(row, padded)
		}
		t.Row(row...)
	}

	return t.Render()
}

// RenderFooter renders a footer with metadata about the table.
func RenderFooter(current, total int, filename string, width int) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	left := fmt.Sprintf("%d/%d", current, total)
	right := filename

	// Calculate padding
	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	footer := style.Render(left + strings.Repeat(" ", padding) + right)
	return footer
}

func formatValue(val Value, fieldType, format string) string {
	// TODO: Duck should normalize field types (TIMESTAMP -> timestamp)
	if format != "" && fieldType == "TIMESTAMP" {
		if t, err := val.Time(); err == nil {
			return t.Format(format)
		}
	}
	return val.String()
}
