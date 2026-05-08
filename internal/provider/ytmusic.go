package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// dig safely traverses a nested map[string]interface{} or []interface{} using a list of keys/indices.
func dig(v interface{}, keys ...string) interface{} {
	for _, k := range keys {
		switch node := v.(type) {
		case map[string]interface{}:
			val, ok := node[k]
			if !ok {
				return nil
			}
			v = val
		case []interface{}:
			idx, err := strconv.Atoi(k)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil
			}
			v = node[idx]
		default:
			return nil
		}
	}
	return v
}

// digStr is a wrapper around dig that ensures the final result is a string.
// It also handles YT Music's "runs" objects automatically.
func digStr(v interface{}, keys ...string) string {
	result := dig(v, keys...)
	if result == nil {
		return ""
	}
	// If it's a map, maybe it's a {"runs": [...]} or {"simpleText": "..."}
	if m, ok := result.(map[string]interface{}); ok {
		if simple, ok := m["simpleText"].(string); ok {
			return simple
		}
		if runs, ok := m["runs"].([]interface{}); ok && len(runs) > 0 {
			fullText := ""
			for _, r := range runs {
				if text, ok := dig(r, "text").(string); ok {
					fullText += text
				}
			}
			return fullText
		}
	}
	s, ok := result.(string)
	if !ok {
		return ""
	}
	return s
}

// parseDuration converts a string like "3:45" or "1:05:20" into total seconds.
func parseDuration(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	total := 0
	for i, part := range parts {
		val, _ := strconv.Atoi(part)
		multiplier := 1
		// reverse calculation: parts[len-1] is seconds, parts[len-2] is minutes, etc.
		power := len(parts) - 1 - i
		for p := 0; p < power; p++ {
			multiplier *= 60
		}
		total += val * multiplier
	}
	return total
}

type YTMusicProvider struct {
	client *http.Client
}

func NewYTMusicProvider() *YTMusicProvider {
	return &YTMusicProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func getThumb(v interface{}) string {
	paths := [][]string{
		{"thumbnail", "musicThumbnailRenderer", "thumbnail", "thumbnails"},
		{"thumbnail", "thumbnails"},
		{"thumbnails"},
	}
	for _, p := range paths {
		if thumbList, ok := dig(v, p...).([]interface{}); ok && len(thumbList) > 0 {
			url := digStr(thumbList[len(thumbList)-1], "url")
			if url != "" {
				if strings.HasPrefix(url, "//") {
					return "https:" + url
				}
				return url
			}
		}
	}
	return ""
}

func (p *YTMusicProvider) Search(ctx context.Context, query string) ([]Track, error) {
	url := "https://music.youtube.com/youtubei/v1/search?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE"

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240401.01.00",
				"hl":            "en",
				"gl":            "US",
			},
		},
		"query":  query,
		"params": "EgWKAQIIAWoKEAkQBRAKEAMQBA==", // Filter for songs only
	}

	bodyBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://music.youtube.com")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, ErrRateLimited
	}

	var root map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	// Navigate the massive YT JSON tree
	contents := dig(root, "contents", "tabbedSearchResultsRenderer", "tabs", "0", "tabRenderer", "content", "sectionListRenderer", "contents")
	contentList, ok := contents.([]interface{})
	if !ok || len(contentList) == 0 {
		return nil, nil
	}

	// Find the section that contains musicResponsiveListItemRenderer
	var items []interface{}
	for _, section := range contentList {
		if results := dig(section, "musicShelfRenderer", "contents"); results != nil {
			if rList, ok := results.([]interface{}); ok {
				items = rList
				break
			}
		}
	}

	tracks := make([]Track, 0, len(items))
	for _, item := range items {
		renderer := dig(item, "musicResponsiveListItemRenderer")
		if renderer == nil {
			continue
		}

		videoID := digStr(renderer, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
		if videoID == "" {
			// Try alternative path for videoId
			videoID = digStr(renderer, "navigationEndpoint", "watchEndpoint", "videoId")
		}
		if videoID == "" {
			continue
		}

		flexColumns := dig(renderer, "flexColumns")
		cols, ok := flexColumns.([]interface{})
		if !ok || len(cols) < 2 {
			continue
		}

		title := digStr(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text")
		
		artist := ""
		duration := ""
		subtitle := digStr(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text")
		
		parts := strings.Split(subtitle, " • ")
		if len(parts) > 0 {
			artist = parts[0]
		}
		if len(parts) > 1 {
			lastPart := strings.TrimSpace(parts[len(parts)-1])
			if strings.Contains(lastPart, ":") {
				duration = lastPart
			}
		}

		tracks = append(tracks, Track{
			VideoID:  videoID,
			Title:    title,
			Artist:   artist,
			Duration: parseDuration(duration),
			ThumbURL: getThumb(renderer),
		})
	}

	return tracks, nil
}

func (p *YTMusicProvider) GetSuggestions(ctx context.Context, input string) ([]string, error) {
	url := "https://music.youtube.com/youtubei/v1/music/get_search_suggestions?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE"

	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20240401.01.00",
				"hl":            "en",
				"gl":            "US",
			},
		},
		"input": input,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var root map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, err
	}

	contents := dig(root, "contents", "0", "searchSuggestionsSectionRenderer", "contents")
	suggestionsList, ok := contents.([]interface{})
	if !ok {
		return nil, nil
	}

	suggestions := make([]string, 0, len(suggestionsList))
	for _, item := range suggestionsList {
		suggestion := digStr(item, "searchSuggestionRenderer", "suggestion")
		if suggestion != "" {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

func (p *YTMusicProvider) GetUpNext(videoID string) ([]Track, string, error) {
	url := "https://music.youtube.com/youtubei/v1/next?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE"

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
