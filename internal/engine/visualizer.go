package engine

func (e *DefaultEngine) GetVisualizerBars(n int) []float64 {
	return e.player.GetVisualizerBars(n)
}
