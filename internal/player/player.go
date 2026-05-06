package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	oto "github.com/ebitengine/oto/v3"
)

// Speaker manages the underlying OS audio device
type Speaker struct {
	ctx *oto.Context
}

// NewSpeaker initializes the oto audio context once for the app.
func NewSpeaker() (*Speaker, error) {
	otoCtx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   4096,
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
	cancel  context.CancelFunc
	stderr  *bytes.Buffer
	waitErr error
	done    chan struct{}
}

// New creates a new player instance attached to the speaker
func New(s *Speaker) *Player {
	return &Player{speaker: s}
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
		"-probesize", "65536",
		"-analyzeduration", "100000",
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-i", url,
		"-vn",
		"-f", "s16le",
		"-ar", "44100",
		"-ac", "2",
		"-flush_packets", "1",
		"pipe:1",
	)
	p.cmd.Stderr = p.stderr

	var err error
	p.pipe, err = p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	// Prime the buffer (from your excellent spike!)
	const primeBytes = 4096
	primeBuf := make([]byte, primeBytes)
	if _, err := io.ReadFull(p.pipe, primeBuf); err != nil {
		p.stopLocked()
		return fmt.Errorf("failed to prime audio buffer: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	primedReader := io.MultiReader(bytes.NewReader(primeBuf), p.pipe)

	p.otoPlay = p.speaker.ctx.NewPlayer(primedReader)
	p.otoPlay.Play()

	// Launch a background waiter to ensure Wait is only called once
	go func(cmd *exec.Cmd, done chan struct{}) {
		err := cmd.Wait()
		p.mu.Lock()
		p.waitErr = err
		close(done)
		p.mu.Unlock()
	}(p.cmd, p.done)

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

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitErr
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
	// The background goroutine launched in Play() will finish its Wait() 
	// and close the channel independently.
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && p.otoPlay.IsPlaying() {
		p.otoPlay.Pause()
	}
}

func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && !p.otoPlay.IsPlaying() {
		p.otoPlay.Play()
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
		return true
	}
	p.otoPlay.Play()
	return false
}
