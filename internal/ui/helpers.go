package ui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/Kush-Singh-26/goktave/internal/provider"
)

func (m Model) cancelRunningTask() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m Model) isLibraryTab() bool {
	return m.activeTab == TabPlaylists || m.activeTab == TabLiked || m.activeTab == TabHistory || m.activeTab == TabDownloads
}

func (m *Model) getActiveLibrary() *LibraryList {
	switch m.activeTab {
	case TabPlaylists:
		return &m.playlists
	case TabLiked:
		return &m.liked
	case TabHistory:
		return &m.history
	case TabDownloads:
		return &m.downloads
	}
	return nil
}

func (m Model) renderLibrary(height, width int) string {
	switch m.activeTab {
	case TabPlaylists:
		return m.playlists.View(height, width, m.engine.IsLiked)
	case TabLiked:
		return m.liked.View(height, width, m.engine.IsLiked)
	case TabHistory:
		return m.history.View(height, width, m.engine.IsLiked)
	case TabDownloads:
		return m.downloads.View(height, width, m.engine.IsLiked)
	}
	return ""
}

func (m Model) getVisibleHeight() int {
	headerBoxHeight := 4
	headerHeight := headerBoxHeight + HeaderStyle.GetVerticalFrameSize()
	helpHeight := 1
	if m.help.ShowAll && m.terminalHeight >= 20 {
		helpHeight = 7
	}

	available := m.terminalHeight - headerHeight - helpHeight
	if available < 2 {
		available = 2
	}
	return available
}

func (m *Model) refreshLibrary() {
	// Always keep the liked list up to date in memory
	l, _ := m.engine.GetLikedTracks()
	m.liked.SetItems(l, nil, nil, nil, nil)

	switch m.activeTab {
	case TabHistory:
		h, _ := m.engine.GetHistory(100)
		m.history.SetItems(nil, h, nil, nil, nil)
	case TabDownloads:
		d, _ := m.engine.GetDownloadedTracks()
		m.downloads.SetItems(nil, nil, nil, nil, d)
	case TabPlaylists:
		p, _ := m.engine.GetPlaylists()
		var pt []provider.Track
		if m.playlists.activePlaylist != "" {
			pt, _ = m.engine.GetPlaylistTracks(m.playlists.activePlaylist)
		}
		m.playlists.SetItems(nil, nil, p, pt, nil)
	}
}

func (m *Model) refreshResults() {
	for i, t := range m.results.tracks {
		if dbTrack, err := m.engine.GetTrack(t.VideoID); err == nil {
			m.results.tracks[i].LocalPath = dbTrack.LocalPath
		}
	}
}

func (m *Model) blurAll() {
	m.search.Blur()
	m.results.Blur()
	m.playlists.Blur()
	m.liked.Blur()
	m.history.Blur()
	m.downloads.Blur()
	m.queue.Blur()
}

func (m Model) placeOverlay(base string, overlay string) string {
	return lipgloss.Place(m.terminalWidth, m.terminalHeight,
		lipgloss.Center, lipgloss.Center,
		overlay,
	)
}

func (m Model) getSelectedTrack() *provider.Track {
	if m.focusArea == AreaContent {
		if m.activeTab == TabResults {
			return m.results.GetSelected()
		} else if m.isLibraryTab() {
			if lib := m.getActiveLibrary(); lib != nil {
				return lib.GetSelected()
			}
		}
	} else if m.focusArea == AreaQueue {
		q := m.engine.GetQueue()
		if m.queue.Cursor >= 0 && m.queue.Cursor < len(q) {
			return &q[m.queue.Cursor]
		}
	}
	return nil
}

func clampLines(text string, maxLines int) string {
	if maxLines <= 0 || text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n")
}

func (m Model) suggestCmds(msg tea.Msg, oldVal string) []tea.Cmd {
	if !m.search.Focused() {
		return nil
	}
	if _, ok := msg.(tea.KeyMsg); !ok {
		return nil
	}
	if m.search.Value() != "" {
		if m.search.Value() != oldVal {
			m.suggestionIndex = -1
		}
		return []tea.Cmd{func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			res, err := m.engine.GetSuggestions(ctx, m.search.Value())
			return SuggestionsMsg{Suggestions: res, Err: err}
		}}
	}
	return []tea.Cmd{func() tea.Msg {
		res, _ := m.engine.GetSearchHistory(5)
		return SearchHistoryMsg{History: res}
	}}
}
