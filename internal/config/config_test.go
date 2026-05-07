package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.SampleRate != 44100 {
		t.Errorf("expected 44100, got %d", cfg.SampleRate)
	}
	if cfg.BufferSize != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", cfg.BufferSize)
	}
	if cfg.YtDlpPath != "yt-dlp" {
		t.Errorf("expected yt-dlp, got %s", cfg.YtDlpPath)
	}
}
