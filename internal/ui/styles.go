package ui

import lipgloss "charm.land/lipgloss/v2"

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Teal)

	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(TealBright).
			Background(BgHover)

	StyleNormal = lipgloss.NewStyle().
			Foreground(FgPrimary)

	StyleMeta = lipgloss.NewStyle().
			Foreground(FgMuted)

	StyleInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(FgGhost).
			Padding(0, 1)
)
