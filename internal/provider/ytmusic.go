package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
