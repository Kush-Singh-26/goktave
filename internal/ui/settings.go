package ui

import (
	"fmt"
	"strings"

	"github.com/Kush-Singh-26/goktave/internal/engine"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type SettingsView struct {
	engine     engine.Engine
	Cursor     int
	ThemeIndex int
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
			if v.Cursor < 2 {
				v.Cursor++
			}
		case "right", "l", "enter":
			if v.Cursor == 2 {
				v.ThemeIndex = (v.ThemeIndex + 1) % len(Themes)
				ActiveTheme = &Themes[v.ThemeIndex]
				cfg := v.engine.GetConfig()
				cfg.Theme = ActiveTheme.Name
				_ = v.engine.SaveConfig()
				RefreshStyles()
			}
		case "left", "h":
			if v.Cursor == 2 {
				v.ThemeIndex = (v.ThemeIndex - 1 + len(Themes)) % len(Themes)
				ActiveTheme = &Themes[v.ThemeIndex]
				cfg := v.engine.GetConfig()
				cfg.Theme = ActiveTheme.Name
				_ = v.engine.SaveConfig()
				RefreshStyles()
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

func (v SettingsView) View(width, height int, cacheLimit float64, cacheSize int64) string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, StyleTitle.Render("  Settings"))
	lines = append(lines, "")

	sizeStr := fmt.Sprintf("%.2f MB", float64(cacheSize)/(1024*1024))
	if cacheSize > 1024*1024*1024 {
		sizeStr = fmt.Sprintf("%.2f GB", float64(cacheSize)/(1024*1024*1024))
	}

	settings := []struct {
		Label string
		Value string
	}{
		{"Cache Limit (GB)", fmt.Sprintf("%.1f GB", cacheLimit)},
		{"Current Cache Size", sizeStr},
		{"UI Theme", Themes[v.ThemeIndex].Name},
	}

	for i, s := range settings {
		style := StyleNormal
		if i == v.Cursor {
			style = StyleSelected
		}

		cursor := "  "
		if i == v.Cursor {
			cursor = "> "
		}

		lines = append(lines, fmt.Sprintf("%s%-20s %s", cursor, style.Render(s.Label), StyleMeta.Render(s.Value)))
	}

	lines = append(lines, "")
	lines = append(lines, StyleMeta.Render("  [+] / [-] : Adjust cache limit"))
	lines = append(lines, StyleMeta.Render("  [X]       : Clear entire cache"))
	lines = append(lines, "")
	lines = append(lines, StyleMeta.Render("  [Enter] / [L] / [R] on Theme: Switch themes"))
	lines = append(lines, "")
	lines = append(lines, StyleMeta.Render("  Use arrow keys or j/k to move."))

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}
