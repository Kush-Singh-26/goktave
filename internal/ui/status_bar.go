package ui

import (
	"fmt"
	"image/color"
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
	vizMode  VizColorMode
	peaks    []float64 // Pre-allocated peaks for falling gravity dots
}

func NewStatusBar() StatusBar {
	prog := progress.New(
		progress.WithoutPercentage(),
	)
	prog.SetWidth(40)
	return StatusBar{
		progress: prog,
		peaks:    make([]float64, 1000), // Supports up to 1000 columns without reallocation
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

	elapsedStyle := lipgloss.NewStyle().Foreground(AccentDim)

	statusIcon := "> "
	switch state {
	case player.StatePaused:
		statusIcon = "|| "
	case player.StateBuffering:
		statusIcon = ".. "
	case player.StateStopped:
		statusIcon = "[] "
	}

	// Calculate exact progWidth taking statusIcon length and padding spaces into account!
	// Total width of non-prog elements is: len(statusIcon) + len(elapsedStr) + 2 (spaces around prog) + len(totalStr)
	progWidth := width - len(statusIcon) - len(elapsedStr) - len(totalStr) - 2
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

	// Dynamic sleek progress bar with a circle seeker "●"
	var prog string
	fullCells := int(float64(progWidth) * pct)
	if fullCells > progWidth {
		fullCells = progWidth
	}
	
	if fullCells > 0 {
		for i := 0; i < fullCells-1; i++ {
			cellPct := float64(i) / float64(progWidth)
			c := GetGradientColor(cellPct)
			prog += lipgloss.NewStyle().Foreground(c).Render("━")
		}
		lastColor := GetGradientColor(float64(fullCells) / float64(progWidth))
		prog += lipgloss.NewStyle().Foreground(lastColor).Bold(true).Render("●")
		
		emptyCells := progWidth - fullCells
		if emptyCells > 0 {
			prog += lipgloss.NewStyle().Foreground(SurfaceDeep).Render(strings.Repeat("─", emptyCells))
		}
	} else {
		prog += lipgloss.NewStyle().Foreground(AccentBright).Bold(true).Render("●")
		emptyCells := progWidth - 1
		if emptyCells > 0 {
			prog += lipgloss.NewStyle().Foreground(SurfaceDeep).Render(strings.Repeat("─", emptyCells))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		elapsedStyle.Render(statusIcon),
		elapsedStyle.Render(elapsedStr),
		" "+prog+" ",
		elapsedStyle.Render(totalStr),
	)
}

func (b *StatusBar) SetVizMode(mode VizColorMode) {
	b.vizMode = mode
}

func (b StatusBar) VisualizerView(bars []float64, width int, height int) string {
	if height <= 0 {
		height = 3
	}
	fullBlocks := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	symBars := make([]float64, width)
	center := width / 2
	for i := 0; i < center; i++ {
		idx := int(float64(center-i-1) / float64(center) * float64(len(bars)))
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		val := bars[idx]
		symBars[i] = val
		if center+i < width {
			symBars[center+i] = val
		}
	}

	// Update visualizer peaks and apply falling decay
	for i := 0; i < width; i++ {
		val := 0.0
		if i < len(symBars) {
			val = symBars[i]
		}
		if val > b.peaks[i] {
			b.peaks[i] = val
		} else {
			b.peaks[i] -= 0.02 // Gravity rate of peak decay
			if b.peaks[i] < 0 {
				b.peaks[i] = 0
			}
		}
	}

	lines := make([]string, height)
	for h := 0; h < height; h++ {
		line := ""
		threshold := float64(height-h-1) / float64(height)
		nextThreshold := float64(height-h) / float64(height)

		for i, v := range symBars {
			var barColor color.Color
			switch b.vizMode {
			case VizAccentSolid:
				barColor = Accent
			case VizRainbow:
				angle := float64(i) / float64(width)
				barColor = rainbowColor(angle)
			case VizDualColor:
				if i < center {
					barColor = AccentBright
				} else {
					barColor = AccentDim
				}
			default:
				dist := math.Abs(float64(i-center)) / float64(center)
				barColor = GetGradientColor(1.0 - dist)
			}

			cellFill := (v - threshold) * float64(height)
			if cellFill > 0 {
				if cellFill >= 1 {
					cellFill = 1
				}
				idx := int(cellFill * float64(len(fullBlocks)-1))
				char := fullBlocks[idx]
				line += lipgloss.NewStyle().Foreground(barColor).Render(char)
			} else {
				// Falling peak decay dot logic
				peakVal := b.peaks[i]
				if peakVal >= threshold && peakVal < nextThreshold && peakVal > 0.02 {
					line += lipgloss.NewStyle().Foreground(barColor).Bold(true).Render("·")
				} else {
					line += " "
				}
			}
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

func rainbowColor(angle float64) color.Color {
	// Generate rainbow colors cycling through hues
	hue := int(angle * 360)
	r := uint8(0)
	g := uint8(0)
	b := uint8(0)
	region := hue / 60 % 6
	local := hue % 60
	switch region {
	case 0:
		r, g, b = 255, uint8(local*255/60), 0
	case 1:
		r, g, b = uint8((60-local)*255/60), 255, 0
	case 2:
		r, g, b = 0, 255, uint8(local*255/60)
	case 3:
		r, g, b = 0, uint8((60-local)*255/60), 255
	case 4:
		r, g, b = uint8(local*255/60), 0, 255
	case 5:
		r, g, b = 255, 0, uint8((60-local)*255/60)
	}
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func (b StatusBar) View(currentTrack *provider.Track, state player.State, bars []float64) string {
	// This method is kept for compatibility or can be removed if not needed.
	// We'll use PlaybarView and VisualizerView separately in Model.View.
	return b.PlaybarView(currentTrack, state, 100)
}
