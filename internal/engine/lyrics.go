package engine

func (e *DefaultEngine) GetLyrics() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.currentLyrics == "" {
		return "No lyrics available for this track."
	}
	return e.currentLyrics
}
