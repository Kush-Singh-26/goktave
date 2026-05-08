package ui

import (
	"math"
	"strings"
	"time"

	progress "charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type StatusBar struct {
	progress progress.Model
	status   string
	elapsed  time.Duration
	lastTick time.Time
}

func NewStatusBar() StatusBar {
	prog := progress.New(
		progress.WithoutPercentage(),
	)
	prog.SetWidth(40)
	return StatusBar{
		progress: prog,
	}
}

func (b *StatusBar) SetStatus(s string) {
	b.status = s
}

func (b *StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case progress.FrameMsg:
		var newProg progress.Model
		newProg, cmd = b.progress.Update(msg)
		b.progress = newProg
	}
	return *b, cmd
}

func (b StatusBar) PlaybarView(currentTrack *provider.Track, state player.State, width int) string {
	if currentTrack == nil {
		return ""
	}

	elapsedStr := formatDuration(int(b.elapsed.Seconds()))
	totalStr := formatDuration(currentTrack.Duration)

	progWidth := width - len(elapsedStr) - len(totalStr) - 4
	if progWidth < 5 {
		progWidth = 5
	}

	pct := 0.0
	if currentTrack.Duration > 0 {
		pct = float64(b.elapsed.Seconds()) / float64(currentTrack.Duration)
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	fullWidth := float64(progWidth) * pct
	fullCells := int(fullWidth)
	remainder := fullWidth - float64(fullCells)
	smoothBlocks := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	charIdx := int(remainder * 8)
	if charIdx >= len(smoothBlocks) {
		charIdx = len(smoothBlocks) - 1
	}

	filled := lipgloss.NewStyle().Foreground(Terracotta).Render(strings.Repeat("█", fullCells))
	lastChar := ""
	if fullCells < progWidth {
		lastChar = lipgloss.NewStyle().Foreground(TerracottaBright).Render(smoothBlocks[charIdx])
	}
	emptyCells := progWidth - fullCells
	if lastChar != "" {
		emptyCells--
	}
	if emptyCells < 0 {
		emptyCells = 0
	}
	track := lipgloss.NewStyle().Foreground(SurfaceDeep).Render(strings.Repeat("─", emptyCells))
	prog := filled + lastChar + track

	elapsedStyle := lipgloss.NewStyle().Foreground(TerracottaDim)

	statusIcon := "▶ "
	switch state {
	case player.StatePaused:
		statusIcon = "⏸ "
	case player.StateBuffering:
		statusIcon = "⏳ "
	case player.StateStopped:
		statusIcon = "■ "
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		elapsedStyle.Render(statusIcon),
		elapsedStyle.Render(elapsedStr),
		" "+prog+" ",
		elapsedStyle.Render(totalStr),
	)
}

func (b StatusBar) VisualizerView(bars []float64, width int, height int) string {
	if height <= 0 {
		height = 3
	}
	// Multi-level blocks for vertical resolution
	fullBlocks := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	
	// Create symmetrical bars
	symBars := make([]float64, width)
	center := width / 2
	for i := 0; i < center; i++ {
		// Map indices to bars
		idx := int(float64(center-i-1) / float64(center) * float64(len(bars)))
		if idx >= len(bars) { idx = len(bars) - 1 }
		val := bars[idx]
		symBars[i] = val
		if center+i < width {
			symBars[center+i] = val
		}
	}

	lines := make([]string, height)
	for h := 0; h < height; h++ {
		line := ""
		threshold := float64(height-h-1) / float64(height)
		
		for i, v := range symBars {
			cellFill := (v - threshold) * float64(height)
			if cellFill <= 0 {
				line += " "
				continue
			}
			if cellFill >= 1 {
				cellFill = 1
			}
			
			idx := int(cellFill * float64(len(fullBlocks)-1))
			char := fullBlocks[idx]

			// Color based on distance from center for a cool effect
			dist := math.Abs(float64(i-center)) / float64(center)
			color := GetGradientColor(1.0 - dist)
			line += lipgloss.NewStyle().Foreground(color).Render(char)
		}
		lines[h] = line
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Bottom).
		Render(content)
}

func (b StatusBar) View(currentTrack *provider.Track, state player.State, bars []float64) string {
	// This method is kept for compatibility or can be removed if not needed.
	// We'll use PlaybarView and VisualizerView separately in Model.View.
	return b.PlaybarView(currentTrack, state, 100)
}
