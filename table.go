package parcours

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func RenderTable(fields []Field, lines []Line, selectedRow, width int, layout *Layout) string {
	var b strings.Builder

	// Build field lookup maps
	fieldMap := make(map[string]Field)
	fieldIndex := make(map[string]int)
	for i, f := range fields {
		fieldMap[f.Name] = f
		fieldIndex[f.Name] = i
	}

	// Header row
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	var headerCols []string
	for _, col := range layout.Columns {
		if col.Hidden || col.Demote {
			continue
		}
		cell := headerStyle.Width(col.Width).Render(col.Field)
		headerCols = append(headerCols, cell)
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, headerCols...))
	b.WriteString("\n")

	// Separator
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	b.WriteString(sepStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Data rows
	for i, line := range lines {
		var rowCols []string
		for _, col := range layout.Columns {
			if col.Hidden || col.Demote {
				continue
			}

			cellStyle := lipgloss.NewStyle()
			if i == selectedRow {
				cellStyle = cellStyle.Background(lipgloss.Color("63"))
			}

			field := fieldMap[col.Field]
			idx := fieldIndex[col.Field]
			formatted := formatValue(line[idx], field.Type, col.Format)
			cell := cellStyle.Width(col.Width).Render(formatted)
			rowCols = append(rowCols, cell)
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, rowCols...))
		b.WriteString("\n")
	}

	return b.String()
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
