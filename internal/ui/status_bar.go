package ui

import (
	"fmt"
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
	prog := progress.New()
	prog.SetWidth(50)
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

func (b StatusBar) View(currentTrack *provider.Track, isPlaying bool) string {
	s := ""
	if b.status != "" {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80")).Bold(true).Render(b.status) + "\n"
	}

	if currentTrack != nil {
		elapsedStr := formatDuration(int(b.elapsed.Seconds()))
		totalStr := formatDuration(currentTrack.Duration)

		s += "\n" + b.progress.View() + "\n"
		s += StyleMuted.Render(fmt.Sprintf("%s / %s", elapsedStr, totalStr)) + "\n"
	}
	return s
}
