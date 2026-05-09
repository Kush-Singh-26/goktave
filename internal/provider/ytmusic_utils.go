package provider

import (
	"strconv"
	"strings"
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
