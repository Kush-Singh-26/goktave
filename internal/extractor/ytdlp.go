package extractor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
)

// YtDlpExtractor implements the Extractor interface using the yt-dlp binary.
type YtDlpExtractor struct {
	cfg *config.Config
}

// New creates a new YtDlpExtractor. 
func New(cfg *config.Config) *YtDlpExtractor {
	return &YtDlpExtractor{
		cfg: cfg,
	}
}

func (e *YtDlpExtractor) Extract(ctx context.Context, videoID string) (*StreamInfo, error) {
	// Give yt-dlp a strict 15-second deadline to find the URL, but also respect the parent context
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.cfg.YtDlpPath,
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"--format", "bestaudio[ext=webm]/bestaudio[ext=m4a]/bestaudio/best",
		"--print", "%(url)s",
		"https://youtube.com/watch?v="+videoID,
	)

	// Capture stderr to surface helpful errors (like "Video unavailable")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errors.New("yt-dlp timeout: took longer than 15 seconds")
		}
		
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("yt-dlp failed: %s", msg)
	}

	url := strings.TrimSpace(string(out))
	if url == "" || !strings.HasPrefix(url, "http") {
		return nil, errors.New("yt-dlp returned an empty or invalid URL")
	}

	return &StreamInfo{
		URL: url,
	}, nil
}