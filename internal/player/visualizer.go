package player

import (
	"io"
	"math"
	"math/cmplx"

	"github.com/madelynnblue/go-dsp/fft"
)

func (p *Player) GetVisualizerBars(n int) []float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.visualizerBars) == 0 {
		return make([]float64, n)
	}

	// Resample internal bars (usually 128) to requested n bars
	res := make([]float64, n)
	src := p.visualizerBars
	srcLen := len(src)

	for i := 0; i < n; i++ {
		// Linear interpolation or simple mapping
		srcIdx := float64(i) * float64(srcLen) / float64(n)
		idx := int(srcIdx)
		if idx >= srcLen {
			idx = srcLen - 1
		}
		res[i] = src[idx]
	}

	return res
}

type analyzerReader struct {
	src io.Reader
	p   *Player
}

func (r *analyzerReader) Read(p []byte) (n int, err error) {
	n, err = r.src.Read(p)
	if n > 0 {
		r.p.updateVisualizer(p[:n])
	}
	return n, err
}

func (p *Player) updateVisualizer(data []byte) {
	p.mu.Lock()

	for i := 0; i < len(data); i += 4 { // 2 channels, 16-bit
		if i+1 >= len(data) {
			break
		}
		// Convert s16le to float64 mono
		val := int16(data[i]) | int16(data[i+1])<<8
		p.sampleBuf[p.bufPtr] = float64(val) / 32768.0
		p.bufPtr++

		// Trigger FFT whenever buffer is full
		if p.bufPtr >= len(p.sampleBuf) {
			p.bufPtr = 0

			// Copy buffer for processing outside potential intensive loops
			bufCopy := make([]float64, len(p.sampleBuf))
			copy(bufCopy, p.sampleBuf)

			// Unlock briefly to allow other operations (volume etc) while we do FFT
			p.mu.Unlock()
			p.processFFT(bufCopy)
			p.mu.Lock()
		}
	}
	p.mu.Unlock()
}

func (p *Player) processFFT(samples []float64) {
	coeffs := fft.FFTReal(samples)
	const internalBars = 128
	bars := make([]float64, internalBars)

	half := len(coeffs) / 2
	// Use logarithmic binning for a more musical feel (more bins for low/mids)
	// But for simplicity and TUI clarity, we'll use slightly skewed linear binning
	for i := 0; i < internalBars; i++ {
		// Logarithmic mapping of FFT bins to bars
		start := float64(i) / internalBars
		end := float64(i+1) / internalBars

		// Square the indices to give more weight to lower frequencies
		startIdx := int(start * start * float64(half))
		endIdx := int(end * end * float64(half))

		if endIdx <= startIdx {
			endIdx = startIdx + 1
		}

		magnitude := 0.0
		count := 0
		for j := startIdx; j < endIdx && j < half; j++ {
			magnitude += cmplx.Abs(coeffs[j])
			count++
		}

		avg := 0.0
		if count > 0 {
			avg = magnitude / float64(count)
		}

		// Re-tuned Sensitivity:
		// Lower base multiplier (from 10.0 to 4.0) to avoid ceiling
		// Gentler frequency boost
		boost := 1.0 + (math.Pow(float64(i)/internalBars, 1.5) * 3.0)
		val := math.Log10(1+avg*5.0) * boost * 0.7

		if val > 1.0 {
			val = 1.0
		}
		if val < 0 {
			val = 0
		}
		bars[i] = val
	}

	p.mu.Lock()
	if len(p.prevBars) != internalBars {
		p.prevBars = make([]float64, internalBars)
	}

	// Apply Peak Decay (Gravity)
	// Bars rise instantly but fall smoothly
	decayFactor := 0.85
	for i := 0; i < internalBars; i++ {
		if bars[i] < p.prevBars[i]*decayFactor {
			bars[i] = p.prevBars[i] * decayFactor
		}
	}

	// Horizontal Smoothing (Blur) to avoid blockiness
	smoothed := make([]float64, internalBars)
	for i := 0; i < internalBars; i++ {
		val := bars[i]
		weight := 1.0
		if i > 0 {
			val += bars[i-1] * 0.5
			weight += 0.5
		}
		if i < internalBars-1 {
			val += bars[i+1] * 0.5
			weight += 0.5
		}
		smoothed[i] = val / weight
	}
	bars = smoothed

	for i := 0; i < internalBars; i++ {
		if bars[i] < 0.08 { // Noise floor
			bars[i] = 0
		}
	}

	p.visualizerBars = bars
	copy(p.prevBars, bars)
	p.mu.Unlock()
}
