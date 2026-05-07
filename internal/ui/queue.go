package ui

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type QueueView struct {
	Cursor  int
	Focused bool
}

func (q QueueView) View(queue []provider.Track) string {
	if len(queue) == 0 {
		if q.Focused {
			return "\n" + lipgloss.NewStyle().Foreground(Teal).Bold(true).Render("📋 Queue is empty") + "\n"
		}
		return ""
	}
	headerStyle := lipgloss.NewStyle().Foreground(FgMuted)
	if q.Focused {
		headerStyle = lipgloss.NewStyle().Foreground(Teal).Bold(true)
	}
	s := "\n" + headerStyle.Render("📋 Up Next:") + "\n"

	for i, t := range queue {
		prefix := "  "
		content := fmt.Sprintf("%d. %s - %s", i+1, t.Title, StyleMeta.Render(t.Artist))

		if i == q.Cursor {
			if q.Focused {
				prefix = StyleTitle.Render("▶ ")
				s += prefix + StyleSelected.Render(content) + "\n"
			} else {
				prefix = StyleMeta.Render("▶ ")
				s += prefix + StyleNormal.Render(content) + "\n"
			}
		} else {
			s += prefix + StyleNormal.Render(content) + "\n"
		}
	}
	return s
}
