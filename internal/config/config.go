package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type Config struct {
	// Paths
	ConfigDir     string
	QueuePath     string
	DBPath        string
	AudioCacheDir string

	// Player settings
	SampleRate   int
	ChannelCount int
	BufferSize   time.Duration

	// Extractor settings
	YtDlpPath string

	// Provider settings
	YTMusicKey string

	// Cache settings
	MaxCacheSizeGB float64
	Theme          string
	AccentOverride string
	VizMode        int
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "goktave")
	_ = os.MkdirAll(configDir, 0755)

	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = filepath.Join(home, ".cache")
	}
	audioCacheDir := filepath.Join(cacheDir, "goktave", "audio")
	_ = os.MkdirAll(audioCacheDir, 0755)

	return &Config{
		ConfigDir:      configDir,
		QueuePath:      filepath.Join(configDir, "queue.json"),
		DBPath:         filepath.Join(configDir, "goktave.db"),
		AudioCacheDir:  audioCacheDir,
		SampleRate:     44100,
		ChannelCount:   2,
		BufferSize:     100 * time.Millisecond,
		YtDlpPath:      getEnv("YTDLP_PATH", "yt-dlp"),
		YTMusicKey:     provider.DefaultAPIKey,
		MaxCacheSizeGB: 1.0,
		Theme:          "Terracotta (Default)",
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
