package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/cmplx"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
	"github.com/madelynnblue/go-dsp/fft"
	oto "github.com/ebitengine/oto/v3"
)

type State int

const (
	StateStopped State = iota
	StateBuffering
	StatePlaying
	StatePaused
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "Stopped"
	case StateBuffering:
		return "Buffering"
	case StatePlaying:
		return "Playing"
	case StatePaused:
		return "Paused"
	default:
		return "Unknown"
	}
}

type AudioPlayer interface {
	Play(url string) error
	Stop()
	Pause()
	Resume()
	TogglePause() bool
	SetVolume(v float64)
	GetVolume() float64
	State() State
	Wait() error
	GetVisualizerBars(n int) []float64
}

type Speaker struct {
	ctx *oto.Context
}

func NewSpeaker(cfg *config.Config) (*Speaker, error) {
	otoCtx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   cfg.SampleRate,
		ChannelCount: cfg.ChannelCount,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   cfg.BufferSize,
	})
	if err != nil {
		return nil, fmt.Errorf("oto init: %w", err)
	}
	<-ready
	return &Speaker{ctx: otoCtx}, nil
}

type Player struct {
	mu      sync.Mutex
	speaker *Speaker
	cmd     *exec.Cmd
	pipe    io.ReadCloser
	otoPlay *oto.Player
	volume  float64
	state   State
	cancel  context.CancelFunc
	stderr  *bytes.Buffer
	waitErr error
	done    chan struct{}

	visualizerBars []float64
	prevBars       []float64 // For decay
	sampleBuf      []float64
	bufPtr         int
}

func New(s *Speaker) *Player {
	return &Player{
		speaker:   s,
		volume:    1.0,
		sampleBuf: make([]float64, 1024),
	}
}

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
		
		if val > 1.0 { val = 1.0 }
		if val < 0 { val = 0 }
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
	p.prevBars = make([]float64, internalBars)
	copy(p.prevBars, bars)
	p.mu.Unlock()
}

func (p *Player) Play(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.stderr = new(bytes.Buffer)
	p.done = make(chan struct{})

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}

	isRemote := strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
	if isRemote {
		// Use more universally compatible reconnect flags
		args = append(args,
			"-reconnect", "1",
			"-reconnect_streamed", "1",
			"-reconnect_at_eof", "1",
			"-reconnect_delay_max", "5",
		)
	}

	args = append(args,
		"-probesize", "65536",
		"-analyzeduration", "100000",
		"-i", url,
		"-vn",
		"-f", "s16le",
		"-ar", "44100",
		"-ac", "2",
		"pipe:1",
	)

	p.cmd = exec.CommandContext(ctx, "ffmpeg", args...)
	p.cmd.Stderr = p.stderr
	p.state = StateBuffering

	pr, pw := io.Pipe()
	p.cmd.Stdout = pw
	p.pipe = pr

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	go func() {
		_ = p.cmd.Wait()
		pw.Close()
	}()

	const primeBytes = 32768
	primeBuf := make([]byte, primeBytes)
	if _, err := io.ReadFull(p.pipe, primeBuf); err != nil {
		p.stopLocked()
		return fmt.Errorf("failed to prime audio buffer: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	primedReader := io.MultiReader(bytes.NewReader(primeBuf), p.pipe)
	analyzedReader := &analyzerReader{src: primedReader, p: p}

	p.otoPlay = p.speaker.ctx.NewPlayer(analyzedReader)
	p.otoPlay.SetVolume(p.volume)
	p.otoPlay.Play()
	p.state = StatePlaying

	go func(done chan struct{}) {
		for {
			p.mu.Lock()
			if p.otoPlay == nil {
				p.mu.Unlock()
				break
			}
			// If not playing and buffer is empty, it might be done.
			// BUT only if we are NOT in StatePaused.
			if p.state != StatePaused && !p.otoPlay.IsPlaying() && p.otoPlay.BufferedSize() == 0 {
				p.mu.Unlock()
				break
			}
			p.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}(p.done)

	return nil
}

func (p *Player) Wait() error {
	p.mu.Lock()
	d := p.done
	p.mu.Unlock()

	if d == nil {
		return nil
	}
	<-d

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

func (p *Player) stopLocked() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if p.otoPlay != nil {
		p.otoPlay.Close()
		p.otoPlay = nil
	}
	p.state = StateStopped
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && p.state == StatePlaying {
		p.otoPlay.Pause()
		p.state = StatePaused
	}
}

func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && p.state == StatePaused {
		p.otoPlay.Play()
		p.state = StatePlaying
	}
}

func (p *Player) TogglePause() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay == nil {
		return false
	}
	if p.state == StatePlaying {
		p.otoPlay.Pause()
		p.state = StatePaused
		return true
	} else if p.state == StatePaused {
		p.otoPlay.Play()
		p.state = StatePlaying
		return false
	}
	return false
}

func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if v < 0 {
		v = 0
	}
	if v > 2.0 {
		v = 2.0
	}
	p.volume = v
	if p.otoPlay != nil {
		p.otoPlay.SetVolume(v)
	}
}

func (p *Player) GetVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}
