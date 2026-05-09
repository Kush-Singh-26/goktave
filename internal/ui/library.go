package ui

import (
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"fmt"
	"github.com/Kush-Singh-26/goktave/internal/db"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type LibraryItemType int

const (
	ItemHeader LibraryItemType = iota
	ItemTrack
	ItemPlaylist
	ItemDownloadHeader
)

type LibraryItem struct {
	Type     LibraryItemType
	Label    string
	Track    *provider.Track
	Playlist string
}

type LibraryList struct {
	items        []LibraryItem
	cursor       int
	scrollOffset int
	focused      bool

	// State for nested view
	activePlaylist string
}

func NewLibraryList() LibraryList {
	return LibraryList{}
}

func (l *LibraryList) SetItems(liked []provider.Track, history []provider.Track, playlists []db.Playlist, playlistTracks []provider.Track, downloaded []provider.Track) {
	var items []LibraryItem

	if l.activePlaylist != "" {
		items = append(items, LibraryItem{Type: ItemHeader, Label: "📁 Playlist: " + l.activePlaylist})
		items = append(items, LibraryItem{Type: ItemPlaylist, Label: "  [ Back to Playlists ]", Playlist: "BACK"})
		for _, t := range playlistTracks {
			track := t
			items = append(items, LibraryItem{Type: ItemTrack, Track: &track})
		}
	} else {
		if len(playlists) > 0 {
			items = append(items, LibraryItem{Type: ItemHeader, Label: "Playlists"})
			for _, p := range playlists {
				items = append(items, LibraryItem{Type: ItemPlaylist, Label: "  " + p.Name, Playlist: p.Name})
			}
		}

		if len(downloaded) > 0 {
			items = append(items, LibraryItem{Type: ItemHeader, Label: "📥 Downloads"})
			for _, t := range downloaded {
				track := t
				items = append(items, LibraryItem{Type: ItemTrack, Track: &track})
			}
		}

		if len(liked) > 0 {
			items = append(items, LibraryItem{Type: ItemHeader, Label: "❤️ Liked Songs"})
			for _, t := range liked {
				track := t
				items = append(items, LibraryItem{Type: ItemTrack, Track: &track})
			}
		}

		if len(history) > 0 {
			items = append(items, LibraryItem{Type: ItemHeader, Label: "🕒 Recent History"})
			for _, t := range history {
				track := t
				items = append(items, LibraryItem{Type: ItemTrack, Track: &track})
			}
		}
	}

	if len(items) == 0 {
		items = append(items, LibraryItem{Type: ItemHeader, Label: "No activity yet. Start listening!"})
	}

	l.items = items
	// Reset cursor if it's out of bounds or on a header
	if l.cursor >= len(l.items) {
		l.cursor = 0
	}
	l.ensureValidCursor()
}

func (l *LibraryList) ensureValidCursor() {
	if len(l.items) == 0 {
		return
	}
	// If cursor is on a header, move to next track if possible
	if l.items[l.cursor].Type == ItemHeader {
		l.Next()
	}
}

func (l *LibraryList) Focus() {
	l.focused = true
}

func (l *LibraryList) Blur() {
	l.focused = false
}

func (l *LibraryList) Update(msg tea.Msg, visibleHeight int) (LibraryList, tea.Cmd) {
	return *l, nil
}

func (l *LibraryList) Next() {
	if len(l.items) == 0 {
		return
	}
	start := l.cursor
	for {
		l.cursor++
		if l.cursor >= len(l.items) {
			l.cursor = 0
		}
		if l.items[l.cursor].Type != ItemHeader || l.cursor == start {
			break
		}
	}
}

func (l *LibraryList) Prev() {
	if len(l.items) == 0 {
		return
	}
	start := l.cursor
	for {
		l.cursor--
		if l.cursor < 0 {
			l.cursor = len(l.items) - 1
		}
		if l.items[l.cursor].Type != ItemHeader || l.cursor == start {
			break
		}
	}
}

func (l *LibraryList) SyncScroll(visibleHeight int) {
	if l.cursor < l.scrollOffset {
		l.scrollOffset = l.cursor
	} else if l.cursor >= l.scrollOffset+visibleHeight {
		l.scrollOffset = l.cursor - visibleHeight + 1
	}
}

func (l *LibraryList) View(visibleHeight int, width int) string {
	if len(l.items) == 0 {
		return "  " + StyleMeta.Render("Loading library...")
	}

	l.SyncScroll(visibleHeight)
	s := ""
	end := l.scrollOffset + visibleHeight
	if end > len(l.items) {
		end = len(l.items)
	}

	rowWidth := width
	if rowWidth < 0 {
		rowWidth = 0
	}

	for i := l.scrollOffset; i < end; i++ {
		item := l.items[i]
		rowStyle := lipgloss.NewStyle().Width(rowWidth).MaxWidth(rowWidth)

		if item.Type == ItemHeader {
			s += "\n " + StyleTitle.Render(item.Label) + "\n"
			continue
		}

		var content string
		if item.Type == ItemTrack {
			track := item.Track
			content = fmt.Sprintf("%s • %s", track.Title, track.Artist)
			if track.LocalPath != "" {
				content += " " + StyleMeta.Render("✔")
			}
		} else {
			content = item.Label
		}

		cursor := " "

		if l.cursor == i {
			if l.focused {
				cursor = StyleTitle.Render("▶")
				line := " " + cursor + " " + StyleSelected.Copy().UnsetBackground().Render(content)
				s += rowStyle.Render(line) + "\n"
			} else {
				cursor = StyleMeta.Render("▶")
				line := " " + cursor + " " + StyleNormal.Render(content)
				s += rowStyle.Render(line) + "\n"
			}
		} else {
			s += rowStyle.Render("   "+StyleNormal.Render(content)) + "\n"
		}
	}
	return s
}

func (l *LibraryList) GetSelected() *provider.Track {
	if l.cursor < 0 || l.cursor >= len(l.items) {
		return nil
	}
	return l.items[l.cursor].Track
}

func (l *LibraryList) GetSelectedPlaylist() string {
	if l.cursor < 0 || l.cursor >= len(l.items) {
		return ""
	}
	return l.items[l.cursor].Playlist
}
