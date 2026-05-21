package engine

import (
	"context"
	"os"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

func (e *DefaultEngine) Play(track provider.Track) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.playLockedWithOffset(track, true, 0)
}

func (e *DefaultEngine) playLocked(track provider.Track, addToHistory bool) error {
	return e.playLockedWithOffset(track, addToHistory, 0)
}

func (e *DefaultEngine) playLockedWithOffset(track provider.Track, addToHistory bool, offset time.Duration) error {
	// Try to get enriched metadata from DB
	if dbTrack, err := e.db.GetTrack(track.VideoID); err == nil {
		track = *dbTrack
	}

	logger.L.Info("Engine playing track", "title", track.Title, "id", track.VideoID, "offset", offset)

	e.currentLyrics = "Fetching lyrics..."
	e.lyricsBrowseID = ""

	if e.mpris != nil {
		e.mpris.UpdateMetadata(&track)
		e.mpris.UpdatePosition(offset)
		e.mpris.UpdateStatus("Playing")
	}

	if e.cancel != nil {
		e.cancel()
	}
	e.player.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	if addToHistory && e.currentTrack != nil {
		isDuplicate := false
		if len(e.history) > 0 {
			lastTrack := e.history[len(e.history)-1]
			if lastTrack.VideoID == e.currentTrack.VideoID {
				isDuplicate = true
			}
		}
		if !isDuplicate {
			e.history = append(e.history, *e.currentTrack)
			if len(e.history) > 50 {
				e.history = e.history[1:]
			}
			_ = e.db.AddToHistory(*e.currentTrack)
		}
	}

	if e.currentTrack == nil || e.currentTrack.VideoID != track.VideoID {
		e.currentStreamURL = ""
	}

	trackCopy := track
	e.currentTrack = &trackCopy
	e.currentTrackStart = time.Now().Add(-offset)
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
			for i := 1; i < 21 && i < len(tracks); i++ {
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

		// 3. Check if cached stream URL exists
		if streamURL == "" {
			e.mu.Lock()
			if e.currentStreamURL != "" {
				streamURL = e.currentStreamURL
			}
			e.mu.Unlock()
		}

		// 4. Extract fresh URL
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

		e.mu.Lock()
		e.currentStreamURL = streamURL
		e.mu.Unlock()

		if err := e.player.PlayWithOffset(streamURL, offset); err != nil {
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
		if ctx.Err() != nil {
			e.mu.Unlock()
			return
		}
		if err == nil && e.currentTrack != nil && e.currentTrack.VideoID == track.VideoID {
			e.mu.Unlock()
			e.Next()
		} else {
			e.mu.Unlock()
		}
	}()

	return nil
}

func (e *DefaultEngine) Seek(offset time.Duration) error {
	e.mu.Lock()
	if e.currentTrack == nil {
		e.mu.Unlock()
		return nil
	}
	track := *e.currentTrack
	e.mu.Unlock()

	if offset < 0 {
		offset = 0
	}
	if track.Duration > 0 && offset.Seconds() > float64(track.Duration) {
		offset = time.Duration(track.Duration) * time.Second
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	return e.playLockedWithOffset(track, false, offset)
}

func (e *DefaultEngine) Stop() {
	e.player.Stop()
	if e.mpris != nil {
		e.mpris.UpdatePosition(0)
		e.mpris.UpdateStatus("Stopped")
	}
}

func (e *DefaultEngine) TogglePause() bool {
	e.mu.Lock()
	if e.player.State() == player.StateStopped && e.currentTrack != nil {
		t := *e.currentTrack
		e.mu.Unlock()
		_ = e.playLocked(t, false)
		return false
	}
	e.mu.Unlock()

	paused := e.player.TogglePause()
	if e.mpris != nil {
		pos := e.player.Position()
		if paused {
			// Sync position cache immediately before signalling Paused.
			// Some widgets query Position right after receiving PlaybackStatus=Paused.
			e.mpris.UpdatePosition(pos)
			e.mpris.UpdateStatus("Paused")
		} else {
			// Emit Seeked so widgets reset their local position tracker to the
			// exact resume point instead of extrapolating from a stale value.
			e.mpris.EmitSeeked(pos)
			e.mpris.UpdateStatus("Playing")
		}
	}
	return paused
}
