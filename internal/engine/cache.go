package engine

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	"github.com/Kush-Singh-26/goktave/internal/thumbnail"
)

// thumbCachePrefix marks cached art as native half-block output, so
// stale ascii-image-converter entries from older versions regenerate.
const thumbCachePrefix = "hb:"

func (e *DefaultEngine) GetThumbnailArt(track provider.Track, cols int) (string, error) {
	if track.ThumbURL == "" {
		return "", nil
	}

	// Try to get from DB first to see if we have it cached for this size
	t, err := e.db.GetTrack(track.VideoID)
	if err == nil && strings.HasPrefix(t.ThumbASCII, thumbCachePrefix) && t.ThumbWidth == cols {
		return strings.TrimPrefix(t.ThumbASCII, thumbCachePrefix), nil
	}

	// Not cached or size changed, generate it
	art, err := thumbnail.Render(track.ThumbURL, cols)
	if err != nil {
		return "", err
	}

	// Update track metadata and save to DB
	track.ThumbASCII = thumbCachePrefix + art
	track.ThumbWidth = cols
	_ = e.db.SaveTrack(track)

	return art, nil
}

func (e *DefaultEngine) GetTrack(videoID string) (*provider.Track, error) {
	return e.db.GetTrack(videoID)
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
