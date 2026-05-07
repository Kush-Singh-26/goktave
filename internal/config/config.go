package config

import (
	"os"
	"time"
)

type Config struct {
	// Player settings
	SampleRate   int
	ChannelCount int
	BufferSize   time.Duration

	// Extractor settings
	YtDlpPath string

	// Provider settings
	YTMusicKey string
}

func Default() *Config {
	return &Config{
		SampleRate:   44100,
		ChannelCount: 2,
		BufferSize:   100 * time.Millisecond,
		YtDlpPath:    getEnv("YTDLP_PATH", "yt-dlp"),
		YTMusicKey:   "AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE",
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
