package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	oto "github.com/ebitengine/oto/v3"
)

const videoID = "ugm2SOScEqA"

// resolveStreamURL fetches a fresh YouTube stream URL via yt-dlp.
// The call is bounded by the provided context (use a timeout context).
func resolveStreamURL(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"--format", "bestaudio[ext=webm]/bestaudio[ext=m4a]/bestaudio/best",
		"--print", "%(url)s",
		"https://youtube.com/watch?v="+videoID,
	)

	// Capture stderr so we can surface yt-dlp error messages.
	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("yt-dlp: %s", msg)
	}

	url := strings.TrimSpace(string(out))
	if url == "" {
		return "", errors.New("yt-dlp returned an empty URL")
	}
	return url, nil
}

// decoder holds a running ffmpeg process and the pipe to its stdout.
type decoder struct {
	cmd    *exec.Cmd
	pipe   io.ReadCloser
	stderr *strings.Builder
}

// startDecoder spawns ffmpeg, wiring stdout to a pipe that emits raw s16le PCM.
// The context is used only for startup — ffmpeg runs until explicitly stopped.
func startDecoder(url string) (*decoder, error) {
	cmd := exec.Command("ffmpeg",
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

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}

	// ← CHANGE: buffer stderr instead of streaming to os.Stderr directly
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ffmpeg start: %w", err)
	}

	return &decoder{cmd: cmd, pipe: pipe, stderr: &stderrBuf}, nil
}

// stop shuts down the decoder in the correct order:
//  1. Kill the ffmpeg process so it stops writing.
//  2. Close the pipe so any blocked reader unblocks.
//  3. Wait for the process to fully exit.
func (d *decoder) stop() {
	if d.cmd.Process != nil {
		_ = d.cmd.Process.Kill()
	}
	_ = d.pipe.Close()
	_ = d.cmd.Wait()
}

// session bundles an active decoder and oto player so both are closed together.
type session struct {
	dec    *decoder
	player *oto.Player
}

// close tears everything down in the safe order:
// player first (stops consuming the pipe), then the decoder.
func (s *session) close() {
	if s.player != nil {
		_ = s.player.Close()
		s.player = nil
	}
	if s.dec != nil {
		s.dec.stop()
		s.dec = nil
	}
}

// backoff returns a capped exponential wait duration.
// attempts starts at 0; first retry waits 1 s, doubling up to maxWait.
func backoff(attempts int, maxWait time.Duration) time.Duration {
	d := time.Duration(1<<uint(attempts)) * time.Second
	if d > maxWait {
		d = maxWait
	}
	return d
}

func main() {
	// Root context — cancelled on Ctrl+C / SIGTERM.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Single goroutine converts OS signals into context cancellation.
	// Buffered so the signal runtime never blocks.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			fmt.Println("\nshutting down cleanly...")
			cancel()
		case <-ctx.Done():
		}
	}()

	// oto context is created once for the lifetime of the process.
	// SampleRate/ChannelCount/Format MUST match the ffmpeg -ar/-ac/-f flags.
	otoCtx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   4096,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "oto init: %v\n", err)
		os.Exit(1)
	}
	<-ready

	fmt.Println("Playing... Ctrl+C to stop")

	attempts := 0
	const maxBackoff = 30 * time.Second

	for {
		// Respect cancellation at the top of every iteration.
		if ctx.Err() != nil {
			return
		}

		// --- Resolve stream URL (bounded by a 15 s timeout) ---
		resolveCtx, resolveCancel := context.WithTimeout(ctx, 15*time.Second)
		url, err := resolveStreamURL(resolveCtx)
		resolveCancel()

		if err != nil {
			if ctx.Err() != nil {
				return // cancelled while resolving — clean exit
			}
			wait := backoff(attempts, maxBackoff)
			fmt.Fprintf(os.Stderr, "resolve error: %v — retrying in %v\n", err, wait)
			attempts++
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}
			continue
		}

		fmt.Println("stream loaded")

		// --- Start decoder ---
		dec, err := startDecoder(url)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			wait := backoff(attempts, maxBackoff)
			fmt.Fprintf(os.Stderr, "decoder error: %v — retrying in %v\n", err, wait)
			attempts++
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}
			continue
		}

		// Reset backoff counter on a successful start.
		attempts = 0

		// Prime the pipe before creating the player so oto's buffer fills
		// immediately with real audio, not silence.
		const primeBytes = 4096
		primeBuf := make([]byte, primeBytes)
		if _, err := io.ReadFull(dec.pipe, primeBuf); err != nil {
			dec.stop()
			if ctx.Err() != nil {
				return
			}
			wait := backoff(attempts, maxBackoff)
			fmt.Fprintf(os.Stderr, "prime error: %v — retrying in %v\n", err, wait)
			attempts++
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}
			continue
		}

		primedReader := io.MultiReader(bytes.NewReader(primeBuf), dec.pipe)
		sess := &session{
			dec:    dec,
			player: otoCtx.NewPlayer(primedReader), // single player, primed data ready
		}
		sess.player.Play()

		// Wait for ffmpeg to exit or for the root context to be cancelled.
		done := make(chan error, 1)
		go func() { done <- dec.cmd.Wait() }()

		select {
		case <-ctx.Done():
			// Ctrl+C: close session and exit.
			sess.close()
			return

		case ffmpegErr := <-done:
			if ffmpegErr != nil {
				msg := strings.TrimSpace(sess.dec.stderr.String())
				if msg != "" {
					fmt.Fprintf(os.Stderr, "ffmpeg: %s\n", msg)
				}
				fmt.Fprintln(os.Stderr, "stream ended, restarting...")
			} else {
				fmt.Println("stream ended, restarting...")
			}
			sess.close()
		}

		// Brief pause before restarting so we don't hammer the network.
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}
