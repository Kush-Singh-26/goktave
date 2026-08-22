package engine

import (
	"context"
	"sync"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
	"github.com/Kush-Singh-26/goktave/internal/db"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/mpris"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type Engine interface {
	Search(ctx context.Context, query string) ([]provider.Track, error)
	GetSuggestions(ctx context.Context, input string) ([]string, error)
	Play(track provider.Track) error
	Queue(track provider.Track)
	Next() error
	Prev() error
	Stop()
	TogglePause() bool
	Seek(offset time.Duration) error
	ToggleLike(videoID string) (bool, error)
	IsLiked(videoID string) bool
	GetLikedTracks() ([]provider.Track, error)
	GetHistory(limit int) ([]provider.Track, error)
	GetSearchHistory(limit int) ([]string, error)
	SetVolume(v float64)
	GetVolume() float64
	GetQueue() []provider.Track
	GetCurrentTrack() *provider.Track
	GetState() player.State
	GetPlayPosition() time.Duration
	IsPreloading() bool
	Preload()
	HasPreloaded(id string) bool
	RemoveFromQueue(index int)
	ClearQueue()
	MoveInQueue(fromIndex, toIndex int)
	Shuffle()
	SetMPRIS(m *mpris.Manager)
	UpdateMPRISPosition()
	PlayFromQueue(index int) error
	GetLyrics() string
	GetVisualizerBars(n int) []float64
	GetThumbnailArt(track provider.Track, cols int) (string, error)
	GetTrack(videoID string) (*provider.Track, error)
	CreatePlaylist(name string) error
	DeletePlaylist(name string) error
	GetPlaylists() ([]db.Playlist, error)
	AddTrackToPlaylist(playlistName string, videoID string) error
	RemoveTrackFromPlaylist(playlistName string, videoID string) error
	GetPlaylistTracks(name string) ([]provider.Track, error)
	PlayPlaylist(name string) error
	PlayTracks(tracks []provider.Track) error
	GetActiveDownloads() map[string]float64
	GetDownloadedTracks() ([]provider.Track, error)
	DeleteDownload(videoID string) error
	ClearCache() error
	GetConfig() *config.Config
	SaveConfig() error
	GetCacheSize() int64

	// PollStatus returns the most recent transient status event (e.g. a
	// playback failure) if one has occurred since the last poll.
	PollStatus() (StatusEvent, bool)

	DownloadTrack(track provider.Track)
}

// StatusEvent is a short-lived message destined for the UI status area.
type StatusEvent struct {
	Message string
	IsError bool
}

type DefaultEngine struct {
	mu sync.Mutex

	cfg       *config.Config
	provider  provider.Provider
	extractor extractor.Extractor
	player    player.AudioPlayer
	mpris     *mpris.Manager
	db        *db.DB

	preloadID    string
	preloadURL   string
	isPreloading bool

	queue             []provider.Track
	history           []provider.Track
	currentTrack      *provider.Track
	currentStreamURL  string
	currentTrackStart time.Time
	currentLyrics     string
	lyricsBrowseID    string
	cancel            context.CancelFunc
	retryCount        int

	downloads map[string]float64

	statusMsg     string
	statusIsError bool
}

func New(cfg *config.Config, prov provider.Provider, ext extractor.Extractor, pl player.AudioPlayer, database *db.DB) *DefaultEngine {
	e := &DefaultEngine{
		cfg:       cfg,
		provider:  prov,
		extractor: ext,
		player:    pl,
		db:        database,
		downloads: make(map[string]float64),
	}
	e.loadQueue()
	e.loadState()
	return e
}

func (e *DefaultEngine) setStatus(msg string, isError bool) {
	e.mu.Lock()
	e.statusMsg = msg
	e.statusIsError = isError
	e.mu.Unlock()
}

func (e *DefaultEngine) PollStatus() (StatusEvent, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.statusMsg == "" {
		return StatusEvent{}, false
	}
	ev := StatusEvent{Message: e.statusMsg, IsError: e.statusIsError}
	e.statusMsg = ""
	return ev, true
}
