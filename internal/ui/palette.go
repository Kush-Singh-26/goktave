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
	{
		Name:      "Dracula",
		BgBase:    lipgloss.Color("#282a36"),
		BgPanel:   lipgloss.Color("#21222c"),
		BgElevate: lipgloss.Color("#44475a"),
		BgHover:   lipgloss.Color("#44475a"),
		FgPrimary: lipgloss.Color("#f8f8f2"),
		FgSub:     lipgloss.Color("#e9e9e9"),
		FgMuted:   lipgloss.Color("#6272a4"),
		FgGhost:   lipgloss.Color("#44475a"),
		BorderDim: lipgloss.Color("#44475a"),
		BorderMid: lipgloss.Color("#565a7b"),
		SurfaceDeep: lipgloss.Color("#383a4a"),
		Accent:       lipgloss.Color("#ff79c6"),
		AccentBright: lipgloss.Color("#ff92df"),
		AccentDim:    lipgloss.Color("#bd93f9"),
		AccentFaint:  lipgloss.Color("#6272a4"),
		Warm:         lipgloss.Color("#f1fa8c"),
		Danger:       lipgloss.Color("#ff5555"),
		Ok:           lipgloss.Color("#50fa7b"),
		Spectrum:     []string{"#6272a4", "#bd93f9", "#ff79c6", "#f8f8f2"},
	},
	{
		Name:      "Gruvbox Dark",
		BgBase:    lipgloss.Color("#1d2021"),
		BgPanel:   lipgloss.Color("#282828"),
		BgElevate: lipgloss.Color("#3c3836"),
		BgHover:   lipgloss.Color("#3c3836"),
		FgPrimary: lipgloss.Color("#ebdbb2"),
		FgSub:     lipgloss.Color("#d5c4a1"),
		FgMuted:   lipgloss.Color("#928374"),
		FgGhost:   lipgloss.Color("#504945"),
		BorderDim: lipgloss.Color("#3c3836"),
		BorderMid: lipgloss.Color("#504945"),
		SurfaceDeep: lipgloss.Color("#32302f"),
		Accent:       lipgloss.Color("#d65d0e"),
		AccentBright: lipgloss.Color("#fe8019"),
		AccentDim:    lipgloss.Color("#b57614"),
		AccentFaint:  lipgloss.Color("#8f3f1a"),
		Warm:         lipgloss.Color("#fabd2f"),
		Danger:       lipgloss.Color("#cc241d"),
		Ok:           lipgloss.Color("#98971a"),
		Spectrum:     []string{"#8f3f1a", "#d65d0e", "#fabd2f", "#ebdbb2"},
	},
	{
		Name:      "Tokyo Night",
		BgBase:    lipgloss.Color("#1a1b26"),
		BgPanel:   lipgloss.Color("#16161e"),
		BgElevate: lipgloss.Color("#24283b"),
		BgHover:   lipgloss.Color("#24283b"),
		FgPrimary: lipgloss.Color("#c0caf5"),
		FgSub:     lipgloss.Color("#a9b1d6"),
		FgMuted:   lipgloss.Color("#565f89"),
		FgGhost:   lipgloss.Color("#3b4261"),
		BorderDim: lipgloss.Color("#24283b"),
		BorderMid: lipgloss.Color("#33415e"),
		SurfaceDeep: lipgloss.Color("#1f2335"),
		Accent:       lipgloss.Color("#7aa2f7"),
		AccentBright: lipgloss.Color("#7dcfff"),
		AccentDim:    lipgloss.Color("#3d59a1"),
		AccentFaint:  lipgloss.Color("#2f3a62"),
		Warm:         lipgloss.Color("#e0af68"),
		Danger:       lipgloss.Color("#f7768e"),
		Ok:           lipgloss.Color("#9ece6a"),
		Spectrum:     []string{"#2f3a62", "#3d59a1", "#7aa2f7", "#c0caf5"},
	},
	{
		Name:      "Rose Pine",
		BgBase:    lipgloss.Color("#191724"),
		BgPanel:   lipgloss.Color("#1f1d2e"),
		BgElevate: lipgloss.Color("#26233a"),
		BgHover:   lipgloss.Color("#26233a"),
		FgPrimary: lipgloss.Color("#e0def4"),
		FgSub:     lipgloss.Color("#c4a7e7"),
		FgMuted:   lipgloss.Color("#6e6a86"),
		FgGhost:   lipgloss.Color("#403d52"),
		BorderDim: lipgloss.Color("#26233a"),
		BorderMid: lipgloss.Color("#353154"),
		SurfaceDeep: lipgloss.Color("#2a273f"),
		Accent:       lipgloss.Color("#ebbcba"),
		AccentBright: lipgloss.Color("#f6c2c0"),
		AccentDim:    lipgloss.Color("#c4a7e7"),
		AccentFaint:  lipgloss.Color("#6e6a86"),
		Warm:         lipgloss.Color("#f6c177"),
		Danger:       lipgloss.Color("#eb6f92"),
		Ok:           lipgloss.Color("#31748f"),
		Spectrum:     []string{"#6e6a86", "#c4a7e7", "#ebbcba", "#e0def4"},
	},
	{
		Name:      "Spotify Premium",
		BgBase:    lipgloss.Color("#121212"),
		BgPanel:   lipgloss.Color("#181818"),
		BgElevate: lipgloss.Color("#282828"),
		BgHover:   lipgloss.Color("#282828"),
		FgPrimary: lipgloss.Color("#FFFFFF"),
		FgSub:     lipgloss.Color("#B3B3B3"),
		FgMuted:   lipgloss.Color("#727272"),
		FgGhost:   lipgloss.Color("#3E3E3E"),
		BorderDim: lipgloss.Color("#282828"),
		BorderMid: lipgloss.Color("#535353"),
		SurfaceDeep: lipgloss.Color("#1f1f1f"),
		Accent:       lipgloss.Color("#1DB954"), // Spotify Green
		AccentBright: lipgloss.Color("#1ED760"),
		AccentDim:    lipgloss.Color("#1aa34a"),
		AccentFaint:  lipgloss.Color("#14823a"),
		Warm:         lipgloss.Color("#c4b5fd"),
		Danger:       lipgloss.Color("#e91429"),
		Ok:           lipgloss.Color("#1DB954"),
		Spectrum:     []string{"#14823a", "#1DB954", "#1ED760", "#FFFFFF"},
	},
	{
		Name:      "Cyberpunk Neon",
		BgBase:    lipgloss.Color("#020813"),
		BgPanel:   lipgloss.Color("#051026"),
		BgElevate: lipgloss.Color("#0a204c"),
		BgHover:   lipgloss.Color("#0a204c"),
		FgPrimary: lipgloss.Color("#F3F4F6"),
		FgSub:     lipgloss.Color("#00FFFF"), // Neon Cyan
		FgMuted:   lipgloss.Color("#A5B4FC"),
		FgGhost:   lipgloss.Color("#1e293b"),
		BorderDim: lipgloss.Color("#0f172a"),
		BorderMid: lipgloss.Color("#FF007F"), // Neon Pink
		SurfaceDeep: lipgloss.Color("#08193a"),
		Accent:       lipgloss.Color("#FF007F"),
		AccentBright: lipgloss.Color("#FF3399"),
		AccentDim:    lipgloss.Color("#CC0066"),
		AccentFaint:  lipgloss.Color("#99004C"),
		Warm:         lipgloss.Color("#FFE600"), // Neon Yellow
		Danger:       lipgloss.Color("#EF4444"),
		Ok:           lipgloss.Color("#10B981"),
		Spectrum:     []string{"#00FFFF", "#FF007F", "#FFE600", "#FFFFFF"},
	},
	{
		Name:      "Retro Tape",
		BgBase:    lipgloss.Color("#FDF6E3"), // Solarized Base
		BgPanel:   lipgloss.Color("#EEE8D5"),
		BgElevate: lipgloss.Color("#E4DCD3"),
		BgHover:   lipgloss.Color("#E4DCD3"),
		FgPrimary: lipgloss.Color("#586E75"),
		FgSub:     lipgloss.Color("#657B83"),
		FgMuted:   lipgloss.Color("#93A1A1"),
		FgGhost:   lipgloss.Color("#D3C6A9"),
		BorderDim: lipgloss.Color("#D3C6A9"),
		BorderMid: lipgloss.Color("#CB4B16"), // Tape Orange
		SurfaceDeep: lipgloss.Color("#EFEAD4"),
		Accent:       lipgloss.Color("#CB4B16"),
		AccentBright: lipgloss.Color("#DC322F"),
		AccentDim:    lipgloss.Color("#B58900"), // Vintage Gold
		AccentFaint:  lipgloss.Color("#859900"),
		Warm:         lipgloss.Color("#B58900"),
		Danger:       lipgloss.Color("#DC322F"),
		Ok:           lipgloss.Color("#859900"),
		Spectrum:     []string{"#859900", "#B58900", "#CB4B16", "#586E75"},
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
	Accent       = ActiveTheme.Accent
	AccentBright = ActiveTheme.AccentBright
	AccentDim    = ActiveTheme.AccentDim
	AccentFaint  = ActiveTheme.AccentFaint
	Warm   = ActiveTheme.Warm
	Danger = ActiveTheme.Danger
	Ok     = ActiveTheme.Ok
	Sp0 = lipgloss.Color(ActiveTheme.Spectrum[0])
	Sp1 = lipgloss.Color(ActiveTheme.Spectrum[1])
	Sp2 = lipgloss.Color(ActiveTheme.Spectrum[2])
	Sp3 = lipgloss.Color(ActiveTheme.Spectrum[3])
)

var accentOverride string

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
	Accent = ActiveTheme.Accent
	AccentBright = ActiveTheme.AccentBright
	AccentDim = ActiveTheme.AccentDim
	AccentFaint = ActiveTheme.AccentFaint
	Warm = ActiveTheme.Warm
	Danger = ActiveTheme.Danger
	Ok = ActiveTheme.Ok
	Sp0 = lipgloss.Color(ActiveTheme.Spectrum[0])
	Sp1 = lipgloss.Color(ActiveTheme.Spectrum[1])
	Sp2 = lipgloss.Color(ActiveTheme.Spectrum[2])
	Sp3 = lipgloss.Color(ActiveTheme.Spectrum[3])

	if accentOverride != "" {
		c := lipgloss.Color(accentOverride)
		Terracotta = c
		Accent = c
	}
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

func SetAccentColor(hex string) {
	if hex == "" {
		accentOverride = ""
	} else {
		accentOverride = hex
	}
	RefreshStyles()
}

type VizColorMode int

const (
	VizThemeGradient VizColorMode = iota
	VizAccentSolid
	VizRainbow
	VizDualColor
)

func (v VizColorMode) String() string {
	switch v {
	case VizThemeGradient:
		return "Theme Gradient"
	case VizAccentSolid:
		return "Accent Solid"
	case VizRainbow:
		return "Rainbow"
	case VizDualColor:
		return "Dual Color"
	default:
		return "Theme Gradient"
	}
}

var AccentPresets = []string{
	"",
	"#C84B2F",
	"#cba6f7",
	"#88c0d0",
	"#a7c080",
	"#ff79c6",
	"#7aa2f7",
	"#d65d0e",
	"#f59e0b",
	"#10b981",
	"#f43f5e",
	"#1DB954",
	"#FF007F",
	"#CB4B16",
}

func hexToRGB(h string) [3]uint8 {
	var r, g, b uint8
	fmt.Sscanf(h, "#%02x%02x%02x", &r, &g, &b)
	return [3]uint8{r, g, b}
}
