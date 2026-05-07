package ui

import (
	"fmt"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type QueueView struct{}

func (q QueueView) View(queue []provider.Track) string {
	if len(queue) == 0 {
		return ""
	}

	s := "\n" + StyleTitle.Render("📋 Up Next:") + "\n"
	displayLimit := 5
	if len(queue) < displayLimit {
		displayLimit = len(queue)
	}
	for i := 0; i < displayLimit; i++ {
		t := queue[i]
		s += fmt.Sprintf("  %d. %s - %s\n", i+1, t.Title, StyleMuted.Render(t.Artist))
	}
	if len(queue) > displayLimit {
		s += StyleMuted.Render(fmt.Sprintf("  ... and %d more in queue", len(queue)-displayLimit)) + "\n"
	}
	return s
}
