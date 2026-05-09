package engine

import (
	"bufio"
	"context"
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
	// Don't download if already exists
	if track.LocalPath != "" {
		if _, err := os.Stat(track.LocalPath); err == nil {
			return
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		destPath := filepath.Join(e.cfg.AudioCacheDir, track.VideoID+".webm")
		tmpPath := destPath + ".tmp"

		logger.L.Info("Starting background download", "title", track.Title)

		e.mu.Lock()
		e.downloads[track.Title] = 0
		e.mu.Unlock()
		defer func() {
			e.mu.Lock()
			delete(e.downloads, track.Title)
			e.mu.Unlock()
		}()

		cmd := exec.CommandContext(ctx, e.cfg.YtDlpPath,
			"--no-playlist",
			"--no-warnings",
			"--newline",
			"--progress",
			"--format", "bestaudio[ext=webm]/bestaudio",
			"-o", tmpPath,
			"https://youtube.com/watch?v="+track.VideoID,
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.L.Error("Failed to get stdout pipe", "err", err)
			return
		}

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
					e.downloads[track.Title] = pct / 100.0
					e.mu.Unlock()
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			logger.L.Error("Background download failed", "title", track.Title, "err", err)
			os.Remove(tmpPath)
			return
		}

		if err := os.Rename(tmpPath, destPath); err != nil {
			logger.L.Error("Failed to rename temp file", "err", err)
			return
		}

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
		
		// Run pruning check
		e.pruneCache()
	}()
}

func (e *DefaultEngine) pruneCache() {
	e.mu.Lock()
	defer e.mu.Unlock()

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
				if err := os.Remove(t.LocalPath); err == nil {
					totalSize -= size
					t.LocalPath = ""
					_ = e.db.SaveTrack(t)
					logger.L.Info("Pruned cache file", "path", t.LocalPath)
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
