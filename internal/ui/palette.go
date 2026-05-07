package ui

import lipgloss "charm.land/lipgloss/v2"

var (
	// Backgrounds — 4 layers of depth
	BgBase    = lipgloss.Color("#0D0D14") // canvas
	BgPanel   = lipgloss.Color("#12121C") // panels
	BgElevate = lipgloss.Color("#1A1A28") // cards, rows
	BgHover   = lipgloss.Color("#222235") // selection

	// Foregrounds
	FgPrimary = lipgloss.Color("#E8E6F0") // main text
	FgSub     = lipgloss.Color("#9993B0") // secondary
	FgMuted   = lipgloss.Color("#4A4660") // metadata, timestamps
	FgGhost   = lipgloss.Color("#242235") // borders, dividers

	// Accent — one hero color only
	Teal       = lipgloss.Color("#2DD4BF") // active, playing, focus
	TealDim    = lipgloss.Color("#0F4F47") // teal backgrounds
	TealBright = lipgloss.Color("#99F6E4") // progress head, highlights

	// Semantic
	Warm   = lipgloss.Color("#F59E0B") // liked, starred
	Danger = lipgloss.Color("#F43F5E") // error, remove
	Ok     = lipgloss.Color("#34D399") // success, online

	// Spectrum (visualizer only)
	Sp0 = lipgloss.Color("#3B82F6")
	Sp1 = lipgloss.Color("#2DD4BF")
	Sp2 = lipgloss.Color("#F59E0B")
	Sp3 = lipgloss.Color("#F43F5E")
)
