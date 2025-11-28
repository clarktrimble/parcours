package parcours

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

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
