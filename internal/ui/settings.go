package ui

import (
	"fmt"
	"strings"

	"github.com/Kush-Singh-26/goktave/internal/engine"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type SettingsView struct {
	engine          engine.Engine
	Cursor          int
	ThemeIndex      int
	AccentPresetIdx int
}

func (v *SettingsView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if v.Cursor > 0 {
				v.Cursor--
			}
		case "down", "j":
			if v.Cursor < 4 {
				v.Cursor++
			}
		case "right", "l", "enter":
			switch v.Cursor {
			case 2:
				v.ThemeIndex = (v.ThemeIndex + 1) % len(Themes)
				v.applyTheme()
			case 3:
				v.AccentPresetIdx = (v.AccentPresetIdx + 1) % len(AccentPresets)
				v.applyAccent()
			case 4:
				cfg := v.engine.GetConfig()
				cfg.VizMode = (cfg.VizMode + 1) % 4
				_ = v.engine.SaveConfig()
			}
		case "left", "h":
			switch v.Cursor {
			case 2:
				v.ThemeIndex = (v.ThemeIndex - 1 + len(Themes)) % len(Themes)
				v.applyTheme()
			case 3:
				v.AccentPresetIdx = (v.AccentPresetIdx - 1 + len(AccentPresets)) % len(AccentPresets)
				v.applyAccent()
			case 4:
				cfg := v.engine.GetConfig()
				cfg.VizMode = (cfg.VizMode - 1 + 4) % 4
				_ = v.engine.SaveConfig()
			}
		case "+", "=":
			if v.Cursor == 0 {
				cfg := v.engine.GetConfig()
				cfg.MaxCacheSizeGB += 0.1
				_ = v.engine.SaveConfig()
			}
		case "-", "_":
			if v.Cursor == 0 {
				cfg := v.engine.GetConfig()
				if cfg.MaxCacheSizeGB > 0.1 {
					cfg.MaxCacheSizeGB -= 0.1
					_ = v.engine.SaveConfig()
				}
			}
		}
	}
	return nil
}

func (v *SettingsView) applyTheme() {
	ActiveTheme = &Themes[v.ThemeIndex]
	cfg := v.engine.GetConfig()
	cfg.Theme = ActiveTheme.Name
	if cfg.AccentOverride == "" {
		_ = v.engine.SaveConfig()
		RefreshStyles()
	} else {
		v.applyAccent()
	}
}

func (v *SettingsView) applyAccent() {
	cfg := v.engine.GetConfig()
	hex := AccentPresets[v.AccentPresetIdx]
	cfg.AccentOverride = hex
	_ = v.engine.SaveConfig()
	SetAccentColor(hex)
}

func (v SettingsView) View(width, height int, cacheLimit float64, cacheSize int64) string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, StyleTitle.Render("  Settings"))
	lines = append(lines, "")

	sizeStr := fmt.Sprintf("%.2f MB", float64(cacheSize)/(1024*1024))
	if cacheSize > 1024*1024*1024 {
		sizeStr = fmt.Sprintf("%.2f GB", float64(cacheSize)/(1024*1024*1024))
	}

	accentVal := "(theme default)"
	if v.AccentPresetIdx > 0 {
		accentVal = AccentPresets[v.AccentPresetIdx]
	}

	cfg := v.engine.GetConfig()
	vizMode := VizColorMode(cfg.VizMode)

	settings := []struct {
		Label string
		Value string
	}{
		{"Cache Limit (GB)", fmt.Sprintf("%.1f GB", cacheLimit)},
		{"Current Cache Size", sizeStr},
		{"UI Theme", Themes[v.ThemeIndex].Name},
		{"Accent Color", accentVal},
		{"Visualizer", vizMode.String()},
	}

	for i, s := range settings {
		cursor := "  "
		labelStyle := StyleNormal
		if i == v.Cursor {
			cursor = "> "
			labelStyle = StyleSelected
		}

		value := StyleMeta.Render(s.Value)

		if i == 3 && v.AccentPresetIdx > 0 {
			hex := AccentPresets[v.AccentPresetIdx]
			swatch := lipgloss.NewStyle().
				Background(lipgloss.Color(hex)).
				Foreground(lipgloss.Color("#ffffff")).
				Render("  ")
			value = swatch + " " + value
		}

		lines = append(lines, fmt.Sprintf("%s%-20s %s", cursor, labelStyle.Render(s.Label), value))
	}

	lines = append(lines, "")
	lines = append(lines, StyleMeta.Render("  [+] / [-] : Adjust cache limit"))
	lines = append(lines, StyleMeta.Render("  [X]       : Clear entire cache"))
	lines = append(lines, "")
	lines = append(lines, StyleMeta.Render("  [L] / [R] : Switch theme / accent / viz mode"))
	lines = append(lines, "")
	lines = append(lines, StyleMeta.Render("  Use arrow keys or j/k to move."))

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}
