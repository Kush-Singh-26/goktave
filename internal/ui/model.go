package ui

import (
	"context"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

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
