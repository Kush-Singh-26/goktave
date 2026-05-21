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

	// Premium Custom UI Styles
	StyleLeftHighlight lipgloss.Style
	StyleBadgeLiked     lipgloss.Style
	StyleBadgeCached    lipgloss.Style
	StyleBadgeOffline   lipgloss.Style
	StyleSelectedText   lipgloss.Style
	StyleVolumeIcon     lipgloss.Style
	StylePlaybarThumb   lipgloss.Style
)

func init() {
	RefreshStyles()
}

func RefreshStyles() {
	UpdatePalette()

	StyleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(Accent)

	StyleSelected = lipgloss.NewStyle().
		Bold(true).
		Foreground(AccentBright)

	StyleNormal = lipgloss.NewStyle().
		Foreground(FgPrimary)

	StyleMeta = lipgloss.NewStyle().
		Foreground(FgSub)

	StyleInput = lipgloss.NewStyle().
		Foreground(FgPrimary)

	PaneStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderDim).
		Padding(0, 1)

	// Beautiful double border for focused pane
	ActivePaneStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(Accent).
		Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderDim).
		Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderDim).
		Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(FgMuted)

	ActiveTabStyle = TabStyle.Copy().
		Foreground(Accent).
		Bold(true)

	ThumbnailStyle = lipgloss.NewStyle().
		Padding(0).
		Align(lipgloss.Center)

	ThumbnailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderDim).
		Padding(0)

	DocStyle = lipgloss.NewStyle().
		Background(BgBase).
		Foreground(FgPrimary)

	// Custom aesthetics
	StyleLeftHighlight = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	StyleBadgeLiked = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f43f5e")).
		Bold(true)

	StyleBadgeCached = lipgloss.NewStyle().
		Foreground(Ok).
		Bold(true)

	StyleBadgeOffline = lipgloss.NewStyle().
		Foreground(FgMuted).
		Bold(true)

	StyleSelectedText = lipgloss.NewStyle().
		Foreground(AccentBright).
		Bold(true)

	StyleVolumeIcon = lipgloss.NewStyle().
		Foreground(Accent)

	StylePlaybarThumb = lipgloss.NewStyle().
		Foreground(AccentBright).
		Bold(true)
}
