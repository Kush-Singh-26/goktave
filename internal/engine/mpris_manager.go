package engine

import "github.com/Kush-Singh-26/goktave/internal/mpris"

func (e *DefaultEngine) SetMPRIS(m *mpris.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mpris = m
}
