package player

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

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
	speaker *Speaker
	cmd     *exec.Cmd
	pipe    io.ReadCloser
	otoPlay *oto.Player
}

// New creates a new player instance attached to the speaker
func New(s *Speaker) *Player {
	return &Player{speaker: s}
}

// Play takes a raw stream URL, starts ffmpeg, and pumps audio to the speaker
func (p *Player) Play(url string) error {
	// Stop any currently playing audio first
	p.Stop()

	p.cmd = exec.Command("ffmpeg",
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

	var err error
	p.pipe, err = p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	// Prime the buffer (from your excellent spike!)
	const primeBytes = 4096
	primeBuf := make([]byte, primeBytes)
	if _, err := io.ReadFull(p.pipe, primeBuf); err != nil {
		p.Stop()
		return fmt.Errorf("failed to prime audio buffer: %w", err)
	}

	primedReader := io.MultiReader(bytes.NewReader(primeBuf), p.pipe)
	
	p.otoPlay = p.speaker.ctx.NewPlayer(primedReader)
	p.otoPlay.Play()

	return nil
}

// Stop cleanly kills ffmpeg and the oto player
func (p *Player) Stop() {
	if p.otoPlay != nil {
		p.otoPlay.Close()
		p.otoPlay = nil
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	if p.pipe != nil {
		p.pipe.Close()
	}
	if p.cmd != nil {
		p.cmd.Wait()
		p.cmd = nil
	}
}