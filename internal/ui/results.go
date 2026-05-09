package ui

import (
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"fmt"
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

func (r *ResultsList) View(visibleHeight int, width int) string {
	s := ""

	if len(r.tracks) == 0 {
		return StyleMeta.Render("No results found. Start searching!")
	}

	r.SyncScroll(visibleHeight)

	end := r.scrollOffset + visibleHeight
	if end > len(r.tracks) {
		end = len(r.tracks)
	}

	// Width is already the inner pane width
	rowWidth := width
	if rowWidth < 0 {
		rowWidth = 0
	}

	for i := r.scrollOffset; i < end; i++ {
		track := r.tracks[i]
		cursor := " "
		trackStr := fmt.Sprintf("%s • %s", track.Title, track.Artist)
		if track.LocalPath != "" {
			trackStr += " " + StyleMeta.Render("✔")
		}

		// Base style for all rows to ensure background consistency
		rowStyle := lipgloss.NewStyle().
			Width(rowWidth).
			MaxWidth(rowWidth)

		if r.cursor == i {
			if r.focused {
				cursor = StyleTitle.Render("▶")
				line := " " + cursor + " " + StyleSelected.Copy().UnsetBackground().Render(trackStr)
				s += rowStyle.Render(line) + "\n"
			} else {
				cursor = StyleMeta.Render("▶")
				line := " " + cursor + " " + StyleNormal.Render(trackStr)
				s += rowStyle.Render(line) + "\n"
			}
		} else {
			s += rowStyle.Render("   "+StyleNormal.Render(trackStr)) + "\n"
		}
	}
	return s
}

func (r ResultsList) GetSelected() *provider.Track {
	if len(r.tracks) == 0 || r.cursor < 0 || r.cursor >= len(r.tracks) {
		return nil
	}
	return &r.tracks[r.cursor]
}
