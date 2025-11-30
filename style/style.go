package style

import "charm.land/lipgloss/v2"

var (
	TableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	HlRowStyle       = lipgloss.NewStyle().Background(lipgloss.Color("63"))
	MutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	UnStyle          = lipgloss.NewStyle()
)
