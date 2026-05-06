package extractor

import "context"

// StreamInfo holds the raw audio URL extracted by yt-dlp.
type StreamInfo struct {
	URL string
}

// Extractor defines the contract for resolving playable audio streams.
type Extractor interface {
	Extract(ctx context.Context, videoID string) (*StreamInfo, error)
}
