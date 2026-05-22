package ui

import (
	"context"
	"time"

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

type SyncedLine struct {
	Time time.Duration
	Text string
}

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
	syncedLines    []SyncedLine
	lastActiveLine int

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

	vinylFrame int

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

	// Apply saved accent override
	presetIdx := 0
	if cfg.AccentOverride != "" {
		for i, p := range AccentPresets {
			if p == cfg.AccentOverride {
				presetIdx = i
				break
			}
		}
		SetAccentColor(cfg.AccentOverride)
	}

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

	m := Model{
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
		settings:        SettingsView{engine: e, ThemeIndex: themeIdx, AccentPresetIdx: presetIdx},
		lyrics:          viewport.New(),
		help:            h,
		lastState:       -1,
		suggestionIndex: -1,
		lastActiveLine:  -2,
	}
	m.statusBar = NewStatusBar()
	m.statusBar.SetVizMode(VizColorMode(cfg.VizMode))
	return m
}
