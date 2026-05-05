package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// dig safely traverses a nested map[string]interface{} using a list of keys.
// If a key is missing or the type is wrong, it returns nil instead of panicking.
func dig(v interface{}, keys ...string) interface{} {
	for _, k := range keys {
		switch node := v.(type) {
		case map[string]interface{}:
			val, ok := node[k]
			if !ok {
				return nil
			}
			v = val
		default:
			return nil
		}
	}
	return v
}

// digStr is a wrapper around dig that ensures the final result is a string.
func digStr(v interface{}, keys ...string) string {
	result := dig(v, keys...)
	if result == nil {
		return ""
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
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

type YTMusicProvider struct {
	client *http.Client
}

func NewYTMusic() *YTMusicProvider {
	return &YTMusicProvider{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *YTMusicProvider) Search(query string) ([]Track, error) {
	url := "https://music.youtube.com/youtubei/v1/search?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE"

	// This is the exact JSON structure the internal API expects
	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "WEB_REMIX",
				"clientVersion": "1.20231204.01.00",
				"hl":            "en",
				"gl":            "US",
			},
		},
		"query": query,
		// This base64 params string filters the search results specifically to "Songs"
		"params": "EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D",
	}

	bodyBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))

	// These headers bypass the basic bot protections
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://music.youtube.com")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, ErrRateLimited
	}

	var root map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	return parseSearchResults(root)
}

func parseSearchResults(root map[string]interface{}) ([]Track, error) {
	// 1. Drill down to the main list of items
	contents := dig(root, "contents", "tabbedSearchResultsRenderer", "tabs")
	tabs, ok := contents.([]interface{})
	if !ok || len(tabs) == 0 {
		return nil, fmt.Errorf("could not find tabs in response")
	}

	// 2. Drill into the first tab (Songs)
	sectionList := dig(tabs[0], "tabRenderer", "content", "sectionListRenderer", "contents")
	sections, ok := sectionList.([]interface{})
	if !ok || len(sections) == 0 {
		return nil, fmt.Errorf("could not find sections")
	}

	// 3. Drill into the music shelf
	shelfContents := dig(sections[0], "musicShelfRenderer", "contents")
	items, ok := shelfContents.([]interface{})
	if !ok {
		return nil, fmt.Errorf("could not find items in shelf")
	}

	var tracks []Track

	// 4. Iterate over each song and extract metadata
	for _, item := range items {
		renderer := dig(item, "musicResponsiveListItemRenderer")
		if renderer == nil {
			continue
		}

		videoID := digStr(renderer, "overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId")
		if videoID == "" {
			continue // Skip if it's not a playable track
		}

		// Title and Artist are buried inside arrays of "flexColumns"
		flexColumns := dig(renderer, "flexColumns")
		cols, ok := flexColumns.([]interface{})
		if !ok || len(cols) < 2 {
			continue
		}

		titleData := dig(cols[0], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
		title := ""
		if titleRuns, ok := titleData.([]interface{}); ok && len(titleRuns) > 0 {
			title = digStr(titleRuns[0], "text")
		}

		// Artist, Album, and Duration are jammed together separated by " • "
		artistData := dig(cols[1], "musicResponsiveListItemFlexColumnRenderer", "text", "runs")
		artist := ""
		duration := ""

		if artistRuns, ok := artistData.([]interface{}); ok {
			fullSubtitle := ""
			for _, run := range artistRuns {
				fullSubtitle += digStr(run, "text")
			}

			// Split by the bullet character YT uses
			parts := strings.Split(fullSubtitle, " • ")
			if len(parts) > 0 {
				artist = parts[0] // First part is always the artist
			}
			if len(parts) > 1 {
				// The last part is usually the duration
				lastPart := strings.TrimSpace(parts[len(parts)-1])
				if strings.Contains(lastPart, ":") {
					duration = lastPart
				}
			}
		}

		// Fallback: Check fixedColumns just in case YT changes their mind
		fixedColumns := dig(renderer, "fixedColumns")
		if fCols, ok := fixedColumns.([]interface{}); ok && len(fCols) > 0 {
			if durRuns, ok := dig(fCols[0], "musicResponsiveListItemFixedColumnRenderer", "text", "runs").([]interface{}); ok && len(durRuns) > 0 {
				if d := digStr(durRuns[0], "text"); d != "" {
					duration = d
				}
			}
		}

		tracks = append(tracks, Track{
			VideoID:  videoID,
			Title:    title,
			Artist:   artist,
			Duration: parseDuration(duration),
		})
	}

	return tracks, nil
}
