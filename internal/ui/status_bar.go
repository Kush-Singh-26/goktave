package ui

import (
	"fmt"
	"strings"
	"time"

	progress "charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
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

func (b StatusBar) PlaybarView(currentTrack *provider.Track, isPlaying bool, width int) string {
	if currentTrack == nil {
		if b.status != "" {
			return lipgloss.NewStyle().Height(3).Align(lipgloss.Left, lipgloss.Center).Render(
				lipgloss.NewStyle().Foreground(Ok).Bold(true).Render("  " + b.status),
			)
		}
		return lipgloss.NewStyle().Height(3).Align(lipgloss.Left, lipgloss.Center).Render(
			"  " + StyleMeta.Render("No track playing"),
		)
	}

	elapsedStr := formatDuration(int(b.elapsed.Seconds()))
	totalStr := formatDuration(currentTrack.Duration)

	title := currentTrack.Title
	artist := currentTrack.Artist
	maxTextWidth := width - 4
	if len(title)+len(artist)+5 > maxTextWidth {
		if len(title) > maxTextWidth/2 {
			title = title[:maxTextWidth/2-1] + "…"
		}
		if len(artist) > maxTextWidth/2 {
			artist = artist[:maxTextWidth/2-1] + "…"
		}
	}

	trackInfo := lipgloss.NewStyle().Foreground(TerracottaBright).Bold(true).Render(fmt.Sprintf("%s — %s", title, artist))

	progWidth := width - len(elapsedStr) - len(totalStr) - 6
	if progWidth < 5 {
		progWidth = 5
	}

	pct := b.progress.Percent()
	if pct < 0 { pct = 0 }
	if pct > 1 { pct = 1 }

	fullWidth := float64(progWidth) * pct
	fullCells := int(fullWidth)
	remainder := fullWidth - float64(fullCells)
	smoothBlocks := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	charIdx := int(remainder * 8)

	filled := lipgloss.NewStyle().Foreground(Terracotta).Render(strings.Repeat("█", fullCells))
	lastChar := ""
	if fullCells < progWidth {
		lastChar = lipgloss.NewStyle().Foreground(TerracottaBright).Render(smoothBlocks[charIdx])
	}
	emptyCells := progWidth - fullCells
	if lastChar != "" { emptyCells-- }
	track := lipgloss.NewStyle().Foreground(SurfaceDeep).Render(strings.Repeat("─", emptyCells))
	prog := filled + lastChar + track

	statusText := ""
	if b.status != "" {
		statusText = lipgloss.NewStyle().Foreground(Ok).Render("● " + b.status)
	} else {
		statusText = lipgloss.NewStyle().Foreground(FgSub).Render("○ Ready")
	}

	elapsedStyle := lipgloss.NewStyle().Foreground(TerracottaDim)
	
	// Stacked Layout
	return lipgloss.JoinVertical(lipgloss.Left,
		" "+statusText,
		" "+trackInfo,
		" "+elapsedStyle.Render(elapsedStr)+" "+prog+" "+elapsedStyle.Render(totalStr),
	)
}

func (b StatusBar) VisualizerView(bars []float64, width int, height int) string {
	if height <= 0 {
		height = 3
	}
	// Multi-level blocks for vertical resolution
	// We'll render from top to bottom
	fullBlocks := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	
	lines := make([]string, height)
	for h := 0; h < height; h++ {
		line := ""
		threshold := float64(height-h-1) / float64(height)
		
		for i, v := range bars {
			// Calculate how much of this specific cell is filled
			// relative to the current row 'h'
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

			color := GetGradientColor(float64(i) / float64(len(bars)))
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

func (b StatusBar) View(currentTrack *provider.Track, isPlaying bool, bars []float64) string {
	// This method is kept for compatibility or can be removed if not needed.
	// We'll use PlaybarView and VisualizerView separately in Model.View.
	return b.PlaybarView(currentTrack, isPlaying, 100)
}
