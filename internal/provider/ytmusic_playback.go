package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (p *YTMusicProvider) GetUpNext(videoID string) ([]Track, string, error) {
	url := "https://music.youtube.com/youtubei/v1/next?key=" + p.apiKey

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240401.01.00",
				"hl":            "en",
				"gl":            "US",
			},
		},
		"videoId":    videoID,
		"playlistId": "RDAMVM" + videoID,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://music.youtube.com")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, "", ErrRateLimited
	}

	var root map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, "", err
	}

	// Suggestions/Queue items
	// Path 1: suggestions (some versions)
	results := dig(root, "contents", "singleColumnMusicWatchNextResultsRenderer", "playlist", "playlist", "suggestions")
	if results == nil {
		// Path 2: tabs[0] -> musicQueueRenderer (newer versions)
		results = dig(root, "contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs", "0", "tabRenderer", "content", "musicQueueRenderer", "content", "playlistPanelRenderer", "contents")
	}

	suggestions, ok := results.([]interface{})
	tracks := make([]Track, 0, len(suggestions))
	if ok {
		for _, s := range suggestions {
			renderer := dig(s, "musicResponsiveListItemRenderer")
			if renderer == nil {
				renderer = dig(s, "playlistPanelVideoRenderer")
			}
			if renderer == nil {
				continue
			}

			vID := digStr(renderer, "videoId")
			if vID == "" {
				vID = digStr(renderer, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
			}
			if vID == "" {
				continue
			}

			title := digStr(renderer, "title")
			artist := ""
			duration := ""

			// Try to find artist and duration from longBylineText or similar
			byline := digStr(renderer, "longBylineText")
			if byline == "" {
				byline = digStr(renderer, "shortBylineText")
			}

			parts := strings.Split(byline, " • ")
			if len(parts) > 0 {
				artist = parts[0]
			}

			durText := digStr(renderer, "lengthText")
			if durText != "" {
				duration = durText
			}

			tracks = append(tracks, Track{
				VideoID:  vID,
				Title:    title,
				Artist:   artist,
				Duration: parseDuration(duration),
				ThumbURL: getThumb(renderer),
			})
		}
	}

	// Lyrics browseId
	lyricsBrowseID := ""
	tabs := dig(root, "contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs")
	if tabList, ok := tabs.([]interface{}); ok {
		for _, t := range tabList {
			tab := dig(t, "tabRenderer")
			title := digStr(tab, "title")
			if strings.EqualFold(title, "Lyrics") {
				lyricsBrowseID = digStr(tab, "endpoint", "browseEndpoint", "browseId")
				break
			}
		}
	}

	return tracks, lyricsBrowseID, nil
}

func (p *YTMusicProvider) GetLyrics(ctx context.Context, browseID string) (string, error) {
	if browseID == "" {
		return "No lyrics available for this track.", nil
	}

	url := "https://music.youtube.com/youtubei/v1/browse?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE"

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240401.01.00",
				"hl":            "en",
				"gl":            "US",
			},
		},
		"browseId": browseID,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var root map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return "", err
	}

	lyricsText := ""
	runs := dig(root, "contents", "sectionListRenderer", "contents", "0", "musicDescriptionShelfRenderer", "description", "runs")
	if runList, ok := runs.([]interface{}); ok {
		for _, r := range runList {
			lyricsText += digStr(r, "text")
		}
	}

	if lyricsText == "" {
		return "Lyrics content is empty.", nil
	}

	return lyricsText, nil
}
