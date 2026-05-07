package ui

import (
	"fmt"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	lipgloss "charm.land/lipgloss/v2"
)

type QueueView struct {
	Cursor       int
	ScrollOffset int
	Focused      bool
}

func (q *QueueView) Focus() {
	q.Focused = true
}

func (q *QueueView) Blur() {
	q.Focused = false
}

func (q *QueueView) SyncScroll(visibleHeight int) {
	if q.Cursor < q.ScrollOffset {
		q.ScrollOffset = q.Cursor
	} else if q.Cursor >= q.ScrollOffset+visibleHeight {
		q.ScrollOffset = q.Cursor - visibleHeight + 1
	}
}

func (q *QueueView) View(queue []provider.Track, visibleHeight int, width int) string {
	if len(queue) == 0 {
		return StyleMeta.Render("Queue is empty")
	}

	q.SyncScroll(visibleHeight)

	s := ""
	end := q.ScrollOffset + visibleHeight
	if end > len(queue) {
		end = len(queue)
	}

	rowWidth := width - 2
	if rowWidth < 0 {
		rowWidth = 0
	}

	for i := q.ScrollOffset; i < end; i++ {
		t := queue[i]
		cursor := " "
		content := fmt.Sprintf("%d. %s - %s", i+1, t.Title, StyleMeta.Render(t.Artist))

		rowStyle := lipgloss.NewStyle().
			Width(rowWidth).
			MaxWidth(rowWidth)

		if i == q.Cursor {
			if q.Focused {
				cursor = StyleTitle.Render("▶")
				line := " " + cursor + " " + StyleSelected.Copy().UnsetBackground().Render(content)
				s += rowStyle.Background(BgHover).Render(line) + "\n"
			} else {
				cursor = StyleMeta.Render("▶")
				line := " " + cursor + " " + StyleNormal.Render(content)
				s += rowStyle.Render(line) + "\n"
			}
		} else {
			s += rowStyle.Render("   " + StyleNormal.Render(content)) + "\n"
		}
	}
	return s
}
