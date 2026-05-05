package extractor

// StreamInfo holds the raw audio URL extracted by yt-dlp.
type StreamInfo struct {
	URL string
}

// Extractor defines the contract for resolving playable audio streams.
type Extractor interface {
	Extract(videoID string) (*StreamInfo, error)
}