package engine

func (e *DefaultEngine) SetVolume(v float64) {
	e.player.SetVolume(v)
}

func (e *DefaultEngine) GetVolume() float64 {
	return e.player.GetVolume()
}
