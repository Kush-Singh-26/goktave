package engine

import (
	"context"
	"sync"

	"github.com/Kush-Singh-26/goktave/internal/config"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/logger"
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
}

type DefaultEngine struct {
	mu sync.Mutex

	cfg       *config.Config
	provider  provider.Provider
	extractor extractor.Extractor
	player    player.AudioPlayer

	queue        []provider.Track
	currentTrack *provider.Track
	cancel       context.CancelFunc
}

func New(cfg *config.Config, prov provider.Provider, ext extractor.Extractor, pl player.AudioPlayer) *DefaultEngine {
	e := &DefaultEngine{
		cfg:       cfg,
		provider:  prov,
		extractor: ext,
		player:    pl,
	}
	return e
}

func (e *DefaultEngine) Search(ctx context.Context, query string) ([]provider.Track, error) {
	logger.L.Debug("Engine search", "query", query)
	return e.provider.Search(ctx, query)
}

func (e *DefaultEngine) Play(track provider.Track) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	logger.L.Info("Engine playing track", "title", track.Title, "id", track.VideoID)
	
	if e.cancel != nil {
		e.cancel()
	}
	e.player.Stop() // Stop old track immediately

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	e.currentTrack = &track

	go func() {
		info, err := e.extractor.Extract(ctx, track.VideoID)
		if err != nil {
			logger.L.Error("failed to extract stream", "err", err)
			return
		}

		if err := e.player.Play(info.URL); err != nil {
			logger.L.Error("failed to play stream", "err", err)
			return
		}

		// Wait for track to end
		err = e.player.Wait()
		e.mu.Lock()
		if err == nil && e.currentTrack != nil && e.currentTrack.VideoID == track.VideoID {
			e.mu.Unlock()
			// Track ended naturally, play next
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
	logger.L.Info("Adding to queue", "title", track.Title)
	e.queue = append(e.queue, track)
}

func (e *DefaultEngine) Next() error {
	e.mu.Lock()
	
	if len(e.queue) == 0 {
		// If queue is empty, we could potentially trigger auto-radio here
		// For now just stop
		e.mu.Unlock()
		if e.currentTrack != nil {
			logger.L.Info("Queue empty, fetching radio for", "title", e.currentTrack.Title)
			results, err := e.provider.GetUpNext(e.currentTrack.VideoID)
			if err == nil && len(results) > 1 {
				// Add a few tracks from radio
				for i := 1; i < 6 && i < len(results); i++ {
					e.Queue(results[i])
				}
				return e.Next()
			}
		}
		return nil
	}

	next := e.queue[0]
	e.queue = e.queue[1:]
	e.mu.Unlock()

	return e.Play(next)
}

func (e *DefaultEngine) Stop() {
	e.player.Stop()
}

func (e *DefaultEngine) TogglePause() bool {
	return e.player.TogglePause()
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
