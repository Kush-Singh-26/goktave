package ui

import (
	"fmt"
	"github.com/Kush-Singh-26/goktave/internal/provider"
	tea "charm.land/bubbletea/v2"
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if r.cursor > 0 {
				r.cursor--
				if r.cursor < r.scrollOffset {
					r.scrollOffset = r.cursor
				}
			}
		case "down":
			if r.cursor < len(r.tracks)-1 {
				r.cursor++
				if r.cursor >= r.scrollOffset+visibleHeight {
					r.scrollOffset = r.cursor - visibleHeight + 1
				}
			}
		}
	}
	return *r, nil
}

func (r ResultsList) View(visibleHeight int) string {
	s := ""
	end := r.scrollOffset + visibleHeight
	if end > len(r.tracks) {
		end = len(r.tracks)
	}

	for i := r.scrollOffset; i < end; i++ {
		track := r.tracks[i]
		cursor := "  "
		trackStr := fmt.Sprintf("%s • %s", track.Title, track.Artist)

		if r.cursor == i {
			if r.focused {
				cursor = StyleTitle.Render("▶ ")
				s += cursor + StyleSelected.Render(trackStr) + "\n"
			} else {
				cursor = StyleMuted.Render("▶ ")
				s += cursor + StyleNormal.Render(trackStr) + "\n"
			}
		} else {
			s += cursor + StyleNormal.Render(trackStr) + "\n"
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
