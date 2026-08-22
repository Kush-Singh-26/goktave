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
	PlayWithOffset(url string, offset time.Duration) error
	Stop()
	Pause()
	Resume()
	TogglePause() bool
	SetVolume(v float64)
	GetVolume() float64
	State() State
	Position() time.Duration
	LastPosition() time.Duration
	Wait() error
	GetVisualizerBars(n int) []float64
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
	cmdDone chan struct{}

	lastPos time.Duration

	posStart   time.Time
	posElapsed time.Duration

	visualizerBars []float64
	prevBars       []float64
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

func (p *Player) Play(url string) error {
	return p.PlayWithOffset(url, 0)
}

func (p *Player) PlayWithOffset(url string, offset time.Duration) error {
	p.mu.Lock()
	p.stopLocked()

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.stderr = new(bytes.Buffer)
	p.done = make(chan struct{})
	p.cmdDone = make(chan struct{})
	p.waitErr = nil

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

	if offset > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", offset.Seconds()))
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
		p.mu.Unlock()
		return fmt.Errorf("ffmpeg start: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}

	cmd := p.cmd
	cmdDone := p.cmdDone
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		if p.cmd == cmd {
			p.waitErr = err
		}
		p.mu.Unlock()
		pw.Close()
		close(cmdDone)
	}()

	const primeBytes = 32768
	primeBuf := make([]byte, primeBytes)
	p.mu.Unlock()
	if _, err := io.ReadFull(p.pipe, primeBuf); err != nil {
		p.mu.Lock()
		p.stopLocked()
		p.mu.Unlock()
		return fmt.Errorf("failed to prime audio buffer: %w (stderr: %s)", err, strings.TrimSpace(p.stderr.String()))
	}
	p.mu.Lock()

	primedReader := io.MultiReader(bytes.NewReader(primeBuf), p.pipe)
	analyzedReader := &analyzerReader{src: primedReader, p: p}

	p.otoPlay = p.speaker.ctx.NewPlayer(analyzedReader)
	p.otoPlay.SetVolume(p.volume)
	p.otoPlay.Play()
	p.state = StatePlaying
	p.posElapsed = offset
	p.posStart = time.Now()

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

	p.mu.Unlock()
	return nil
}

// Wait blocks until playback of the current stream has finished (either
// naturally or because it failed) and returns the underlying error, if any.
func (p *Player) Wait() error {
	p.mu.Lock()
	d := p.done
	cd := p.cmdDone
	p.mu.Unlock()

	if d == nil {
		return nil
	}
	<-d
	if cd != nil {
		<-cd
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitErr
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
	if p.pipe != nil {
		p.pipe.Close()
		p.pipe = nil
	}
	if p.otoPlay != nil {
		p.otoPlay.Close()
		p.otoPlay = nil
	}
	switch p.state {
	case StatePlaying:
		p.lastPos = p.posElapsed + time.Since(p.posStart)
	case StatePaused:
		p.lastPos = p.posElapsed
	}
	p.state = StateStopped
	p.posElapsed = 0
}

// LastPosition returns the playback position the player had when it was
// last stopped. Used to resume after an explicit stop.
func (p *Player) LastPosition() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastPos
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && p.state == StatePlaying {
		p.otoPlay.Pause()
		p.state = StatePaused
		p.posElapsed += time.Since(p.posStart)
	}
}

func (p *Player) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.otoPlay != nil && p.state == StatePaused {
		p.otoPlay.Play()
		p.state = StatePlaying
		p.posStart = time.Now()
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
		p.posElapsed += time.Since(p.posStart)
		return true
	} else if p.state == StatePaused {
		p.otoPlay.Play()
		p.state = StatePlaying
		p.posStart = time.Now()
		return false
	}
	return false
}

func (p *Player) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == StatePlaying {
		return p.posElapsed + time.Since(p.posStart)
	}
	return p.posElapsed
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
