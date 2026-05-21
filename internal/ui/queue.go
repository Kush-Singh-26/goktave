package ui

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/provider"
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

func (q *QueueView) View(queue []provider.Track, visibleHeight int, width int, isLiked func(string) bool) string {
	if len(queue) == 0 {
		return StyleMeta.Render("Queue is empty")
	}

	q.SyncScroll(visibleHeight)

	s := ""
	end := q.ScrollOffset + visibleHeight
	if end > len(queue) {
		end = len(queue)
	}

	rowWidth := width
	if rowWidth < 0 {
		rowWidth = 0
	}

	for i := q.ScrollOffset; i < end; i++ {
		t := queue[i]
		
		// Build premium badges
		var badges []string
		if isLiked != nil && isLiked(t.VideoID) {
			badges = append(badges, StyleBadgeLiked.Render("[LIKED]"))
		}
		if t.LocalPath != "" {
			badges = append(badges, StyleBadgeCached.Render("[CACHED]"))
		}
		badgeStr := ""
		if len(badges) > 0 {
			badgeStr = " " + strings.Join(badges, " ")
		}

		// Subtract space for left selection bar (3 chars) and badges
		badgeLen := lipgloss.Width(badgeStr)
		availableTextWidth := rowWidth - 8 - badgeLen // subtract index prefix too
		if availableTextWidth < 15 {
			availableTextWidth = 15
		}

		title := truncateText(t.Title, int(float64(availableTextWidth)*0.6))
		artist := truncateText(t.Artist, int(float64(availableTextWidth)*0.4))
		
		// Content with index
		content := fmt.Sprintf("%d. %s - %s", i+1, title, StyleMeta.Render(artist)) + badgeStr

		rowStyle := lipgloss.NewStyle().
			Width(rowWidth).
			MaxWidth(rowWidth)

		var line string
		if i == q.Cursor {
			if q.Focused {
				leftBar := StyleLeftHighlight.Render("┃ ")
				line = leftBar + StyleSelected.Copy().UnsetBackground().Render(content)
			} else {
				leftBar := StyleMeta.Render("│ ")
				line = leftBar + StyleNormal.Render(content)
			}
		} else {
			line = "  " + StyleNormal.Render(content)
		}
		s += rowStyle.Render(line) + "\n"
	}
	return s
}
