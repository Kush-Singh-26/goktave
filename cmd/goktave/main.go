package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/Kush-Singh-26/goktave/internal/config"
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
	ext := extractor.New(cfg)

	// Initialize the Engine
	eng := engine.New(cfg, prov, ext, pl)

	// Initialize the UI Model
	m := ui.NewModel(eng)

	// Launch the Bubble Tea program using the Alternate Screen buffer
	p := tea.NewProgram(m)

	manager, err := mpris.Start(
		func() {
			eng.TogglePause()
		},
		func() {
			eng.Next()
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
