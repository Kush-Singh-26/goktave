package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type ResultsList struct {
	tracks       []provider.Track
	cursor       int
	scrollOffset int
	focused      bool
}

func NewResultsList() ResultsList {
	return ResultsList{}
}

func (r *ResultsList) SetTracks(tracks []provider.Track) {
	r.tracks = tracks
	r.cursor = 0
	r.scrollOffset = 0
}

func (r *ResultsList) Focus() {
	r.focused = true
}

func (r *ResultsList) Blur() {
	r.focused = false
}

func (r *ResultsList) Update(msg tea.Msg, visibleHeight int) (ResultsList, tea.Cmd) {
	return *r, nil
}

func (r *ResultsList) Next() {
	if r.cursor < len(r.tracks)-1 {
		r.cursor++
	}
}

func (r *ResultsList) Prev() {
	if r.cursor > 0 {
		r.cursor--
	}
}

func (r *ResultsList) SyncScroll(visibleHeight int) {
	if r.cursor < r.scrollOffset {
		r.scrollOffset = r.cursor
	} else if r.cursor >= r.scrollOffset+visibleHeight {
		r.scrollOffset = r.cursor - visibleHeight + 1
	}
}

func (r *ResultsList) View(visibleHeight int, width int, isLiked func(string) bool) string {
	s := ""

	if len(r.tracks) == 0 {
		return StyleMeta.Render("No results found. Start searching!")
	}

	r.SyncScroll(visibleHeight)

	end := r.scrollOffset + visibleHeight
	if end > len(r.tracks) {
		end = len(r.tracks)
	}

	rowWidth := width
	if rowWidth < 0 {
		rowWidth = 0
	}

	for i := r.scrollOffset; i < end; i++ {
		track := r.tracks[i]
		
		// Build premium badges
		var badges []string
		if isLiked != nil && isLiked(track.VideoID) {
			badges = append(badges, StyleBadgeLiked.Render("[LIKED]"))
		}
		if track.LocalPath != "" {
			badges = append(badges, StyleBadgeCached.Render("[CACHED]"))
		}
		badgeStr := ""
		if len(badges) > 0 {
			badgeStr = " " + strings.Join(badges, " ")
		}

		// Subtract space for left selection bar (3 chars) and badges
		badgeLen := lipgloss.Width(badgeStr)
		availableTextWidth := rowWidth - 4 - badgeLen
		if availableTextWidth < 15 {
			availableTextWidth = 15
		}

		title := truncateText(track.Title, int(float64(availableTextWidth)*0.6))
		artist := truncateText(track.Artist, int(float64(availableTextWidth)*0.4))
		trackStr := fmt.Sprintf("%s • %s", title, artist) + badgeStr

		rowStyle := lipgloss.NewStyle().Width(rowWidth).MaxWidth(rowWidth)

		var line string
		if r.cursor == i {
			if r.focused {
				leftBar := StyleLeftHighlight.Render("┃ ")
				line = leftBar + StyleSelected.Copy().UnsetBackground().Render(trackStr)
			} else {
				leftBar := StyleMeta.Render("│ ")
				line = leftBar + StyleNormal.Render(trackStr)
			}
		} else {
			line = "  " + StyleNormal.Render(trackStr)
		}
		s += rowStyle.Render(line) + "\n"
	}
	return s
}

func (r ResultsList) GetSelected() *provider.Track {
	if len(r.tracks) == 0 || r.cursor < 0 || r.cursor >= len(r.tracks) {
		return nil
	}
	return &r.tracks[r.cursor]
}
