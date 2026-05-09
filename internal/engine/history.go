package engine

import (
	"context"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

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

func (e *DefaultEngine) GetHistory(limit int) ([]provider.Track, error) {
	return e.db.GetHistory(limit)
}

func (e *DefaultEngine) GetSearchHistory(limit int) ([]string, error) {
	return e.db.GetSearchHistory(limit)
}
