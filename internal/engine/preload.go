package engine

import (
	"context"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
)

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
	// Check if this track is already preloaded
	if e.preloadID == nextTrack.VideoID && e.preloadURL != "" {
		e.mu.Unlock()
		return
	}

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
