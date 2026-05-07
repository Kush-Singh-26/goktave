package engine

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/mpris"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type Engine interface {
	Search(ctx context.Context, query string) ([]provider.Track, error)
	Play(track provider.Track) error
	Queue(track provider.Track)
	Next() error
	Stop()
	TogglePause() bool
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
}

type DefaultEngine struct {
	mu sync.Mutex

	cfg       *config.Config
	provider  provider.Provider
	extractor extractor.Extractor
	player    player.AudioPlayer
	mpris     *mpris.Manager

	preloadID    string
	preloadURL   string
	isPreloading bool

	queue          []provider.Track
	currentTrack   *provider.Track
	currentLyrics  string
	lyricsBrowseID string
	cancel         context.CancelFunc
	retryCount     int
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

func New(cfg *config.Config, prov provider.Provider, ext extractor.Extractor, pl player.AudioPlayer) *DefaultEngine {
	e := &DefaultEngine{
		cfg:       cfg,
		provider:  prov,
		extractor: ext,
		player:    pl,
	}
	e.loadQueue()
	return e
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
	return e.provider.Search(ctx, query)
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
	return e.playLocked(track)
}

func (e *DefaultEngine) playLocked(track provider.Track) error {
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

	e.currentTrack = &track

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

		e.mu.Lock()
		if e.preloadID == track.VideoID && e.preloadURL != "" {
			logger.L.Info("Using preloaded URL", "title", track.Title)
			streamURL = e.preloadURL
			e.preloadID = ""
			e.preloadURL = ""
		}
		e.mu.Unlock()

		if streamURL == "" {
			info, err := e.extractor.Extract(ctx, track.VideoID)
			if err != nil {
				logger.L.Error("failed to extract stream", "err", err)
				e.mu.Lock()
				e.retryCount++
				if e.retryCount < 3 {
					e.mu.Unlock()
					e.Next()
				} else {
					e.retryCount = 0
					e.mu.Unlock()
				}
				return
			}
			streamURL = info.URL
		}

		if err := e.player.Play(streamURL); err != nil {
			logger.L.Error("failed to play stream", "err", err)
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

	return e.playLocked(next)
}

func (e *DefaultEngine) Stop() {
	e.player.Stop()
	if e.mpris != nil {
		e.mpris.UpdateStatus("Stopped")
	}
}

func (e *DefaultEngine) TogglePause() bool {
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

	return e.playLocked(track)
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
