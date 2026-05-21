package engine

import (
	"time"

	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

func (e *DefaultEngine) GetCurrentTrack() *provider.Track {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.currentTrack
}

func (e *DefaultEngine) GetState() player.State {
	return e.player.State()
}

func (e *DefaultEngine) UpdateMPRISPosition() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.mpris != nil {
		e.mpris.UpdatePosition(e.player.Position())
	}
}

func (e *DefaultEngine) GetPlayPosition() time.Duration {
	return e.player.Position()
}
