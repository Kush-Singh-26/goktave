package ui

import (
	"fmt"
	"image/color"

	lipgloss "charm.land/lipgloss/v2"
)

type Theme struct {
	Name string

	// Backgrounds
	BgBase    color.Color
	BgPanel   color.Color
	BgElevate color.Color
	BgHover   color.Color

	// Foregrounds
	FgPrimary color.Color
	FgSub     color.Color
	FgMuted   color.Color
	FgGhost   color.Color

	// Borders & Surface
	BorderDim   color.Color
	BorderMid   color.Color
	SurfaceDeep color.Color

	// Accents
	Accent       color.Color
	AccentBright color.Color
	AccentDim    color.Color
	AccentFaint  color.Color

	// Semantic
	Warm   color.Color
	Danger color.Color
	Ok     color.Color

	// Spectrum
	Spectrum []string
}

var Themes = []Theme{
	{
		Name:      "Terracotta (Default)",
		BgBase:    lipgloss.Color("#0f0d0c"),
		BgPanel:   lipgloss.Color("#151210"),
		BgElevate: lipgloss.Color("#1c1815"),
		BgHover:   lipgloss.Color("#1c1815"),
		FgPrimary: lipgloss.Color("#F0E8DC"),
		FgSub:     lipgloss.Color("#B8ADA0"),
		FgMuted:   lipgloss.Color("#7A6F65"),
		FgGhost:   lipgloss.Color("#423932"),
		BorderDim: lipgloss.Color("#2e2822"),
		BorderMid: lipgloss.Color("#443c34"),
		SurfaceDeep: lipgloss.Color("#24201c"),
		Accent:       lipgloss.Color("#C84B2F"),
		AccentBright: lipgloss.Color("#E05A38"),
		AccentDim:    lipgloss.Color("#8a3320"),
		AccentFaint:  lipgloss.Color("#6b2818"),
		Warm:         lipgloss.Color("#F59E0B"),
		Danger:       lipgloss.Color("#F43F5E"),
		Ok:           lipgloss.Color("#6A9A72"),
		Spectrum:     []string{"#6b2818", "#C84B2F", "#E05A38", "#F0E8DC"},
	},
	{
		Name:      "Catppuccin Mocha",
		BgBase:    lipgloss.Color("#1e1e2e"),
		BgPanel:   lipgloss.Color("#181825"),
		BgElevate: lipgloss.Color("#313244"),
		BgHover:   lipgloss.Color("#313244"),
		FgPrimary: lipgloss.Color("#cdd6f4"),
		FgSub:     lipgloss.Color("#bac2de"),
		FgMuted:   lipgloss.Color("#7f849c"),
		FgGhost:   lipgloss.Color("#585b70"),
		BorderDim: lipgloss.Color("#313244"),
		BorderMid: lipgloss.Color("#45475a"),
		SurfaceDeep: lipgloss.Color("#313244"),
		Accent:       lipgloss.Color("#cba6f7"), // Mauve
		AccentBright: lipgloss.Color("#f5c2e7"), // Pink
		AccentDim:    lipgloss.Color("#94e2d5"), // Teal
		AccentFaint:  lipgloss.Color("#89b4fa"), // Blue
		Warm:         lipgloss.Color("#f9e2af"),
		Danger:       lipgloss.Color("#f38ba8"),
		Ok:           lipgloss.Color("#a6e3a1"),
		Spectrum:     []string{"#89b4fa", "#cba6f7", "#f5c2e7", "#cdd6f4"},
	},
	{
		Name:      "Nord",
		BgBase:    lipgloss.Color("#2e3440"),
		BgPanel:   lipgloss.Color("#242933"),
		BgElevate: lipgloss.Color("#3b4252"),
		BgHover:   lipgloss.Color("#3b4252"),
		FgPrimary: lipgloss.Color("#eceff4"),
		FgSub:     lipgloss.Color("#e5e9f0"),
		FgMuted:   lipgloss.Color("#d8dee9"),
		FgGhost:   lipgloss.Color("#4c566a"),
		BorderDim: lipgloss.Color("#3b4252"),
		BorderMid: lipgloss.Color("#434c5e"),
		SurfaceDeep: lipgloss.Color("#3b4252"),
		Accent:       lipgloss.Color("#88c0d0"), // Frost blue
		AccentBright: lipgloss.Color("#8fbcbb"), // Frost cyan
		AccentDim:    lipgloss.Color("#81a1c1"), // Frost mid blue
		AccentFaint:  lipgloss.Color("#5e81ac"), // Frost deep blue
		Warm:         lipgloss.Color("#ebcb8b"),
		Danger:       lipgloss.Color("#bf616a"),
		Ok:           lipgloss.Color("#a3be8c"),
		Spectrum:     []string{"#5e81ac", "#88c0d0", "#8fbcbb", "#eceff4"},
	},
	{
		Name:      "Everforest",
		BgBase:    lipgloss.Color("#2d353b"),
		BgPanel:   lipgloss.Color("#232a2e"),
		BgElevate: lipgloss.Color("#343f44"),
		BgHover:   lipgloss.Color("#343f44"),
		FgPrimary: lipgloss.Color("#d3c6aa"),
		FgSub:     lipgloss.Color("#bdae93"),
		FgMuted:   lipgloss.Color("#859289"),
		FgGhost:   lipgloss.Color("#475258"),
		BorderDim: lipgloss.Color("#343f44"),
		BorderMid: lipgloss.Color("#3d484d"),
		SurfaceDeep: lipgloss.Color("#343f44"),
		Accent:       lipgloss.Color("#a7c080"), // Green
		AccentBright: lipgloss.Color("#dbbc7f"), // Yellow
		AccentDim:    lipgloss.Color("#83c092"), // Aqua
		AccentFaint:  lipgloss.Color("#7fbbb3"), // Blue
		Warm:         lipgloss.Color("#e69875"),
		Danger:       lipgloss.Color("#e67e80"),
		Ok:           lipgloss.Color("#a7c080"),
		Spectrum:     []string{"#7fbbb3", "#a7c080", "#dbbc7f", "#d3c6aa"},
	},
}

var ActiveTheme = &Themes[0]

// Legacy variables for compatibility
var (
	BgBase    = ActiveTheme.BgBase
	BgPanel   = ActiveTheme.BgPanel
	BgElevate = ActiveTheme.BgElevate
	BgHover   = ActiveTheme.BgHover
	FgPrimary = ActiveTheme.FgPrimary
	FgSub     = ActiveTheme.FgSub
	FgMuted   = ActiveTheme.FgMuted
	FgGhost   = ActiveTheme.FgGhost
	BorderDim  = ActiveTheme.BorderDim
	BorderMid  = ActiveTheme.BorderMid
	SurfaceDeep = ActiveTheme.SurfaceDeep
	Terracotta       = ActiveTheme.Accent
	TerracottaBright = ActiveTheme.AccentBright
	TerracottaDim    = ActiveTheme.AccentDim
	TerracottaFaint  = ActiveTheme.AccentFaint
	Warm   = ActiveTheme.Warm
	Danger = ActiveTheme.Danger
	Ok     = ActiveTheme.Ok
	Sp0 = lipgloss.Color(ActiveTheme.Spectrum[0])
	Sp1 = lipgloss.Color(ActiveTheme.Spectrum[1])
	Sp2 = lipgloss.Color(ActiveTheme.Spectrum[2])
	Sp3 = lipgloss.Color(ActiveTheme.Spectrum[3])
)

func UpdatePalette() {
	BgBase = ActiveTheme.BgBase
	BgPanel = ActiveTheme.BgPanel
	BgElevate = ActiveTheme.BgElevate
	BgHover = ActiveTheme.BgHover
	FgPrimary = ActiveTheme.FgPrimary
	FgSub = ActiveTheme.FgSub
	FgMuted = ActiveTheme.FgMuted
	FgGhost = ActiveTheme.FgGhost
	BorderDim = ActiveTheme.BorderDim
	BorderMid = ActiveTheme.BorderMid
	SurfaceDeep = ActiveTheme.SurfaceDeep
	Terracotta = ActiveTheme.Accent
	TerracottaBright = ActiveTheme.AccentBright
	TerracottaDim = ActiveTheme.AccentDim
	TerracottaFaint = ActiveTheme.AccentFaint
	Warm = ActiveTheme.Warm
	Danger = ActiveTheme.Danger
	Ok = ActiveTheme.Ok
	Sp0 = lipgloss.Color(ActiveTheme.Spectrum[0])
	Sp1 = lipgloss.Color(ActiveTheme.Spectrum[1])
	Sp2 = lipgloss.Color(ActiveTheme.Spectrum[2])
	Sp3 = lipgloss.Color(ActiveTheme.Spectrum[3])
}

func GetGradientColor(percent float64) color.Color {
	colors := ActiveTheme.Spectrum
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
