package ui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Kush-Singh-26/goktave/internal/player"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle Playlist Popups FIRST
	if m.showPlaylistPrompt {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.showPlaylistPrompt = false
				m.playlistPrompt.Blur()
				m.playlistPrompt.SetValue("")
				return m, nil
			case "enter":
				name := m.playlistPrompt.Value()
				if name != "" {
					_ = m.engine.CreatePlaylist(name)
					m.statusBar.SetStatus("Playlist created: " + name)
					m.showPlaylistPrompt = false
					m.playlistPrompt.Blur()
					m.playlistPrompt.SetValue("")
					if m.isLibraryTab() {
						m.refreshLibrary()
					}
				}
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.playlistPrompt, cmd = m.playlistPrompt.Update(msg)
		return m, cmd
	}

	if m.showPlaylistSelector {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.showPlaylistSelector = false
				return m, nil
			case "up", "k":
				if m.playlistSelectorIndex > 0 {
					m.playlistSelectorIndex--
				}
				return m, nil
			case "down", "j":
				if m.playlistSelectorIndex < len(m.allPlaylists)-1 {
					m.playlistSelectorIndex++
				}
				return m, nil
			case "enter":
				if m.playlistSelectorIndex >= 0 && m.playlistSelectorIndex < len(m.allPlaylists) {
					p := m.allPlaylists[m.playlistSelectorIndex]
					_ = m.engine.AddTrackToPlaylist(p.Name, m.trackToAddToPlaylist.VideoID)
					m.statusBar.SetStatus("Added to playlist: " + p.Name)
					m.showPlaylistSelector = false
				}
				return m, nil
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
	case tea.KeyMsg:
		cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			return m, cmd
		}
	case SearchResultsMsg:
		m.handleSearchResults(msg)
	case SuggestionsMsg:
		m.handleSuggestions(msg)
	case SearchHistoryMsg:
		m.handleSearchHistory(msg)
	case ThumbnailMsg:
		m.handleThumbnail(msg)
	case tickMsg:
		cmds = append(cmds, m.handleTick()...)
	}

	var cmd tea.Cmd
	oldVal := m.search.Value()
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)

	cmds = append(cmds, m.suggestCmds(msg, oldVal)...)

	m.results, cmd = m.results.Update(msg, m.getVisibleHeight())
	cmds = append(cmds, cmd)

	m.playlists, cmd = m.playlists.Update(msg, m.getVisibleHeight())
	cmds = append(cmds, cmd)
	m.liked, cmd = m.liked.Update(msg, m.getVisibleHeight())
	cmds = append(cmds, cmd)
	m.history, cmd = m.history.Update(msg, m.getVisibleHeight())
	cmds = append(cmds, cmd)
	m.downloads, cmd = m.downloads.Update(msg, m.getVisibleHeight())
	cmds = append(cmds, cmd)

	m.statusBar, cmd = m.statusBar.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.terminalWidth = msg.Width
	m.terminalHeight = msg.Height

	wide := m.terminalWidth >= 120
	narrow := m.terminalWidth < 80

	searchWidth := m.terminalWidth - 10
	if wide {
		searchWidth = int(float64(m.terminalWidth-15) * 0.55)
	} else if narrow {
		searchWidth = m.terminalWidth - 10
	} else {
		searchWidth = m.terminalWidth - 16
	}
	if searchWidth < 15 {
		searchWidth = 15
	}
	m.search.SetWidth(searchWidth - 4)

	visibleHeight := m.getVisibleHeight()
	paneFrameV := PaneStyle.GetVerticalFrameSize()
	vizOuterHeight := 4 + paneFrameV
	topHeightOuter := visibleHeight - vizOuterHeight
	minPaneContent := 4
	if topHeightOuter < 1 || visibleHeight-vizOuterHeight < minPaneContent {
		topHeightOuter = visibleHeight
		vizOuterHeight = 0
	}
	topInnerHeight := topHeightOuter - paneFrameV
	if topInnerHeight < 1 {
		topInnerHeight = 1
	}
	contentWidth := m.terminalWidth - 4

	var mainWidth int
	if wide {
		mainWidth = int(float64(contentWidth) * 0.55)
	} else if !narrow {
		mainWidth = int(float64(contentWidth) * 0.75)
	} else {
		mainWidth = contentWidth
	}
	contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
	if contentInnerWidth < 1 {
		contentInnerWidth = 1
	}
	lyricsHeight := topInnerHeight - 3
	if lyricsHeight < 1 {
		lyricsHeight = 1
	}
	m.lyrics.SetWidth(contentInnerWidth)
	m.lyrics.SetHeight(lyricsHeight)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// 1. Absolute Global Keys (Always work)
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "tab":
		if m.search.Focused() && len(m.suggestions) > 0 {
			idx := m.suggestionIndex
			if idx < 0 {
				idx = 0
			}
			m.search.SetValue(m.suggestions[idx])
			m.showSuggest = false
			m.suggestionIndex = -1
			return nil
		}
		m.focusArea = (m.focusArea + 1) % 3
		if m.focusArea == AreaSearch {
			m.search.Focus()
			m.queue.Blur()
			m.results.Blur()
		} else if m.focusArea == AreaQueue {
			m.search.Blur()
			m.queue.Focus()
			m.results.Blur()
		} else {
			m.search.Blur()
			m.queue.Blur()
			m.results.Focus()
		}
		return nil
	}

	// 2. Search Focused Logic (Navigation within suggestions)
	if m.search.Focused() {
		switch msg.String() {
		case "esc":
			m.search.Blur()
			m.showSuggest = false
			m.suggestionIndex = -1
			m.focusArea = AreaContent
			m.results.Focus()
			return nil
		case "up":
			if len(m.suggestions) > 0 {
				if m.suggestionIndex > 0 {
					m.suggestionIndex--
				} else {
					m.suggestionIndex = len(m.suggestions) - 1
				}
				return nil
			}
		case "down":
			if len(m.suggestions) > 0 {
				if m.suggestionIndex < len(m.suggestions)-1 {
					m.suggestionIndex++
				} else {
					m.suggestionIndex = 0
				}
				return nil
			}
		case "enter":
			query := m.search.Value()
			if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
				query = m.suggestions[m.suggestionIndex]
				m.search.SetValue(query)
			}

			if query != "" {
				m.loading = true
				m.err = nil
				m.showSuggest = false
				m.suggestionIndex = -1
				m.statusBar.SetStatus("Searching...")
				m.search.Blur()
				m.focusArea = AreaContent
				m.cancelRunningTask()
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel

				return func() tea.Msg {
					res, err := m.engine.Search(ctx, query)
					return SearchResultsMsg{Results: res, Err: err}
				}
			}
		}
		return nil
	}

	// 3. Navigation & Command keys (Only when NOT typing)
	switch msg.String() {
	case "l": // Toggle Like
		// Prefer the highlighted/selected track; fall back to the currently playing one
		track := m.getSelectedTrack()
		if track == nil {
			track = m.engine.GetCurrentTrack()
		}
		if track != nil {
			liked, _ := m.engine.ToggleLike(track.VideoID)
			if liked {
				m.statusBar.SetStatus("♥ Liked: " + track.Title)
			} else {
				m.statusBar.SetStatus("♡ Unliked: " + track.Title)
			}
			m.refreshLibrary()
		}
		return nil
	case "d": // Manual Download
		selected := m.getSelectedTrack()
		if selected != nil {
			if selected.LocalPath != "" {
				m.statusBar.SetStatus("Already downloaded")
			} else {
				m.engine.DownloadTrack(*selected)
				m.statusBar.SetStatus("Download started: " + selected.Title)
			}
		}
		return nil
	case "P": // Play Playlist or Play All
		switch m.activeTab {
		case TabPlaylists:
			playlist := m.playlists.GetSelectedPlaylist()
			if playlist != "" && playlist != "BACK" && playlist != "DOWNLOADS" {
				_ = m.engine.PlayPlaylist(playlist)
				m.statusBar.SetStatus("Playing playlist: " + playlist)
			}
		case TabLiked:
			tracks, _ := m.engine.GetLikedTracks()
			if len(tracks) > 0 {
				_ = m.engine.PlayTracks(tracks)
				m.statusBar.SetStatus("Playing all Liked songs")
			} else {
				m.statusBar.SetStatus("No liked songs to play")
			}
		case TabHistory:
			tracks, _ := m.engine.GetHistory(100)
			if len(tracks) > 0 {
				_ = m.engine.PlayTracks(tracks)
				m.statusBar.SetStatus("Playing recent history")
			} else {
				m.statusBar.SetStatus("No history to play")
			}
		case TabDownloads:
			tracks, _ := m.engine.GetDownloadedTracks()
			if len(tracks) > 0 {
				_ = m.engine.PlayTracks(tracks)
				m.statusBar.SetStatus("Playing all downloaded songs")
			} else {
				m.statusBar.SetStatus("No downloaded songs to play")
			}
		case TabResults:
			tracks := m.results.tracks
			if len(tracks) > 0 {
				_ = m.engine.PlayTracks(tracks)
				m.statusBar.SetStatus("Playing all search results")
			} else {
				m.statusBar.SetStatus("No search results to play")
			}
		}
		return nil
	case "C": // Create Playlist
		m.showPlaylistPrompt = true
		m.playlistPrompt.Focus()
		return nil
	case "p": // Add to Playlist
		selected := m.getSelectedTrack()
		if selected != nil {
			m.trackToAddToPlaylist = selected
			m.allPlaylists, _ = m.engine.GetPlaylists()
			if len(m.allPlaylists) == 0 {
				m.statusBar.SetStatus("No playlists. Press 'C' to create one.")
			} else {
				m.showPlaylistSelector = true
				m.playlistSelectorIndex = 0
			}
			return nil
		}
	case "x": // Remove from Playlist or Delete Download
		if m.activeTab == TabPlaylists {
			if m.playlists.activePlaylist != "" {
				selected := m.playlists.GetSelected()
				if selected != nil {
					_ = m.engine.RemoveTrackFromPlaylist(m.playlists.activePlaylist, selected.VideoID)
					m.statusBar.SetStatus("Removed from playlist")
					m.refreshLibrary()
				}
			}
			return nil
		} else if m.activeTab == TabDownloads {
			selected := m.downloads.GetSelected()
			if selected != nil {
				_ = m.engine.DeleteDownload(selected.VideoID)
				m.statusBar.SetStatus("Deleted download: " + selected.Title)
				m.refreshLibrary()
			}
			return nil
		}
		if m.focusArea == AreaQueue {
			m.engine.RemoveFromQueue(m.queue.Cursor)
			return nil
		}
	case "D": // Delete Playlist
		if m.activeTab == TabPlaylists && m.playlists.activePlaylist == "" {
			playlist := m.playlists.GetSelectedPlaylist()
			if playlist != "" && playlist != "BACK" && playlist != "DOWNLOADS" {
				_ = m.engine.DeletePlaylist(playlist)
				m.statusBar.SetStatus("Deleted playlist: " + playlist)
				m.refreshLibrary()
			}
			return nil
		}
	case "S": // Shuffle
		m.engine.Shuffle()
		m.statusBar.SetStatus("Shuffled queue")
		return nil
	case "z": // Previous
		if err := m.engine.Prev(); err != nil {
			m.statusBar.SetStatus(err.Error())
		}
		return nil
	case "n": // Next
		_ = m.engine.Next()
		return nil
	case "q":
		m.focusArea = AreaQueue
		m.search.Blur()
		m.queue.Focus()
		m.results.Blur()
		return nil
	case "s", "/":
		m.focusArea = AreaSearch
		m.search.Focus()
		m.queue.Blur()
		m.results.Blur()
		return nil
	case "1":
		m.activeTab = TabResults
		m.focusArea = AreaContent
		m.blurAll()
		m.results.Focus()
		return nil
	case "2":
		m.activeTab = TabLyrics
		m.focusArea = AreaContent
		m.blurAll()
		return nil
	case "3":
		m.activeTab = TabPlaylists
		m.focusArea = AreaContent
		m.blurAll()
		m.playlists.Focus()
		m.refreshLibrary()
		return nil
	case "4":
		m.activeTab = TabLiked
		m.focusArea = AreaContent
		m.blurAll()
		m.liked.Focus()
		m.refreshLibrary()
		return nil
	case "5":
		m.activeTab = TabHistory
		m.focusArea = AreaContent
		m.blurAll()
		m.history.Focus()
		m.refreshLibrary()
		return nil
	case "6":
		m.activeTab = TabDownloads
		m.focusArea = AreaContent
		m.blurAll()
		m.downloads.Focus()
		m.refreshLibrary()
		return nil
	case "7":
		m.activeTab = TabSettings
		m.focusArea = AreaContent
		m.blurAll()
		return nil

	case "?":
		m.help.ShowAll = !m.help.ShowAll
		m.handleWindowSize(tea.WindowSizeMsg{Width: m.terminalWidth, Height: m.terminalHeight})
		return nil

	case "up", "k":
		if m.focusArea == AreaContent && m.activeTab == TabResults {
			m.results.Prev()
		} else if m.focusArea == AreaContent && m.isLibraryTab() {
			if lib := m.getActiveLibrary(); lib != nil {
				lib.Prev()
			}
		} else if m.focusArea == AreaQueue {
			if m.queue.Cursor > 0 {
				m.queue.Cursor--
			}
		} else if m.focusArea == AreaContent && m.activeTab == TabLyrics {
			m.lyrics.ScrollUp(1)
		} else if m.focusArea == AreaContent && m.activeTab == TabSettings {
			m.settings.Update(msg)
		}
		return nil

	case "down", "j":
		if m.focusArea == AreaContent && m.activeTab == TabResults {
			m.results.Next()
		} else if m.focusArea == AreaContent && m.isLibraryTab() {
			if lib := m.getActiveLibrary(); lib != nil {
				lib.Next()
			}
		} else if m.focusArea == AreaQueue {
			if m.queue.Cursor < len(m.engine.GetQueue())-1 {
				m.queue.Cursor++
			}
		} else if m.focusArea == AreaContent && m.activeTab == TabLyrics {
			m.lyrics.ScrollDown(1)
		} else if m.focusArea == AreaContent && m.activeTab == TabSettings {
			m.settings.Update(msg)
		}
		return nil

	case "right", ".":
		if m.activeTab == TabSettings {
			m.settings.Update(msg)
			m.statusBar.SetVizMode(VizColorMode(m.engine.GetConfig().VizMode))
			return nil
		}
		if m.engine.GetCurrentTrack() != nil {
			newPos := m.engine.GetPlayPosition() + 5*time.Second
			_ = m.engine.Seek(newPos)
			m.statusBar.SetStatus(fmt.Sprintf("Seeked to: %s", formatDuration(int(newPos.Seconds()))))
		}
		return nil

	case "left", ",", "h":
		if m.activeTab == TabSettings {
			m.settings.Update(msg)
			m.statusBar.SetVizMode(VizColorMode(m.engine.GetConfig().VizMode))
			return nil
		}
		if m.engine.GetCurrentTrack() != nil {
			newPos := m.engine.GetPlayPosition() - 5*time.Second
			if newPos < 0 {
				newPos = 0
			}
			_ = m.engine.Seek(newPos)
			m.statusBar.SetStatus(fmt.Sprintf("Seeked to: %s", formatDuration(int(newPos.Seconds()))))
		}
		return nil

	case "+", "=":
		if m.activeTab == TabSettings {
			m.settings.Update(msg)
		}
		return nil
	case "-", "_":
		if m.activeTab == TabSettings {
			m.settings.Update(msg)
		}
		return nil

	case "]":
		vol := m.engine.GetVolume()
		vol += 0.05
		if vol > 2.0 {
			vol = 2.0
		}
		m.engine.SetVolume(vol)
		m.statusBar.SetStatus(fmt.Sprintf("Volume: %d%%", int(vol*100)))
		return nil

	case "[":
		vol := m.engine.GetVolume()
		vol -= 0.05
		if vol < 0.0 {
			vol = 0.0
		}
		m.engine.SetVolume(vol)
		m.statusBar.SetStatus(fmt.Sprintf("Volume: %d%%", int(vol*100)))
		return nil

	case "K": // Move Up in Queue
		if m.focusArea == AreaQueue {
			if m.queue.Cursor > 0 {
				m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor-1)
				m.queue.Cursor--
			}
			return nil
		}
	case "J": // Move Down in Queue
		if m.focusArea == AreaQueue {
			queueLen := len(m.engine.GetQueue())
			if m.queue.Cursor < queueLen-1 {
				m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor+1)
				m.queue.Cursor++
			}
			return nil
		}
	case "c": // Clear Queue
		if m.focusArea == AreaQueue {
			m.engine.ClearQueue()
			m.queue.Cursor = 0
			m.statusBar.SetStatus("Queue cleared")
			return nil
		}
	case "a": // Add to Queue
		if m.focusArea == AreaContent && m.activeTab == TabResults {
			selected := m.results.GetSelected()
			if selected != nil {
				m.engine.Queue(*selected)
				m.statusBar.SetStatus("Added to queue: " + selected.Title)
			}
			return nil
		} else if m.focusArea == AreaContent && m.isLibraryTab() {
			if lib := m.getActiveLibrary(); lib != nil {
				selected := lib.GetSelected()
				if selected != nil {
					m.engine.Queue(*selected)
					m.statusBar.SetStatus("Added to queue: " + selected.Title)
				}
			}
			return nil
		}

	case "space":
		paused := m.engine.TogglePause()
		track := m.engine.GetCurrentTrack()
		if track != nil {
			if paused {
				m.statusBar.SetStatus("|| Paused : " + track.Title)
			} else {
				m.statusBar.SetStatus("|> Playing : " + track.Title)
			}
		}
		return nil

	case "X":
		if m.activeTab == TabSettings {
			_ = m.engine.ClearCache()
			m.statusBar.SetStatus("Cache cleared")
		}
		return nil

	case "enter":
		if m.activeTab == TabSettings {
			m.settings.Update(msg)
			return nil
		}
		if m.focusArea == AreaContent && m.activeTab == TabResults {
			selected := m.results.GetSelected()
			if selected != nil {
				_ = m.engine.Play(*selected)
			}
		} else if m.focusArea == AreaContent && m.isLibraryTab() {
			lib := m.getActiveLibrary()
			if lib == nil {
				return nil
			}
			playlist := lib.GetSelectedPlaylist()
			if playlist == "BACK" {
				lib.activePlaylist = ""
				m.refreshLibrary()
				return nil
			} else if playlist != "" {
				lib.activePlaylist = playlist
				m.refreshLibrary()
				return nil
			}

			selected := lib.GetSelected()
			if selected != nil {
				_ = m.engine.Play(*selected)
			}
		} else if m.focusArea == AreaQueue {
			_ = m.engine.PlayFromQueue(m.queue.Cursor)
		}
		return nil
	}

	return nil
}

func (m *Model) handleSearchResults(msg SearchResultsMsg) {
	m.loading = false
	if msg.Err != nil {
		m.err = msg.Err
		m.statusBar.SetStatus("Search error: " + msg.Err.Error())
		return
	}
	m.results.SetTracks(msg.Results)
	m.statusBar.SetStatus(fmt.Sprintf("Found %d results", len(msg.Results)))
	if len(msg.Results) == 0 {
		m.statusBar.SetStatus("No results found for '" + m.search.Value() + "'")
		return
	}
	m.focusArea = AreaContent
	m.activeTab = TabResults
	m.search.Blur()
	m.results.Focus()
}

func (m *Model) handleSuggestions(msg SuggestionsMsg) {
	if m.search.Value() != "" {
		m.suggestions = msg.Suggestions
		m.showSuggest = true
	}
}

func (m *Model) handleSearchHistory(msg SearchHistoryMsg) {
	if m.search.Value() == "" {
		m.suggestions = msg.History
		m.showSuggest = len(m.suggestions) > 0
	}
}

func (m *Model) handleThumbnail(msg ThumbnailMsg) {
	if msg.VideoID != m.lastTrackID || msg.Cols != m.lastThumbWidth {
		return
	}
	if msg.Err == nil {
		m.thumbnail = msg.Art
		m.thumbError = ""
		return
	}
	m.thumbError = fmt.Sprintf("%v", msg.Err)
	m.thumbnail = ""
}

func (m *Model) handleTick() []tea.Cmd {
	var cmds []tea.Cmd

	m.downloadProgress = m.engine.GetActiveDownloads()
	m.refreshResults()

	lyricsText := m.engine.GetLyrics()
	if lyricsText != m.lastLyricsText {
		m.lastLyricsText = lyricsText
		m.syncedLines = parseLRC(lyricsText)
		if len(m.syncedLines) == 0 {
			m.lyrics.SetContent(lyricsText)
		}
		m.lastActiveLine = -2
	}

	if len(m.syncedLines) > 0 {
		m.updateSyncedLyrics()
	}

	track := m.engine.GetCurrentTrack()
	if track != nil {
		state := m.engine.GetState()

		thumbCols := m.thumbTargetCols()

		if track.VideoID != m.lastTrackID || (track.ThumbURL != "" && m.lastThumbURL == "") || thumbCols != m.lastThumbWidth {
			if track.VideoID != m.lastTrackID {
				m.statusBar.elapsed = 0
				m.statusBar.lastTick = time.Now()
				m.lastState = -1
			}
			m.lastTrackID = track.VideoID
			m.lastThumbURL = track.ThumbURL
			m.lastThumbWidth = thumbCols
			m.thumbnail = ""
			m.thumbError = ""

			if track.ThumbURL != "" {
				vid := track.VideoID
				t := *track
				cmds = append(cmds, func() tea.Msg {
					art, err := m.engine.GetThumbnailArt(t, thumbCols)
					return ThumbnailMsg{VideoID: vid, Art: art, Cols: thumbCols, Err: err}
				})
			}
		}

		m.statusBar.elapsed = m.engine.GetPlayPosition()
		if state == player.StatePlaying {
			m.vinylFrame = (m.vinylFrame + 1) % 4
			m.engine.UpdateMPRISPosition()
		}

		pct := 0.0
		if track.Duration > 0 {
			pct = float64(m.statusBar.elapsed.Seconds()) / float64(track.Duration)
		}
		if pct > 1.0 {
			pct = 1.0
		}

		if state != m.lastState {
			m.lastState = state
			title := "goktave"
			if state == player.StatePlaying {
				title = fmt.Sprintf("%s - %s | goktave", track.Title, track.Artist)
			}
			cmds = append(cmds, SetTitleCmd(title))

			switch state {
			case player.StateBuffering:
				m.statusBar.SetStatus("Buffering : " + track.Title + "...")
			case player.StatePlaying:
				m.statusBar.SetStatus("|> Now Playing : " + track.Title)
			case player.StatePaused:
				m.statusBar.SetStatus("|| Paused : " + track.Title)
			case player.StateStopped:
				m.statusBar.SetStatus("Ready : " + track.Title)
			}
		}

		if pct > 0.90 && !m.engine.IsPreloading() {
			m.engine.Preload()
		}
		cmds = append(cmds, m.statusBar.progress.SetPercent(pct))
	}

	cmds = append(cmds, tickCmd())
	return cmds
}
