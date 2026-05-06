package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
	"github.com/Kush-Singh-26/goktave/internal/ui"
	"github.com/Kush-Singh-26/goktave/internal/mpris"
)

func main() {
	// Initialize the audio system
	speaker, err := player.NewSpeaker()
	if err != nil {
		fmt.Printf("Fatal: Could not initialize audio: %v\n", err)
		os.Exit(1)
	}
	pl := player.New(speaker)
	defer pl.Stop()

	// Initialize backend services
	prov := provider.NewYTMusic()
	ext := extractor.New()

	// Initialize the UI Model
	m := ui.NewModel(prov, ext, pl)

	// Launch the Bubble Tea program using the Alternate Screen buffer
	p := tea.NewProgram(m)

	go mpris.Start(
			func() {
				// When Linux says Play/Pause, send a Spacebar press to Bubble Tea!
				p.Send(tea.KeyPressMsg{Code: tea.KeySpace})
			},
			func() {
				// When Linux says Next, send an 'n' press to Bubble Tea!
				p.Send(tea.KeyPressMsg{Code: 'n'})
			},
		)
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}