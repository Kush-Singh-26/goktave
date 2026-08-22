package engine

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

var progressRegex = regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)

func (e *DefaultEngine) downloadTrack(track provider.Track) {
	destPath := filepath.Join(e.cfg.AudioCacheDir, track.VideoID+".webm")

	// 1. If file already exists, no need to download. Just update DB/in-memory if needed.
	if _, err := os.Stat(destPath); err == nil {
		if track.LocalPath == "" {
			track.LocalPath = destPath
			_ = e.db.SaveTrack(track)
		}
		// Update in-memory state
		e.mu.Lock()
		if e.currentTrack != nil && e.currentTrack.VideoID == track.VideoID {
			e.currentTrack.LocalPath = destPath
		}
		for i := range e.queue {
			if e.queue[i].VideoID == track.VideoID {
				e.queue[i].LocalPath = destPath
			}
		}
		e.mu.Unlock()
		return
	}

	// 2. Check if a download is already in progress
	e.mu.Lock()
	if _, exists := e.downloads[track.VideoID]; exists {
		e.mu.Unlock()
		logger.L.Info("Download already in progress", "title", track.Title)
		return
	}
	// Mark as downloading immediately to prevent race conditions
	e.downloads[track.VideoID] = 0
	e.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		tmpPath := destPath + ".tmp"

		logger.L.Info("Starting background download", "title", track.Title)

		defer func() {
			e.mu.Lock()
			delete(e.downloads, track.VideoID)
			e.mu.Unlock()
		}()

		cmd := exec.CommandContext(ctx, e.cfg.YtDlpPath,
			"--no-playlist",
			"--no-warnings",
			"--newline",
			"--progress",
			"--format", "bestaudio[abr<=96][ext=webm]/bestaudio[abr<=96]/bestaudio[ext=webm]/bestaudio",
			"-o", tmpPath,
			"https://youtube.com/watch?v="+track.VideoID,
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.L.Error("Failed to get stdout pipe", "err", err)
			return
		}
		stderrBuf := new(bytes.Buffer)
		cmd.Stderr = stderrBuf

		if err := cmd.Start(); err != nil {
			logger.L.Error("Background download start failed", "title", track.Title, "err", err)
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			matches := progressRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				pct, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					e.mu.Lock()
					e.downloads[track.VideoID] = pct / 100.0
					e.mu.Unlock()
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			logger.L.Error("Background download failed", "title", track.Title,
				"err", err, "stderr", strings.TrimSpace(stderrBuf.String()))
			os.Remove(tmpPath)
			e.setStatus(fmt.Sprintf("✗ Download failed: %s", track.Title), true)
			return
		}

		if err := os.Rename(tmpPath, destPath); err != nil {
			logger.L.Error("Failed to rename temp file", "err", err)
			return
		}

		// Cache lyrics to disk alongside the audio file
		lyricsPath := filepath.Join(e.cfg.AudioCacheDir, track.VideoID+".lrc")
		logger.L.Debug("Background caching lyrics to disk...", "path", lyricsPath)
		go func() {
			lrcCtx, lrcCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer lrcCancel()

			lyricsText, err := e.fetchLrcLibLyrics(lrcCtx, track.Title, track.Artist, track.Duration)
			if (err != nil || lyricsText == "") && e.provider != nil {
				logger.L.Debug("LrcLib background fetch failed, trying YTM fallback", "err", err)
				if _, browseID, nextErr := e.provider.GetUpNext(track.VideoID); nextErr == nil && browseID != "" {
					if ytmLyrics, ytmErr := e.provider.GetLyrics(lrcCtx, browseID); ytmErr == nil && ytmLyrics != "" {
						lyricsText = ytmLyrics
					}
				}
			}
			if lyricsText != "" {
				e.cacheLyricsFile(track.VideoID, lyricsText)
				logger.L.Debug("Successfully cached lyrics on disk", "path", lyricsPath)
			}
		}()

		// Update DB
		track.LocalPath = destPath
		track.LastPlayed = time.Now().Unix()
		if err := e.db.SaveTrack(track); err != nil {
			logger.L.Error("Failed to update track local path in DB", "err", err)
		}

		// Update in-memory state
		e.mu.Lock()
		if e.currentTrack != nil && e.currentTrack.VideoID == track.VideoID {
			e.currentTrack.LocalPath = destPath
		}
		for i := range e.queue {
			if e.queue[i].VideoID == track.VideoID {
				e.queue[i].LocalPath = destPath
			}
		}
		e.mu.Unlock()

		logger.L.Info("Background download complete", "title", track.Title)
		e.setStatus(fmt.Sprintf("✓ Cached: %s", track.Title), false)

		// Run pruning check
		e.pruneCache()
	}()
}

// pruneCache evicts oldest-played audio files until the cache fits the
// configured size budget. It only touches the DB and filesystem, so no
// engine lock is held (a full disk walk would stall the UI otherwise).
func (e *DefaultEngine) pruneCache() {
	tracks, err := e.db.GetTracksByLastPlayed()
	if err != nil {
		return
	}

	// Sort by last played (oldest first)
	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].LastPlayed < tracks[j].LastPlayed
	})

	maxSizeBytes := int64(e.cfg.MaxCacheSizeGB * 1024 * 1024 * 1024)

	totalSize := e.getCacheSize()
	for _, t := range tracks {
		if totalSize <= maxSizeBytes {
			break
		}

		if t.LocalPath != "" {
			info, err := os.Stat(t.LocalPath)
			if err == nil {
				size := info.Size()
				pruned := t.LocalPath
				if err := os.Remove(t.LocalPath); err == nil {
					totalSize -= size
					t.LocalPath = ""
					_ = e.db.SaveTrack(t)
					logger.L.Info("Pruned cache file", "path", pruned)

					// Also prune the matching .lrc file if it exists
					lrcPath := filepath.Join(e.cfg.AudioCacheDir, t.VideoID+".lrc")
					_ = os.Remove(lrcPath)
				}
			}
		}
	}
}

func (e *DefaultEngine) getCacheSize() int64 {
	var size int64
	err := filepath.Walk(e.cfg.AudioCacheDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".tmp") {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return size
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

func (e *DefaultEngine) DownloadTrack(track provider.Track) {
	e.downloadTrack(track)
}
