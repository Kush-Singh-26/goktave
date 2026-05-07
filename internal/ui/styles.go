package ui

import lipgloss "charm.land/lipgloss/v2"

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Terracotta)

	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(TerracottaBright).
			Background(BgHover)

	StyleNormal = lipgloss.NewStyle().
			Foreground(FgPrimary)

	StyleMeta = lipgloss.NewStyle().
			Foreground(FgSub)

	StyleInput = lipgloss.NewStyle().
			Foreground(FgPrimary).
			Background(BgPanel)

	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderMid).
			Padding(0, 1).
			Background(BgPanel)

	ActivePaneStyle = PaneStyle.Copy().
				BorderForeground(Terracotta)

	HeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderMid).
			Padding(0, 1).
			Background(BgPanel)

	FooterStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderMid).
			Padding(0, 1).
			Background(BgPanel)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(FgMuted)

	ActiveTabStyle = TabStyle.Copy().
				Foreground(Terracotta).
				Bold(true).
				Underline(true)

	DocStyle = lipgloss.NewStyle().
			Background(BgBase)
)
