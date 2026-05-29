package parcours

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"parcours/message"
)

// RenderFooter renders a 2-line footer: help above, filename/count below.
// current is 0-indexed internally, but displayed as 1-indexed for users.
func RenderFooter(current, total int, filename string, hints []message.Hint, width int) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Line 1: hints
	helpLine := renderHelp(hints, width)

	// Line 2: filename ... row/total
	left := filename
	right := fmt.Sprintf("%d/%d", current+1, total)
	// Todo: I can has unfiltered total total?
	usedWidth := lipgloss.Width(left) + lipgloss.Width(right)
	padding := max(0, width-usedWidth)
	infoLine := style.Render(left + strings.Repeat(" ", padding) + right)

	return helpLine + "\n" + infoLine
}

// renderHelp formats hints as a single line.
func renderHelp(hints []message.Hint, width int) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	var parts []string
	for _, h := range hints {
		parts = append(parts, fmt.Sprintf("%s:%s", h.Key, h.Desc))
	}

	helpStr := strings.Join(parts, "  ")

	// Pad to width
	padLen := max(0, width-lipgloss.Width(helpStr))
	return style.Render(helpStr + strings.Repeat(" ", padLen))
}
