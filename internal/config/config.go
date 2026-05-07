package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	// Paths
	ConfigDir string
	QueuePath string

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
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "goktave")
	_ = os.MkdirAll(configDir, 0755)

	return &Config{
		ConfigDir:    configDir,
		QueuePath:    filepath.Join(configDir, "queue.json"),
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
