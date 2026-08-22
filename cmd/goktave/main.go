package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Kush-Singh-26/goktave/internal/config"
	"github.com/Kush-Singh-26/goktave/internal/db"
	"github.com/Kush-Singh-26/goktave/internal/engine"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/mpris"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
	"github.com/Kush-Singh-26/goktave/internal/ui"
)

func main() {
	// Initialize logger
	cleanup, err := logger.Init()
	if err != nil {
		fmt.Printf("Fatal: Could not initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	logger.L.Info("Goktave starting...")

	// Dependency check
	if err := checkDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}

	// Initialize config
	cfg := config.Default()

	// Initialize database
	database, err := db.New(cfg.DBPath)
	if err != nil {
		logger.L.Error("Could not initialize database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	// Load saved config if it exists
	var savedCfg config.Config
	if err := database.GetState("config", &savedCfg); err == nil {
		// Only override persistent fields
		cfg.MaxCacheSizeGB = savedCfg.MaxCacheSizeGB
		cfg.Theme = savedCfg.Theme
		cfg.AccentOverride = savedCfg.AccentOverride
		cfg.VizMode = savedCfg.VizMode
	}

	// Initialize the audio system
	speaker, err := player.NewSpeaker(cfg)
	if err != nil {
		logger.L.Error("Could not initialize audio", "err", err)
		os.Exit(1)
	}
	pl := player.New(speaker)
	defer pl.Stop()

	// Initialize backend services
	prov := provider.NewYTMusicProvider()
	ext := extractor.NewCachedExtractor(extractor.New(cfg), 4*time.Hour)

	// Initialize the Engine
	eng := engine.New(cfg, prov, ext, pl, database)

	// Initialize the UI Model
	m := ui.NewModel(eng)

	// Launch the Bubble Tea program using the Alternate Screen buffer
	p := tea.NewProgram(m)

	manager, err := mpris.Start(
		func() {
			eng.TogglePause()
		},
		func() {
			if eng.GetState() == player.StatePaused {
				eng.TogglePause()
			}
		},
		func() {
			if eng.GetState() == player.StatePlaying {
				eng.TogglePause()
			}
		},
		func() {
			eng.Next()
		},
		func() {
			eng.Prev()
		},
	)
	if err == nil {
		eng.SetMPRIS(manager)
	} else {
		logger.L.Warn("MPRIS start failed", "err", err)
	}

	if _, err := p.Run(); err != nil {
		logger.L.Error("Error running program", "err", err)
		os.Exit(1)
	}
}

func checkDependencies() error {
	deps := []string{"ffmpeg", "yt-dlp"}
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s not found in PATH. Please install it", dep)
		}
	}
	return nil
}
