package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
)

// lrcLibAPIBase is the LrcLib GET endpoint; overridden in tests.
var lrcLibAPIBase = "https://lrclib.net/api/get"

type LrcLibResponse struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

func cleanArtist(artist string) string {
	artist = strings.ToLower(artist)
	artist = strings.ReplaceAll(artist, " - topic", "")
	artist = strings.ReplaceAll(artist, "-topic", "")
	artist = strings.ReplaceAll(artist, "official", "")
	artist = strings.ReplaceAll(artist, "vevo", "")
	return strings.TrimSpace(artist)
}

func cleanTitle(title, artist string) string {
	title = strings.ToLower(title)

	// Remove any topic suffix
	title = strings.ReplaceAll(title, " - topic", "")
	title = strings.ReplaceAll(title, "-topic", "")

	// Remove anything in parentheses or brackets to get clean names (e.g. "(Official Video)")
	var result strings.Builder
	inParen := 0
	inBracket := 0
	for _, r := range title {
		if r == '(' {
			inParen++
		} else if r == '[' {
			inBracket++
		} else if r == ')' {
			if inParen > 0 {
				inParen--
			}
		} else if r == ']' {
			if inBracket > 0 {
				inBracket--
			}
		} else if inParen == 0 && inBracket == 0 {
			result.WriteRune(r)
		}
	}
	cleaned := result.String()

	// Replace specific words
	suffixes := []string{
		"official audio", "official video", "official music video",
		"lyric video", "lyrics video", "remastered", "remaster", "hq",
	}
	for _, s := range suffixes {
		cleaned = strings.ReplaceAll(cleaned, s, "")
	}

	// Remove artist prefix if it exists in title like "Artist - Title"
	cleanedArtist := cleanArtist(artist)
	if strings.Contains(cleaned, " - ") {
		parts := strings.SplitN(cleaned, " - ", 2)
		leftClean := strings.TrimSpace(parts[0])
		if leftClean == cleanedArtist {
			cleaned = parts[1]
		}
	}

	return strings.TrimSpace(cleaned)
}

func (e *DefaultEngine) doLrcLibRequest(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoKtave/1.0 (https://github.com/Kush-Singh-26/goktave)")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("lyrics not found on lrclib")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lrclib status error: %d", resp.StatusCode)
	}

	var r LrcLibResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	lyrics, err := lyricsFromLrcLibResponse(r)
	if err == nil {
		if r.SyncedLyrics != "" {
			logger.L.Debug("LrcLib found synchronized lyrics")
		} else {
			logger.L.Debug("LrcLib found plain lyrics")
		}
	}
	return lyrics, err
}

func lyricsFromLrcLibResponse(r LrcLibResponse) (string, error) {
	if r.Instrumental {
		return "", fmt.Errorf("instrumental track")
	}
	if r.SyncedLyrics != "" {
		return r.SyncedLyrics, nil
	}
	if r.PlainLyrics != "" {
		return r.PlainLyrics, nil
	}
	return "", fmt.Errorf("no lyrics content on lrclib")
}

func lrcLibGetURL(artist, title string, durationSec int) string {
	u := fmt.Sprintf("%s?artist_name=%s&track_name=%s",
		lrcLibAPIBase,
		url.QueryEscape(artist),
		url.QueryEscape(title),
	)
	if durationSec > 0 {
		u += fmt.Sprintf("&duration=%d", durationSec)
	}
	return u
}

func (e *DefaultEngine) cacheLyricsFile(videoID, lyrics string) {
	if lyrics == "" || videoID == "" {
		return
	}
	path := filepath.Join(e.cfg.AudioCacheDir, videoID+".lrc")
	if err := os.WriteFile(path, []byte(lyrics), 0644); err != nil {
		logger.L.Error("Failed to write lyrics cache file", "err", err, "path", path)
	}
}

func (e *DefaultEngine) fetchLrcLibLyrics(ctx context.Context, title, artist string, durationSec int) (string, error) {
	cleanedTitle := cleanTitle(title, artist)
	cleanedArtist := cleanArtist(artist)

	logger.L.Debug("Querying LrcLib", "cleanedTitle", cleanedTitle, "cleanedArtist", cleanedArtist, "duration", durationSec)

	// 1. Try with duration first
	var lyrics string
	var err error
	if durationSec > 0 {
		lyrics, err = e.doLrcLibRequest(ctx, lrcLibGetURL(cleanedArtist, cleanedTitle, durationSec))
	}

	// 2. If duration fetch failed or returned empty, retry without duration
	if err != nil || lyrics == "" {
		logger.L.Debug("LrcLib duration query failed/empty, retrying without duration", "err", err)
		lyrics, err = e.doLrcLibRequest(ctx, lrcLibGetURL(cleanedArtist, cleanedTitle, 0))
	}

	if err != nil {
		return "", err
	}
	return lyrics, nil
}
