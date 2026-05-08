package ui

import lipgloss "charm.land/lipgloss/v2"

var (
	StyleTitle      lipgloss.Style
	StyleSelected   lipgloss.Style
	StyleNormal     lipgloss.Style
	StyleMeta       lipgloss.Style
	StyleInput      lipgloss.Style
	PaneStyle       lipgloss.Style
	ActivePaneStyle lipgloss.Style
	HeaderStyle     lipgloss.Style
	FooterStyle     lipgloss.Style
	TabStyle        lipgloss.Style
	ActiveTabStyle  lipgloss.Style
	ThumbnailStyle  lipgloss.Style
	ThumbnailBoxStyle lipgloss.Style
	DocStyle        lipgloss.Style
)

func init() {
	RefreshStyles()
}

func RefreshStyles() {
	UpdatePalette()

	StyleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(Terracotta)

	StyleSelected = lipgloss.NewStyle().
		Bold(true).
		Foreground(TerracottaBright)

	StyleNormal = lipgloss.NewStyle().
		Foreground(FgPrimary)

	StyleMeta = lipgloss.NewStyle().
		Foreground(FgSub)

	StyleInput = lipgloss.NewStyle().
		Foreground(FgPrimary)

	PaneStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderMid).
		Padding(0, 1)

	ActivePaneStyle = PaneStyle.Copy().
		BorderForeground(Terracotta)

	HeaderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderMid).
		Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderMid).
		Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(FgMuted)

	ActiveTabStyle = TabStyle.Copy().
		Foreground(Terracotta).
		Bold(true).
		Underline(true)

	ThumbnailStyle = lipgloss.NewStyle().
		Padding(0).
		Align(lipgloss.Center)

	ThumbnailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderMid).
		Padding(0)

	DocStyle = lipgloss.NewStyle().
		Background(BgBase).
		Foreground(FgPrimary)
}
