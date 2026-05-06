package ui

import lipgloss "charm.land/lipgloss/v2"

var (
	StyleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	StyleSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560")).Background(lipgloss.Color("#1a1a1a"))
	StyleNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e8e8e8"))
	StyleMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	StyleInput    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#333333")).Padding(0, 1)
)