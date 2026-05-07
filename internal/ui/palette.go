package ui

import (
	"fmt"
	"image/color"

	lipgloss "charm.land/lipgloss/v2"
)

var (
	// Backgrounds — Terracotta & Cream Palette
	BgBase    = lipgloss.Color("#0f0d0c") // Canvas
	BgPanel   = lipgloss.Color("#151210") // Panel BG
	BgElevate = lipgloss.Color("#1c1815") // Raised / hover
	BgHover   = lipgloss.Color("#1c1815") // Selected row BG

	// Foregrounds
	FgPrimary = lipgloss.Color("#F0E8DC") // Cream (primary text)
	FgSub     = lipgloss.Color("#B8ADA0") // Cream mid (artist, metadata)
	FgMuted   = lipgloss.Color("#7A6F65") // Cream dim (secondary text)
	FgGhost   = lipgloss.Color("#423932") // Cream ghost (inactive indices)

	// Borders & Surface
	BorderDim  = lipgloss.Color("#2e2822")
	BorderMid  = lipgloss.Color("#443c34")
	SurfaceDeep = lipgloss.Color("#24201c") // Progress track

	// Accent — Terracotta
	Terracotta       = lipgloss.Color("#C84B2F") // Border active, hero
	TerracottaBright = lipgloss.Color("#E05A38") // Progress head, playing title
	TerracottaDim    = lipgloss.Color("#8a3320") // Muted accent
	TerracottaFaint  = lipgloss.Color("#6b2818") // Viz low bars

	// Semantic
	Warm   = lipgloss.Color("#F59E0B")
	Danger = lipgloss.Color("#F43F5E")
	Ok     = lipgloss.Color("#6A9A72") // Status green

	// Spectrum (visualizer gradient: Terracotta -> Cream)
	Sp0 = lipgloss.Color("#6b2818") // Terracotta faint
	Sp1 = lipgloss.Color("#C84B2F") // Terracotta
	Sp2 = lipgloss.Color("#E05A38") // Terracotta bright
	Sp3 = lipgloss.Color("#F0E8DC") // Cream
)

// Interpolate colors for a smooth gradient (0.0 to 1.0)
func GetGradientColor(percent float64) color.Color {
	colors := []string{"#6b2818", "#C84B2F", "#E05A38", "#F0E8DC"}
	if percent <= 0 { return lipgloss.Color(colors[0]) }
	if percent >= 1 { return lipgloss.Color(colors[len(colors)-1]) }

	idx := percent * float64(len(colors)-1)
	i := int(idx)
	f := idx - float64(i)

	c1 := hexToRGB(colors[i])
	c2 := hexToRGB(colors[i+1])

	r := uint8(float64(c1[0]) + f*(float64(c2[0])-float64(c1[0])))
	g := uint8(float64(c1[1]) + f*(float64(c2[1])-float64(c1[1])))
	b := uint8(float64(c1[2]) + f*(float64(c2[2])-float64(c1[2])))

	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func hexToRGB(h string) [3]uint8 {
	var r, g, b uint8
	fmt.Sscanf(h, "#%02x%02x%02x", &r, &g, &b)
	return [3]uint8{r, g, b}
}
