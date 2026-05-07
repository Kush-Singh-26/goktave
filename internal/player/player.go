package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/config"
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

// AudioPlayer defines the contract for an audio playback device
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
}

// Speaker manages the underlying OS audio device
type Speaker struct {
	ctx *oto.Context
}

// NewSpeaker initializes the oto audio context once for the app.
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
	<-ready // wait for the audio device to be ready
	return &Speaker{ctx: otoCtx}, nil
}

// Player controls an active ffmpeg stream
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
}

// New creates a new player instance attached to the speaker
func New(s *Speaker) *Player {
	return &Player{
		speaker: s,
		volume: 1.0,
	}
}

// Play takes a raw stream URL, starts ffmpeg, and pumps audio to the speaker.
// It returns an error if startup fails. It does NOT wait for playback to finish.
func (p *Player) Play(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop any currently playing audio first (manual call to internal stop logic)
	p.stopLocked()

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.stderr = new(bytes.Buffer)
	p.done = make(chan struct{})

	p.cmd = exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-probesize", "65536",
		"-analyzeduration", "100000",
		"-i", url,
		"-vn",
		"-f", "s16le",
		"-ar", "44100",
		"-ac", "2",
		"pipe:1",
	)
	p.cmd.Stderr = p.stderr
	p.state = StateBuffering

	// Use an io.Pipe so ffmpeg exiting doesn't close the reader immediately,
	// allowing us to drain the last bit of audio.
	pr, pw := io.Pipe()
	p.cmd.Stdout = pw
	p.pipe = pr

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	// Launch a goroutine to close the pipe-writer when ffmpeg finishes
	go func() {
		_ = p.cmd.Wait()
		pw.Close()
	}()

	// Prime the buffer
	const primeBytes = 32768
	primeBuf := make([]byte, primeBytes)
	if _, err := io.ReadFull(p.pipe, primeBuf); err != nil {
		p.stopLocked()
		return fmt.Errorf("failed to prime audio buffer: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	primedReader := io.MultiReader(bytes.NewReader(primeBuf), p.pipe)

	p.otoPlay = p.speaker.ctx.NewPlayer(primedReader)
	p.otoPlay.SetVolume(p.volume)
	p.otoPlay.Play()
	p.state = StatePlaying

	go func(done chan struct{}) {
		// We wait for the player to naturally finish or be stopped.
		for {
			p.mu.Lock()
			if p.otoPlay == nil || (!p.otoPlay.IsPlaying() && p.otoPlay.BufferedSize() == 0) {
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

// Wait blocks until the current playback finishes or is stopped.
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

// Stop cleanly kills ffmpeg and the oto player
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

// stopLocked handles cleanup while holding the lock.
// It triggers cancellation but does NOT block waiting for the process to exit,
// ensuring methods like Play() remain atomic and thread-safe.
func (p *Player) stopLocked() {
	p.state = StateStopped
	if p.otoPlay != nil {
		p.otoPlay.Close()
		p.otoPlay = nil
	}
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if p.pipe != nil {
		p.pipe.Close()
		p.pipe = nil
	}
	// Note: We don't wait for p.done here to avoid releasing the mutex.
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && p.otoPlay.IsPlaying() {
		p.otoPlay.Pause()
		p.state = StatePaused
	}
}

func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && !p.otoPlay.IsPlaying() {
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
	if p.otoPlay.IsPlaying() {
		p.otoPlay.Pause()
		p.state = StatePaused
		return true
	}
	p.otoPlay.Play()
	p.state = StatePlaying
	return false
}

func (p *Player) SetVolume(v float64) {
	if v < 0.0 {
		v = 0.0
	}
	if v > 1.0 {
		v = 1.0
	}
	p.volume = v
	if p.otoPlay != nil {
		p.otoPlay.SetVolume(p.volume)
	}
}

func (p *Player) GetVolume() float64 {
	return p.volume
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}
