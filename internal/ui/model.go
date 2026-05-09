package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/Kush-Singh-26/goktave/internal/db"
	"github.com/Kush-Singh-26/goktave/internal/engine"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type FocusArea int

const (
	AreaSearch FocusArea = iota
	AreaQueue
	AreaContent
)

type ContentTab int

const (
	TabResults ContentTab = iota
	TabLyrics
	TabPlaylists
	TabLiked
	TabHistory
	TabDownloads
	TabSettings
)

type Model struct {
	engine engine.Engine

	focusArea FocusArea
	activeTab ContentTab

	search  textinput.Model
	results ResultsList

	// Separate lists for library sections
	playlists LibraryList
	liked     LibraryList
	history   LibraryList
	downloads LibraryList

	queue    QueueView
	settings SettingsView
	lyrics   viewport.Model

	statusBar StatusBar
	help      help.Model

	terminalWidth  int
	terminalHeight int

	err     error
	loading bool

	lastTrackID    string
	lastThumbURL   string
	lastThumbWidth int
	lastState      player.State
	lastLyricsText string

	thumbnail       string
	suggestions     []string
	suggestionIndex int
	showSuggest     bool

	// Playlist management
	playlistPrompt        textinput.Model
	showPlaylistPrompt    bool
	showPlaylistSelector  bool
	playlistSelectorIndex int
	allPlaylists          []db.Playlist
	trackToAddToPlaylist  *provider.Track

	downloadProgress map[string]float64

	cancel context.CancelFunc
}

func NewModel(e engine.Engine) Model {
	// Sync theme from config
	cfg := e.GetConfig()
	themeIdx := 0
	for i, t := range Themes {
		if t.Name == cfg.Theme {
			ActiveTheme = &Themes[i]
			themeIdx = i
			break
		}
	}
	RefreshStyles()

	ti := textinput.New()
	ti.Placeholder = "Search songs, artists..."
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(30)

	h := help.New()
	h.Styles.ShortKey = StyleTitle
	h.Styles.ShortDesc = StyleMeta
	h.Styles.FullKey = StyleTitle
	h.Styles.FullDesc = StyleMeta

	pp := textinput.New()
	pp.Placeholder = "New playlist name..."
	pp.CharLimit = 32
	pp.SetWidth(30)

	return Model{
		engine:          e,
		focusArea:       AreaSearch,
		activeTab:       TabResults,
		search:          ti,
		playlistPrompt:  pp,
		results:         NewResultsList(),
		playlists:       NewLibraryList(),
		liked:           NewLibraryList(),
		history:         NewLibraryList(),
		downloads:       NewLibraryList(),
		queue:           QueueView{},
		settings:        SettingsView{engine: e, ThemeIndex: themeIdx},
		lyrics:          viewport.New(),
		statusBar:       NewStatusBar(),
		help:            h,
		lastState:       -1,
		suggestionIndex: -1,
	}
}

type SearchResultsMsg struct {
	Results []provider.Track
	Err     error
}

type SuggestionsMsg struct {
	Suggestions []string
	Err         error
}

type SearchHistoryMsg struct {
	History []string
}

type ThumbnailMsg struct {
	VideoID string
	ASCII   string
	Err     error
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickCmd())
}

func (m Model) cancelRunningTask() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func SetTitleCmd(title string) tea.Cmd {
	return func() tea.Msg {
		fmt.Printf("\033]2;%s\007", title)
		return nil
	}
}

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
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		compact := m.terminalHeight <= 33
		brandingWidth := int(float64(msg.Width-2) * 0.35)
		remaining := (msg.Width - 2) - brandingWidth
		searchWidth := int(float64(remaining) * 0.45)
		m.search.SetWidth(searchWidth - 6)

		visibleHeight := m.getVisibleHeight()
		paneFrameV := PaneStyle.GetVerticalFrameSize()
		vizOuterHeight := 4 + paneFrameV
		topHeightOuter := visibleHeight - vizOuterHeight
		if topHeightOuter < 1 {
			topHeightOuter = visibleHeight
		}
		topInnerHeight := topHeightOuter - paneFrameV
		if topInnerHeight < 1 {
			topInnerHeight = 1
		}
		if compact {
			vizOuterHeight = 0
			topHeightOuter = visibleHeight
			topInnerHeight = visibleHeight - paneFrameV
			if topInnerHeight < 1 {
				topInnerHeight = 1
			}
		}
		contentWidth := m.terminalWidth - 4
		mainWidth := int(float64(contentWidth) * 0.55)
		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		lyricsHeight := topInnerHeight - 1
		if lyricsHeight < 1 {
			lyricsHeight = 1
		}
		m.lyrics.SetWidth(contentInnerWidth)
		m.lyrics.SetHeight(lyricsHeight)

	case tea.KeyMsg:
		// 1. Absolute Global Keys (Always work)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.search.Focused() && len(m.suggestions) > 0 {
				idx := m.suggestionIndex
				if idx < 0 {
					idx = 0
				}
				m.search.SetValue(m.suggestions[idx])
				m.showSuggest = false
				m.suggestionIndex = -1
				return m, nil
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
			return m, nil
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
				return m, nil
			case "up":
				if len(m.suggestions) > 0 {
					if m.suggestionIndex > 0 {
						m.suggestionIndex--
					} else {
						m.suggestionIndex = len(m.suggestions) - 1
					}
					return m, nil
				}
			case "down":
				if len(m.suggestions) > 0 {
					if m.suggestionIndex < len(m.suggestions)-1 {
						m.suggestionIndex++
					} else {
						m.suggestionIndex = 0
					}
					return m, nil
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

					return m, func() tea.Msg {
						res, err := m.engine.Search(ctx, query)
						return SearchResultsMsg{Results: res, Err: err}
					}
				}
			}
		} else {
			// 3. Navigation & Command keys (Only when NOT typing)
			switch msg.String() {
			case "l": // Toggle Like
				track := m.engine.GetCurrentTrack()
				if track != nil {
					liked, _ := m.engine.ToggleLike(track.VideoID)
					if liked {
						m.statusBar.SetStatus("❤️ Added to Likes")
					} else {
						m.statusBar.SetStatus("💔 Removed from Likes")
					}
					if m.isLibraryTab() {
						m.refreshLibrary()
					}
				}
				return m, nil
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
				return m, nil
			case "P": // Play Playlist
				if m.activeTab == TabPlaylists {
					playlist := m.playlists.GetSelectedPlaylist()
					if playlist != "" && playlist != "BACK" && playlist != "DOWNLOADS" {
						_ = m.engine.PlayPlaylist(playlist)
						m.statusBar.SetStatus("Playing playlist: " + playlist)
					}
				}
				return m, nil
			case "C": // Create Playlist
				m.showPlaylistPrompt = true
				m.playlistPrompt.Focus()
				return m, nil
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
					return m, nil
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
					return m, nil
				} else if m.activeTab == TabDownloads {
					selected := m.downloads.GetSelected()
					if selected != nil {
						_ = m.engine.DeleteDownload(selected.VideoID)
						m.statusBar.SetStatus("Deleted download: " + selected.Title)
						m.refreshLibrary()
					}
					return m, nil
				}
				if m.focusArea == AreaQueue {
					m.engine.RemoveFromQueue(m.queue.Cursor)
					return m, nil
				}
			case "D": // Delete Playlist
				if m.activeTab == TabPlaylists && m.playlists.activePlaylist == "" {
					playlist := m.playlists.GetSelectedPlaylist()
					if playlist != "" && playlist != "BACK" && playlist != "DOWNLOADS" {
						_ = m.engine.DeletePlaylist(playlist)
						m.statusBar.SetStatus("Deleted playlist: " + playlist)
						m.refreshLibrary()
					}
					return m, nil
				}
			case "z": // Previous
				_ = m.engine.Prev()
				return m, nil
			case "n": // Next
				_ = m.engine.Next()
				return m, nil
			case "q":
				m.focusArea = AreaQueue
				m.search.Blur()
				m.queue.Focus()
				m.results.Blur()
				return m, nil
			case "s", "/":
				m.focusArea = AreaSearch
				m.search.Focus()
				m.queue.Blur()
				m.results.Blur()
				return m, nil
			case "1":
				m.activeTab = TabResults
				m.focusArea = AreaContent
				m.blurAll()
				m.results.Focus()
				return m, nil
			case "2":
				m.activeTab = TabLyrics
				m.focusArea = AreaContent
				m.blurAll()
				return m, nil
			case "3":
				m.activeTab = TabPlaylists
				m.focusArea = AreaContent
				m.blurAll()
				m.playlists.Focus()
				m.refreshLibrary()
				return m, nil
			case "4":
				m.activeTab = TabLiked
				m.focusArea = AreaContent
				m.blurAll()
				m.liked.Focus()
				m.refreshLibrary()
				return m, nil
			case "5":
				m.activeTab = TabHistory
				m.focusArea = AreaContent
				m.blurAll()
				m.history.Focus()
				m.refreshLibrary()
				return m, nil
			case "6":
				m.activeTab = TabDownloads
				m.focusArea = AreaContent
				m.blurAll()
				m.downloads.Focus()
				m.refreshLibrary()
				return m, nil
			case "7":
				m.activeTab = TabSettings
				m.focusArea = AreaContent
				m.blurAll()
				return m, nil

			case "?":
				m.help.ShowAll = !m.help.ShowAll
				return m, nil

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
				return m, nil

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
				return m, nil

			case "left", "h", "right":
				if m.activeTab == TabSettings {
					m.settings.Update(msg)
				}
				return m, nil

			case "+", "=":
				if m.activeTab == TabSettings {
					m.settings.Update(msg)
				}
				return m, nil
			case "-", "_":
				if m.activeTab == TabSettings {
					m.settings.Update(msg)
				}
				return m, nil

			case "K": // Move Up in Queue
				if m.focusArea == AreaQueue {
					if m.queue.Cursor > 0 {
						m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor-1)
						m.queue.Cursor--
					}
					return m, nil
				}
			case "J": // Move Down in Queue
				if m.focusArea == AreaQueue {
					queueLen := len(m.engine.GetQueue())
					if m.queue.Cursor < queueLen-1 {
						m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor+1)
						m.queue.Cursor++
					}
					return m, nil
				}
			case "c": // Clear Queue
				if m.focusArea == AreaQueue {
					m.engine.ClearQueue()
					m.queue.Cursor = 0
					m.statusBar.SetStatus("Queue cleared")
					return m, nil
				}
			case "a": // Add to Queue
				if m.focusArea == AreaContent && m.activeTab == TabResults {
					selected := m.results.GetSelected()
					if selected != nil {
						m.engine.Queue(*selected)
						m.statusBar.SetStatus("Added to queue: " + selected.Title)
					}
					return m, nil
				} else if m.focusArea == AreaContent && m.isLibraryTab() {
					if lib := m.getActiveLibrary(); lib != nil {
						selected := lib.GetSelected()
						if selected != nil {
							m.engine.Queue(*selected)
							m.statusBar.SetStatus("Added to queue: " + selected.Title)
						}
					}
					return m, nil
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
				return m, nil

			case "X":
				if m.activeTab == TabSettings {
					_ = m.engine.ClearCache()
					m.statusBar.SetStatus("Cache cleared")
				}
				return m, nil

			case "enter":
				if m.activeTab == TabSettings {
					m.settings.Update(msg)
					return m, nil
				}
				if m.focusArea == AreaContent && m.activeTab == TabResults {
					selected := m.results.GetSelected()
					if selected != nil {
						_ = m.engine.Play(*selected)
					}
				} else if m.focusArea == AreaContent && m.isLibraryTab() {
					lib := m.getActiveLibrary()
					if lib == nil {
						return m, nil
					}
					playlist := lib.GetSelectedPlaylist()
					if playlist == "BACK" {
						lib.activePlaylist = ""
						m.refreshLibrary()
						return m, nil
					} else if playlist != "" {
						lib.activePlaylist = playlist
						m.refreshLibrary()
						return m, nil
					}

					selected := lib.GetSelected()
					if selected != nil {
						_ = m.engine.Play(*selected)
					}
				} else if m.focusArea == AreaQueue {
					_ = m.engine.PlayFromQueue(m.queue.Cursor)
				}
				return m, nil
			}
		}

	case SearchResultsMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.statusBar.SetStatus("Search error: " + msg.Err.Error())
		} else {
			m.results.SetTracks(msg.Results)
			m.statusBar.SetStatus(fmt.Sprintf("Found %d results", len(msg.Results)))
			if len(msg.Results) == 0 {
				m.statusBar.SetStatus("No results found for '" + m.search.Value() + "'")
			} else {
				m.focusArea = AreaContent
				m.activeTab = TabResults
				m.search.Blur()
				m.results.Focus()
			}
		}

	case SuggestionsMsg:
		if m.search.Value() != "" {
			m.suggestions = msg.Suggestions
			m.showSuggest = true
		}

	case SearchHistoryMsg:
		if m.search.Value() == "" {
			m.suggestions = msg.History
			m.showSuggest = len(m.suggestions) > 0
		}

	case ThumbnailMsg:
		if msg.VideoID == m.lastTrackID {
			if msg.Err == nil {
				m.thumbnail = msg.ASCII
			} else {
				m.thumbnail = fmt.Sprintf("\n\n  Error loading thumbnail:\n  %v", msg.Err)
			}
		}

	case tickMsg:
		m.downloadProgress = m.engine.GetActiveDownloads()
		m.refreshResults()

		lyricsText := m.engine.GetLyrics()
		if lyricsText != m.lastLyricsText {
			m.lastLyricsText = lyricsText
			m.lyrics.SetContent(lyricsText)
		}

		track := m.engine.GetCurrentTrack()
		if track != nil {
			state := m.engine.GetState()

			contentWidth := m.terminalWidth - 4
			nowPlayingWidth := int(float64(contentWidth) * 0.25)
			thumbWidth := nowPlayingWidth - 4
			if thumbWidth < 10 {
				thumbWidth = 10
			}
			if thumbWidth > 50 {
				thumbWidth = 50
			}

			if track.VideoID != m.lastTrackID || (track.ThumbURL != "" && m.lastThumbURL == "") || thumbWidth != m.lastThumbWidth {
				if track.VideoID != m.lastTrackID {
					m.statusBar.elapsed = 0
					m.statusBar.lastTick = time.Now()
					m.lastState = -1
				}
				m.lastTrackID = track.VideoID
				m.lastThumbURL = track.ThumbURL
				m.lastThumbWidth = thumbWidth
				m.thumbnail = "Loading thumbnail..."

				if track.ThumbURL != "" {
					vid := track.VideoID
					t := *track
					cmds = append(cmds, func() tea.Msg {
						ascii, err := m.engine.GetASCIIThumbnail(t, thumbWidth)
						return ThumbnailMsg{VideoID: vid, ASCII: ascii, Err: err}
					})
				} else {
					m.thumbnail = "\n\n  No Thumbnail URL found"
				}
			}

			if state == player.StatePlaying {
				now := time.Now()
				m.statusBar.elapsed += now.Sub(m.statusBar.lastTick)
				m.statusBar.lastTick = now
			} else {
				m.statusBar.lastTick = time.Now()
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
	}

	var cmd tea.Cmd
	// SINGLE Update call for search input
	oldVal := m.search.Value()
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)

	// Suggestions trigger logic based on the SINGLE update above
	if m.search.Focused() {
		if _, ok := msg.(tea.KeyMsg); ok {
			if m.search.Value() != "" {
				if m.search.Value() != oldVal {
					m.suggestionIndex = -1
				}
				cmds = append(cmds, func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					res, err := m.engine.GetSuggestions(ctx, m.search.Value())
					return SuggestionsMsg{Suggestions: res, Err: err}
				})
			} else {
				cmds = append(cmds, func() tea.Msg {
					res, _ := m.engine.GetSearchHistory(5)
					return SearchHistoryMsg{History: res}
				})
			}
		}
	}

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
		return m.playlists.View(height, width)
	case TabLiked:
		return m.liked.View(height, width)
	case TabHistory:
		return m.history.View(height, width)
	case TabDownloads:
		return m.downloads.View(height, width)
	}
	return ""
}

func (m Model) getVisibleHeight() int {
	compact := m.terminalHeight <= 33
	headerBoxHeight := 5
	if compact {
		headerBoxHeight = 3
	}
	headerHeight := headerBoxHeight + HeaderStyle.GetVerticalFrameSize()
	helpHeight := 1
	if !compact && m.help.ShowAll {
		helpHeight = 7
	}

	available := m.terminalHeight - headerHeight - helpHeight
	if available < 2 {
		available = 2
	}
	return available
}

func (m Model) View() tea.View {
	if m.terminalWidth < 20 || m.terminalHeight < 10 {
		return tea.View{Content: "Terminal too small"}
	}

	branding := "█▀▀▀█  █▀▀▀█  █  ▄▀  ▀▀█▀▀  █▀▀▀█  █   █  █▀▀▀█\n" +
		"█ ▀▀█  █   █  █▀▀▄     █    █▀▀▀█  █   █  █▀▀▀ \n" +
		"▀▀▀▀▀  ▀▀▀▀▀  ▀  ▀     ▀    ▀   ▀   ▀▀▀   ▀▀▀▀▀"

	brandingWidth := 51
	searchWidth := 40
	suggestWidth := 50
	compact := m.terminalHeight <= 33
	headerBoxHeight := 5
	if compact {
		headerBoxHeight = 3
	}

	// Adaptive scaling for smaller terminals
	if m.terminalWidth < brandingWidth+searchWidth+suggestWidth+4 {
		brandingWidth = int(float64(m.terminalWidth-2) * 0.30)
		remaining := (m.terminalWidth - 2) - brandingWidth
		searchWidth = int(float64(remaining) * 0.45)
		suggestWidth = int(float64(remaining) * 0.40)
	}
	if brandingWidth < 10 {
		brandingWidth = 10
	}
	if searchWidth < 10 {
		searchWidth = 10
	}
	if suggestWidth < 10 {
		suggestWidth = 10
	}

	brandingBox := HeaderStyle.Copy().
		Width(brandingWidth).
		Height(headerBoxHeight).
		Foreground(Terracotta).
		Padding(0, 1).
		Render(branding)

	searchBoxView := StyleMeta.Render("Search:") + "\n" + m.search.View()
	searchBoxStyle := HeaderStyle.Copy().
		Width(searchWidth).
		Height(headerBoxHeight).
		Padding(0, 1).
		Align(lipgloss.Left, lipgloss.Center)

	if m.focusArea == AreaSearch {
		searchBoxStyle = searchBoxStyle.BorderForeground(Terracotta)
	}
	searchBox := searchBoxStyle.Render(searchBoxView)

	var header string
	if compact {
		gap := "  "
		header = lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, gap, searchBox)
	} else {
		var suggestLines []string
		if m.showSuggest {
			title := "Suggestions:"
			if m.search.Value() == "" {
				title = "Recent Searches:"
			}
			suggestLines = append(suggestLines, StyleMeta.Render(title))

			for i, s := range m.suggestions {
				if i >= 4 { // Only show 4 suggestions
					break
				}
				line := s
				if i == m.suggestionIndex {
					line = StyleSelected.Render("> " + s)
				} else {
					line = StyleMeta.Render("  " + s)
				}
				suggestLines = append(suggestLines, line)
			}
		} else if len(m.downloadProgress) > 0 {
			suggestLines = append(suggestLines, StyleTitle.Render("Active Downloads:"))
			keys := make([]string, 0, len(m.downloadProgress))
			for k := range m.downloadProgress {
				keys = append(keys, k)
			}
			// Show up to 4 downloads
			for i := 0; i < 4 && i < len(keys); i++ {
				title := keys[i]
				pct := m.downloadProgress[title]
				barWidth := suggestWidth - 15
				if barWidth < 5 {
					barWidth = 5
				}

				filled := int(float64(barWidth) * pct)
				empty := barWidth - filled
				if empty < 0 {
					empty = 0
				}

				bar := lipgloss.NewStyle().Foreground(Terracotta).Render(strings.Repeat("█", filled)) +
					lipgloss.NewStyle().Foreground(SurfaceDeep).Render(strings.Repeat("░", empty))

				name := title
				if len(name) > 15 {
					name = name[:12] + "..."
				}

				suggestLines = append(suggestLines, fmt.Sprintf("%s %s", StyleMeta.Render(name), bar))
			}
		}

		// Always ensure exactly 5 lines (1 title + 4 items) to prevent jumping
		for len(suggestLines) < 5 {
			if len(suggestLines) == 0 {
				suggestLines = append(suggestLines, StyleMeta.Render("Suggestions:"))
			} else {
				suggestLines = append(suggestLines, "")
			}
		}

		suggestBoxView := strings.Join(suggestLines, "\n")
		suggestBox := HeaderStyle.Copy().
			Width(suggestWidth).
			Height(headerBoxHeight).
			Padding(0, 1).
			Align(lipgloss.Left, lipgloss.Center).
			Render(suggestBoxView)

		// Add a gap between branding and search box to "shift" it right
		gap := "  "
		header = lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, gap, searchBox, suggestBox)
	}

	// Help line
	helpHeight := 1
	if !compact && m.help.ShowAll {
		helpHeight = 7
	}
	helpView := lipgloss.NewStyle().
		Width(m.terminalWidth).
		Height(helpHeight).
		Render("  " + m.help.View(Keys))

	// Dynamic height for body
	headerHeight := lipgloss.Height(header)
	helpRenderedHeight := lipgloss.Height(helpView)
	visibleHeight := m.terminalHeight - headerHeight - helpRenderedHeight
	if visibleHeight < 2 {
		visibleHeight = 2
	}
	paneFrameV := PaneStyle.GetVerticalFrameSize()
	vizOuterHeight := 4 + paneFrameV
	topHeightOuter := visibleHeight - vizOuterHeight
	if topHeightOuter < 1 {
		topHeightOuter = visibleHeight
		vizOuterHeight = 0
	}
	topInnerHeight := topHeightOuter - paneFrameV
	if topInnerHeight < 1 {
		topInnerHeight = 1
	}
	if compact {
		vizOuterHeight = 0
		topHeightOuter = visibleHeight
		topInnerHeight = visibleHeight - paneFrameV
		if topInnerHeight < 1 {
			topInnerHeight = 1
		}
	}

	contentWidth := m.terminalWidth - 4
	queueWidth := int(float64(contentWidth) * 0.20)
	mainWidth := int(float64(contentWidth) * 0.55)
	mainColumnWidth := contentWidth - queueWidth
	nowPlayingWidth := mainColumnWidth - mainWidth
	minQueueWidth := PaneStyle.GetHorizontalFrameSize() + 6
	minNowPlayingWidth := PaneStyle.GetHorizontalFrameSize() + 12
	minMainWidth := PaneStyle.GetHorizontalFrameSize() + 12
	if queueWidth < minQueueWidth {
		queueWidth = minQueueWidth
	}
	if mainWidth < minMainWidth {
		mainWidth = minMainWidth
	}
	if nowPlayingWidth < minNowPlayingWidth {
		nowPlayingWidth = minNowPlayingWidth
	}
	if queueWidth+mainWidth+nowPlayingWidth > contentWidth && contentWidth > 0 {
		available := contentWidth - queueWidth - minNowPlayingWidth
		if available > minMainWidth {
			mainWidth = available
			nowPlayingWidth = contentWidth - queueWidth - mainWidth
		} else {
			mainWidth = minMainWidth
			nowPlayingWidth = contentWidth - queueWidth - mainWidth
			if nowPlayingWidth < minNowPlayingWidth {
				nowPlayingWidth = minNowPlayingWidth
			}
		}
	}
	nowPlayingInnerWidth := nowPlayingWidth - PaneStyle.GetHorizontalFrameSize()
	if nowPlayingInnerWidth < 1 {
		nowPlayingInnerWidth = 1
	}

	// Queue Pane
	queueStyle := PaneStyle.Copy().Width(queueWidth).Height(topHeightOuter)
	if m.focusArea == AreaQueue {
		queueStyle = ActivePaneStyle.Copy().Width(queueWidth).Height(topHeightOuter)
	}
	queueInnerWidth := queueWidth - PaneStyle.GetHorizontalFrameSize()
	if queueInnerWidth < 1 {
		queueInnerWidth = 1
	}
	queueContent := m.queue.View(m.engine.GetQueue(), topInnerHeight, queueInnerWidth)
	queueContent = lipgloss.Place(queueInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, queueContent)
	queuePane := queueStyle.Render(queueContent)

	// Main Content Pane
	tabs := []string{
		"[1] Results",
		"[2] Lyrics",
		"[3] Playlists",
		"[4] Liked",
		"[5] History",
		"[6] Downloads",
		"[7] Settings",
	}
	tabRow := ""
	for i, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if int(m.activeTab) == i {
			style = style.Foreground(Terracotta).Bold(true).Underline(true)
		}
		tabRow += style.Render(t)
	}

	contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
	if contentInnerWidth < 1 {
		contentInnerWidth = 1
	}
	contentInnerHeight := topInnerHeight - 1
	if contentInnerHeight < 1 {
		contentInnerHeight = 1
	}

	var contentBody string
	switch m.activeTab {
	case TabResults:
		contentBody = m.results.View(contentInnerHeight, contentInnerWidth)
	case TabLyrics:
		contentBody = m.lyrics.View()
	case TabPlaylists, TabLiked, TabHistory, TabDownloads:
		contentBody = m.renderLibrary(contentInnerHeight, contentInnerWidth)
	case TabSettings:
		contentBody = m.settings.View(contentInnerWidth, contentInnerHeight, m.engine.GetConfig().MaxCacheSizeGB, m.engine.GetCacheSize())
	}

	contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
	if m.focusArea == AreaContent {
		contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
	}
	contentText := tabRow + "\n" + contentBody
	contentText = lipgloss.Place(contentInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, contentText)
	contentPane := contentStyle.Render(contentText)

	// Now Playing Pane
	nowPlayingContent := ""
	track := m.engine.GetCurrentTrack()
	if track != nil {
		thumb := m.thumbnail
		if thumb == "" {
			thumb = "\n\n  No Thumbnail"
		}

		title := StyleTitle.Render(track.Title)
		artist := StyleMeta.Render(track.Artist)
		likeStatus := ""
		if m.engine.IsLiked(track.VideoID) {
			likeStatus = " ❤️"
		}
		offlineStatus := ""
		if track.LocalPath != "" {
			offlineStatus = " " + StyleMeta.Render("✔")
		}

		playbarWidth := nowPlayingInnerWidth
		playbar := m.statusBar.PlaybarView(track, m.engine.GetState(), playbarWidth)

		// Ensure thumbnail doesn't overflow
		// Content area inside pane is inner height
		// title (1) + artist (1) + playbar (1) + 3 spacing (\n) = 6
		metadataHeight := 6
		innerNowPlayingHeight := topInnerHeight
		if innerNowPlayingHeight < 1 {
			innerNowPlayingHeight = 1
		}
		maxThumbHeight := innerNowPlayingHeight - metadataHeight
		if maxThumbHeight < 5 {
			maxThumbHeight = 5
		}

		// Calculate thumbnail width based on the pane width minus borders/padding
		thumbWidth := nowPlayingInnerWidth
		thumb = clampLines(thumb, maxThumbHeight)
		thumb = lipgloss.Place(thumbWidth, maxThumbHeight, lipgloss.Center, lipgloss.Top, thumb)

		nowPlayingContent = lipgloss.JoinVertical(lipgloss.Center,
			ThumbnailStyle.Width(thumbWidth).MaxHeight(maxThumbHeight).Render(thumb),
			"\n",
			title+likeStatus+offlineStatus,
			artist,
			"\n",
			playbar,
		)
	} else {
		nowPlayingContent = "\n\n  " + StyleMeta.Render("Nothing playing")
	}

	nowPlayingContent = lipgloss.Place(nowPlayingInnerWidth, topInnerHeight, lipgloss.Center, lipgloss.Top, nowPlayingContent)

	nowPlayingPane := PaneStyle.Copy().
		Width(nowPlayingWidth).
		Height(topHeightOuter).
		Render(nowPlayingContent)

	vizHeight := 4
	visualizerWidth := queueWidth + mainWidth
	visualizerInnerWidth := visualizerWidth - PaneStyle.GetHorizontalFrameSize()
	if visualizerInnerWidth < 1 {
		visualizerInnerWidth = 1
	}
	visualizerView := m.statusBar.VisualizerView(m.engine.GetVisualizerBars(visualizerInnerWidth), visualizerInnerWidth, vizHeight)
	leftTopRow := lipgloss.JoinHorizontal(lipgloss.Top, queuePane, contentPane)
	leftColumn := leftTopRow
	if vizOuterHeight > 0 {
		visualizerPane := PaneStyle.Copy().
			Width(visualizerWidth).
			Height(vizHeight + PaneStyle.GetVerticalFrameSize()).
			Render(visualizerView)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftTopRow, visualizerPane)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, nowPlayingPane)

	fullView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		helpView,
	)

	if m.showPlaylistPrompt {
		promptView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Terracotta).
			Padding(1, 2).
			Render("Create New Playlist\n\n" + m.playlistPrompt.View() + "\n\n(Enter to create, Esc to cancel)")

		fullView = m.placeOverlay(fullView, promptView)
	}

	if m.showPlaylistSelector {
		var lines []string
		lines = append(lines, StyleTitle.Render("Add to Playlist"))
		lines = append(lines, "")
		for i, p := range m.allPlaylists {
			if i == m.playlistSelectorIndex {
				lines = append(lines, StyleSelected.Render("> "+p.Name))
			} else {
				lines = append(lines, StyleNormal.Render("  "+p.Name))
			}
		}
		lines = append(lines, "")
		lines = append(lines, StyleMeta.Render("(Enter to select, Esc to cancel)"))

		selectorView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Terracotta).
			Padding(1, 2).
			Render(strings.Join(lines, "\n"))

		fullView = m.placeOverlay(fullView, selectorView)
	}

	return tea.View{
		Content:   fullView,
		AltScreen: true,
	}
}

func (m *Model) refreshLibrary() {
	switch m.activeTab {
	case TabLiked:
		l, _ := m.engine.GetLikedTracks()
		m.liked.SetItems(l, nil, nil, nil, nil)
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
