package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
	"github.com/Kush-Singh-26/goktave/internal/db"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/mpris"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
	"github.com/Kush-Singh-26/goktave/internal/thumbnail"
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
	IsPreloading() bool
	Preload()
	HasPreloaded(id string) bool
	RemoveFromQueue(index int)
	ClearQueue()
	MoveInQueue(fromIndex, toIndex int)
	SetMPRIS(m *mpris.Manager)
	PlayFromQueue(index int) error
	GetLyrics() string
	GetVisualizerBars(n int) []float64
	GetASCIIThumbnail(track provider.Track, width int) (string, error)
	GetTrack(videoID string) (*provider.Track, error)
	CreatePlaylist(name string) error
	DeletePlaylist(name string) error
	GetPlaylists() ([]db.Playlist, error)
	AddTrackToPlaylist(playlistName string, videoID string) error
	RemoveTrackFromPlaylist(playlistName string, videoID string) error
	GetPlaylistTracks(name string) ([]provider.Track, error)
	PlayPlaylist(name string) error
	GetActiveDownloads() map[string]float64
	GetDownloadedTracks() ([]provider.Track, error)
	DeleteDownload(videoID string) error
	ClearCache() error
	GetConfig() *config.Config
	SaveConfig() error
	GetCacheSize() int64

	DownloadTrack(track provider.Track)
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

	queue          []provider.Track
	history        []provider.Track
	currentTrack      *provider.Track
	currentTrackStart time.Time
	currentLyrics     string
	lyricsBrowseID string
	cancel         context.CancelFunc
	retryCount     int

	downloads map[string]float64
}

func (e *DefaultEngine) saveQueue() {
	// Assumes lock is held or not needed
	data, err := json.Marshal(e.queue)
	if err != nil {
		logger.L.Error("failed to marshal queue", "err", err)
		return
	}
	if err := os.WriteFile(e.cfg.QueuePath, data, 0644); err != nil {
		logger.L.Error("failed to save queue", "err", err)
	}
}

func (e *DefaultEngine) loadQueue() {
	data, err := os.ReadFile(e.cfg.QueuePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.L.Error("failed to read queue file", "err", err)
		}
		return
	}
	var q []provider.Track
	if err := json.Unmarshal(data, &q); err != nil {
		logger.L.Error("failed to unmarshal queue", "err", err)
		return
	}
	e.queue = q
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
	e.loadHistory()
	e.loadState()
	return e
}

func (e *DefaultEngine) saveState() {
	if e.currentTrack != nil {
		_ = e.db.SaveState("last_track", e.currentTrack)
	}
}

func (e *DefaultEngine) loadState() {
	var t provider.Track
	if err := e.db.GetState("last_track", &t); err == nil {
		trackCopy := t
		e.currentTrack = &trackCopy
	}
}

func (e *DefaultEngine) loadHistory() {
	h, err := e.db.GetHistory(50)
	if err != nil {
		logger.L.Error("failed to load history from db", "err", err)
		return
	}
	// DB returns history in reverse chronological order (newest first)
	// We want the in-memory history slice to be chronological (oldest first)
	for i := len(h) - 1; i >= 0; i-- {
		e.history = append(e.history, h[i])
	}
}

func (e *DefaultEngine) IsPreloading() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isPreloading
}

func (e *DefaultEngine) HasPreloaded(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.preloadID == id && e.preloadURL != ""
}

func (e *DefaultEngine) Preload() {
	e.mu.Lock()
	if e.isPreloading || len(e.queue) == 0 {
		e.mu.Unlock()
		return
	}

	nextTrack := e.queue[0]
	e.isPreloading = true
	e.mu.Unlock()

	logger.L.Debug("Engine preloading next track", "title", nextTrack.Title)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		urlInfo, err := e.extractor.Extract(ctx, nextTrack.VideoID)

		e.mu.Lock()
		defer e.mu.Unlock()

		e.isPreloading = false

		if err != nil {
			logger.L.Error("failed to preload track", "err", err)
			return
		}

		e.preloadID = nextTrack.VideoID
		e.preloadURL = urlInfo.URL
		logger.L.Info("preload successful", "title", nextTrack.Title)
	}()
}

func (e *DefaultEngine) Search(ctx context.Context, query string) ([]provider.Track, error) {
	logger.L.Debug("Engine search", "query", query)
	_ = e.db.AddSearch(query)
	tracks, err := e.provider.Search(ctx, query)
	if err == nil {
		for i, t := range tracks {
			// Enrich with local info if available
			if dbTrack, err := e.db.GetTrack(t.VideoID); err == nil {
				tracks[i].LocalPath = dbTrack.LocalPath
				// Also sync other metadata like like status if we want, 
				// but results view handles it via e.IsLiked
			}
			_ = e.db.SaveTrack(tracks[i])
		}
	}
	return tracks, err
}

func (e *DefaultEngine) GetSuggestions(ctx context.Context, input string) ([]string, error) {
	return e.provider.GetSuggestions(ctx, input)
}

func (e *DefaultEngine) Prev() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// If playing for more than 3 seconds, just restart
	if e.currentTrack != nil && time.Since(e.currentTrackStart) > 3*time.Second {
		return e.playLocked(*e.currentTrack, false)
	}

	if len(e.history) == 0 {
		return nil
	}

	prev := e.history[len(e.history)-1]
	e.history = e.history[:len(e.history)-1]

	if e.currentTrack != nil {
		e.queue = append([]provider.Track{*e.currentTrack}, e.queue...)
	}

	return e.playLocked(prev, false)
}

func (e *DefaultEngine) ToggleLike(videoID string) (bool, error) {
	return e.db.ToggleLike(videoID)
}

func (e *DefaultEngine) IsLiked(videoID string) bool {
	return e.db.IsLiked(videoID)
}

func (e *DefaultEngine) GetLikedTracks() ([]provider.Track, error) {
	return e.db.GetLikedTracks()
}

func (e *DefaultEngine) GetHistory(limit int) ([]provider.Track, error) {
	return e.db.GetHistory(limit)
}

func (e *DefaultEngine) GetSearchHistory(limit int) ([]string, error) {
	return e.db.GetSearchHistory(limit)
}

func (e *DefaultEngine) SetMPRIS(m *mpris.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mpris = m
}

func (e *DefaultEngine) GetLyrics() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.currentLyrics == "" {
		return "No lyrics available for this track."
	}
	return e.currentLyrics
}

func (e *DefaultEngine) GetVisualizerBars(n int) []float64 {
	return e.player.GetVisualizerBars(n)
}

func (e *DefaultEngine) Play(track provider.Track) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.playLocked(track, true)
}

func (e *DefaultEngine) playLocked(track provider.Track, addToHistory bool) error {
	// Try to get enriched metadata from DB
	if dbTrack, err := e.db.GetTrack(track.VideoID); err == nil {
		track = *dbTrack
	}

	logger.L.Info("Engine playing track", "title", track.Title, "id", track.VideoID)

	e.currentLyrics = "Fetching lyrics..."
	e.lyricsBrowseID = ""

	if e.mpris != nil {
		e.mpris.UpdateMetadata(&track)
		e.mpris.UpdateStatus("Playing")
	}

	if e.cancel != nil {
		e.cancel()
	}
	e.player.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	if addToHistory && e.currentTrack != nil {
		e.history = append(e.history, *e.currentTrack)
		if len(e.history) > 50 {
			e.history = e.history[1:]
		}
		_ = e.db.AddToHistory(*e.currentTrack)
	}

	trackCopy := track
	e.currentTrack = &trackCopy
	e.currentTrackStart = time.Now()
	e.saveState()

	// Background task for lyrics and suggestions
	go func() {
		logger.L.Debug("Engine fetching UpNext", "videoId", track.VideoID)
		tracks, browseID, err := e.provider.GetUpNext(track.VideoID)
		if err != nil {
			logger.L.Error("Engine UpNext failed", "err", err)
			return
		}

		logger.L.Debug("Engine UpNext success", "tracksCount", len(tracks), "browseID", browseID)

		e.mu.Lock()
		e.lyricsBrowseID = browseID
		
		// Refresh current track metadata if thumbnails are missing (e.g. from old DB/Queue)
		if e.currentTrack != nil && e.currentTrack.VideoID == track.VideoID && e.currentTrack.ThumbURL == "" && len(tracks) > 0 {
			if tracks[0].VideoID == track.VideoID {
				e.currentTrack.ThumbURL = tracks[0].ThumbURL
			}
		}

		if len(e.queue) == 0 && len(tracks) > 1 {
			for i := 1; i < 6 && i < len(tracks); i++ {
				e.queue = append(e.queue, tracks[i])
			}
			e.saveQueue()
		}
		e.mu.Unlock()

		if browseID != "" {
			logger.L.Debug("Engine fetching lyrics", "browseID", browseID)
			lyrics, err := e.provider.GetLyrics(ctx, browseID)
			e.mu.Lock()
			if err == nil {
				logger.L.Debug("Engine lyrics success", "len", len(lyrics))
				e.currentLyrics = lyrics
			} else {
				logger.L.Error("Engine lyrics failed", "err", err)
				e.currentLyrics = "Could not fetch lyrics."
			}
			e.mu.Unlock()
		} else {
			e.mu.Lock()
			e.currentLyrics = "No lyrics available for this track."
			e.mu.Unlock()
		}
	}()

	go func() {
		var streamURL string

		// 1. Check if local file exists (Highest priority)
		if track.LocalPath != "" {
			if _, err := os.Stat(track.LocalPath); err == nil {
				logger.L.Info("Playing from local cache", "title", track.Title)
				streamURL = track.LocalPath
				
				// Update LastPlayed
				track.LastPlayed = time.Now().Unix()
				_ = e.db.SaveTrack(track)
			}
		}

		// 2. Check if preloaded URL exists
		if streamURL == "" {
			e.mu.Lock()
			if e.preloadID == track.VideoID && e.preloadURL != "" {
				logger.L.Info("Using preloaded URL", "title", track.Title)
				streamURL = e.preloadURL
				e.preloadID = ""
				e.preloadURL = ""
			}
			e.mu.Unlock()
		}

		// 3. Extract fresh URL
		if streamURL == "" {
			info, err := e.extractor.Extract(ctx, track.VideoID)
			if err != nil {
				logger.L.Error("failed to extract stream", "err", err)
				e.mu.Lock()
				// Only auto-skip if it was a transition (addToHistory=true)
				if addToHistory && e.retryCount < 3 {
					e.retryCount++
					e.mu.Unlock()
					e.Next()
				} else {
					e.retryCount = 0
					e.mu.Unlock()
				}
				return
			}
			streamURL = info.URL
			
			// Trigger background download
			e.downloadTrack(track)
		}

		if err := e.player.Play(streamURL); err != nil {
			logger.L.Error("failed to play stream", "err", err)
			e.mu.Lock()
			// Only auto-skip if it was a transition (addToHistory=true)
			if addToHistory && e.retryCount < 3 {
				e.retryCount++
				e.mu.Unlock()
				e.Next()
			} else {
				e.retryCount = 0
				e.mu.Unlock()
			}
			return
		}

		e.mu.Lock()
		e.retryCount = 0
		e.mu.Unlock()

		err := e.player.Wait()
		e.mu.Lock()
		if err == nil && e.currentTrack != nil && e.currentTrack.VideoID == track.VideoID {
			e.mu.Unlock()
			e.Next()
		} else {
			e.mu.Unlock()
		}
	}()

	return nil
}

func (e *DefaultEngine) Queue(track provider.Track) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.queueLocked(track)
}

func (e *DefaultEngine) queueLocked(track provider.Track) {
	logger.L.Info("Adding to queue", "title", track.Title)
	e.queue = append(e.queue, track)
	e.saveQueue()
}

func (e *DefaultEngine) Next() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.nextLocked()
}

func (e *DefaultEngine) nextLocked() error {
	if len(e.queue) == 0 {
		track := e.currentTrack
		if track != nil {
			logger.L.Info("Queue empty, fetching radio for", "title", track.Title)
			// Unlock briefly for network call if we want, but provider calls are usually fine
			// Actually, GetUpNext is a network call, we should NOT hold the lock.
			e.mu.Unlock()
			results, _, err := e.provider.GetUpNext(track.VideoID)
			e.mu.Lock()

			if err == nil && len(results) > 1 {
				for i := 1; i < 6 && i < len(results); i++ {
					e.queueLocked(results[i])
				}
				return e.nextLocked()
			}
		}
		return nil
	}

	next := e.queue[0]
	e.queue = e.queue[1:]
	e.saveQueue()

	return e.playLocked(next, true)
}

func (e *DefaultEngine) Stop() {
	e.player.Stop()
	if e.mpris != nil {
		e.mpris.UpdateStatus("Stopped")
	}
}

func (e *DefaultEngine) TogglePause() bool {
	e.mu.Lock()
	if e.player.State() == player.StateStopped && e.currentTrack != nil {
		t := *e.currentTrack
		e.mu.Unlock()
		_ = e.playLocked(t, false)
		if e.mpris != nil {
			e.mpris.UpdateStatus("Playing")
		}
		return false
	}
	e.mu.Unlock()

	paused := e.player.TogglePause()
	if e.mpris != nil {
		if paused {
			e.mpris.UpdateStatus("Paused")
		} else {
			e.mpris.UpdateStatus("Playing")
		}
	}
	return paused
}

func (e *DefaultEngine) SetVolume(v float64) {
	e.player.SetVolume(v)
}

func (e *DefaultEngine) GetVolume() float64 {
	return e.player.GetVolume()
}

func (e *DefaultEngine) GetQueue() []provider.Track {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.queue
}

func (e *DefaultEngine) GetCurrentTrack() *provider.Track {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.currentTrack
}

func (e *DefaultEngine) GetState() player.State {
	return e.player.State()
}

func (e *DefaultEngine) RemoveFromQueue(index int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if index < 0 || index >= len(e.queue) {
		return
	}
	e.queue = append(e.queue[:index], e.queue[index+1:]...)
	e.saveQueue()
}

func (e *DefaultEngine) ClearQueue() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.queue = nil
	e.saveQueue()
}

func (e *DefaultEngine) PlayFromQueue(index int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if index < 0 || index >= len(e.queue) {
		return nil
	}
	track := e.queue[index]
	e.queue = append(e.queue[:index], e.queue[index+1:]...)
	e.saveQueue()

	return e.playLocked(track, true)
}

func (e *DefaultEngine) MoveInQueue(from, to int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if from < 0 || from >= len(e.queue) || to < 0 || to >= len(e.queue) {
		return
	}
	track := e.queue[from]
	e.queue = append(e.queue[:from], e.queue[from+1:]...)

	newQueue := make([]provider.Track, 0, len(e.queue)+1)
	newQueue = append(newQueue, e.queue[:to]...)
	newQueue = append(newQueue, track)
	newQueue = append(newQueue, e.queue[to:]...)
	e.queue = newQueue
	e.saveQueue()
}

func (e *DefaultEngine) GetASCIIThumbnail(track provider.Track, width int) (string, error) {
	if track.ThumbURL == "" {
		return "", nil
	}

	// Try to get from DB first to see if we have it cached for this width
	t, err := e.db.GetTrack(track.VideoID)
	if err == nil && t.ThumbASCII != "" && t.ThumbWidth == width {
		return t.ThumbASCII, nil
	}

	// Not cached or width changed, generate it
	ascii, err := thumbnail.GetASCII(track.ThumbURL, width)
	if err != nil {
		return "", err
	}

	// Update track metadata and save to DB
	track.ThumbASCII = ascii
	track.ThumbWidth = width
	_ = e.db.SaveTrack(track)

	return ascii, nil
}

func (e *DefaultEngine) GetTrack(videoID string) (*provider.Track, error) {
	return e.db.GetTrack(videoID)
}

func (e *DefaultEngine) CreatePlaylist(name string) error {
	return e.db.CreatePlaylist(name)
}

func (e *DefaultEngine) DeletePlaylist(name string) error {
	return e.db.DeletePlaylist(name)
}

func (e *DefaultEngine) GetPlaylists() ([]db.Playlist, error) {
	return e.db.GetPlaylists()
}

func (e *DefaultEngine) AddTrackToPlaylist(playlistName string, videoID string) error {
	return e.db.AddTrackToPlaylist(playlistName, videoID)
}

func (e *DefaultEngine) RemoveTrackFromPlaylist(playlistName string, videoID string) error {
	return e.db.RemoveTrackFromPlaylist(playlistName, videoID)
}

func (e *DefaultEngine) GetPlaylistTracks(name string) ([]provider.Track, error) {
	return e.db.GetPlaylistTracks(name)
}

func (e *DefaultEngine) PlayPlaylist(name string) error {
	tracks, err := e.db.GetPlaylistTracks(name)
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.queue = tracks[1:]
	e.saveQueue()
	return e.playLocked(tracks[0], true)
}

func (e *DefaultEngine) GetActiveDownloads() map[string]float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Return a copy to avoid race conditions
	copy := make(map[string]float64)
	for k, v := range e.downloads {
		copy[k] = v
	}
	return copy
}

func (e *DefaultEngine) GetDownloadedTracks() ([]provider.Track, error) {
	return e.db.GetDownloadedTracks()
}

func (e *DefaultEngine) DeleteDownload(videoID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	track, err := e.db.GetTrack(videoID)
	if err != nil {
		return err
	}

	if track.LocalPath != "" {
		_ = os.Remove(track.LocalPath)
		track.LocalPath = ""
		return e.db.SaveTrack(*track)
	}
	return nil
}

func (e *DefaultEngine) ClearCache() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tracks, err := e.db.GetDownloadedTracks()
	if err != nil {
		return err
	}

	for _, t := range tracks {
		if t.LocalPath != "" {
			_ = os.Remove(t.LocalPath)
			t.LocalPath = ""
			_ = e.db.SaveTrack(t)
		}
	}

	// Also clear any remaining files in the cache dir just in case
	files, _ := filepath.Glob(filepath.Join(e.cfg.AudioCacheDir, "*"))
	for _, f := range files {
		_ = os.Remove(f)
	}

	return nil
}

func (e *DefaultEngine) GetCacheSize() int64 {
	return e.getCacheSize()
}

func (e *DefaultEngine) DownloadTrack(track provider.Track) {
	e.downloadTrack(track)
}

func (e *DefaultEngine) GetConfig() *config.Config {
	return e.cfg
}

func (e *DefaultEngine) SaveConfig() error {
	return e.db.SaveConfig(e.cfg)
}
