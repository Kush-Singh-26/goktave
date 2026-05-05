# GoTune: Final Complete Engineering Plan

> Platform: Fedora Linux only.
> Audio backend: ffmpeg (already installed) + oto.
> Extraction: yt-dlp subprocess.
> Search: YouTube Music internal API.
> Media keys: MPRIS2 over D-Bus.
> Nothing skipped.

---

## Table of Contents

1. [Project Overview and Decisions](#1-project-overview-and-decisions)
2. [System Dependencies](#2-system-dependencies)
3. [Repository Structure](#3-repository-structure)
4. [Module Initialization and Go Dependencies](#4-module-initialization-and-go-dependencies)
5. [The Config Layer](#5-the-config-layer)
6. [The Provider Layer — YouTube Music Search](#6-the-provider-layer)
7. [The Extractor Layer — yt-dlp Subprocess](#7-the-extractor-layer)
8. [The Player Layer — ffmpeg + oto](#8-the-player-layer)
9. [The Cache Layer — bbolt + LRU](#9-the-cache-layer)
10. [The MPRIS2 Layer — Media Keys via D-Bus](#10-the-mpris2-layer)
11. [The TUI Layer — Bubbletea](#11-the-tui-layer)
12. [Concurrency Model](#12-concurrency-model)
13. [Error Handling](#13-error-handling)
14. [Signal Handling and Clean Shutdown](#14-signal-handling-and-clean-shutdown)
15. [Build System](#15-build-system)
16. [Testing Strategy](#16-testing-strategy)
17. [Phase-by-Phase Execution Plan](#17-phase-by-phase-execution-plan)
18. [Known Failure Modes and Mitigations](#18-known-failure-modes-and-mitigations)

---

## 1. Project Overview and Decisions

### What it is

A terminal-based music player that streams audio from YouTube Music. Single compiled binary. No Electron. No ads. No YouTube Premium required.

### Finalized decisions

| Concern | Decision | Reason |
|---|---|---|
| Language | Go 1.26.2 | Already chosen |
| OS | Fedora Linux only | No Windows/macOS conditionals anywhere |
| TUI | bubbletea + lipgloss + bubbles | Standard for Go TUIs |
| Audio decoding | ffmpeg subprocess → PCM pipe | Already installed, handles every codec |
| Audio output | oto v2 | Low-level PCM speaker, works on PulseAudio/PipeWire/ALSA |
| Stream extraction | yt-dlp subprocess | Most robust extractor, self-updating |
| YT Music search | Direct HTTP to internal API | No third-party library needed |
| Cache index | bbolt | Right weight, crash-safe, no CGO |
| Config | TOML via BurntSushi/toml | Simple, human-editable |
| Media keys | MPRIS2 over D-Bus via godbus | Linux standard, works with GNOME/KDE/etc |

### What this will not do

- No Windows or macOS support in this plan.
- No YouTube Premium features.
- No lyrics.
- No album art in the terminal (sixel rendering is out of scope for V1).
- No offline-only mode (always needs internet for first play of uncached tracks).

### Maintenance reality

yt-dlp handles YouTube's signature obfuscation and updates within days of YouTube breaking changes. The YouTube Music search API is undocumented and its JSON response shape can change without notice — this is the most fragile part and you must be prepared to fix the response parser when it breaks. Run `yt-dlp -U` periodically. There is no automated fix for this; it requires a human to inspect network traffic and update the parser.

---

## 2. System Dependencies

### Required (must be present at runtime)

**ffmpeg**: Used to decode the audio stream to raw PCM and pipe it to your Go process.

```bash
sudo dnf install ffmpeg
```

If `ffmpeg` is restricted on your Fedora version, enable RPM Fusion first:

```bash
sudo dnf install \
  https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm \
  https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm
sudo dnf install ffmpeg
```

**yt-dlp**: Used to extract direct stream URLs from YouTube video IDs.

```bash
pip install yt-dlp
# or grab the standalone binary:
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
  -o /usr/local/bin/yt-dlp
sudo chmod +x /usr/local/bin/yt-dlp
```

Verify both are in PATH:

```bash
ffmpeg -version
yt-dlp --version
```

### Required system libraries (for oto audio output)

oto on Linux outputs to PulseAudio, PipeWire (via PulseAudio compatibility layer), or ALSA. On a standard Fedora desktop, PipeWire with PulseAudio compatibility is active by default. No extra packages needed. If on a minimal Fedora install without a desktop:

```bash
sudo dnf install pulseaudio-libs-devel alsa-lib-devel
```

oto uses CGO to bind to the audio library. This means you need a C compiler:

```bash
sudo dnf install gcc
```

This is the only CGO dependency in the project. Everything else is pure Go.

### D-Bus (for MPRIS2 media keys)

D-Bus session bus is present on any Fedora desktop. No installation needed. The `godbus` library is pure Go and does not require CGO.

### Startup dependency check

At application startup, before initializing anything else, check for required binaries:

```go
func checkDependencies() error {
    required := []string{"ffmpeg", "yt-dlp"}
    for _, bin := range required {
        if _, err := exec.LookPath(bin); err != nil {
            return fmt.Errorf(
                "%s not found in PATH.\n"+
                "Install with: pip install %s\n"+
                "Or see: https://github.com/yt-dlp/yt-dlp#installation",
                bin, bin,
            )
        }
    }
    return nil
}
```

Print the error to stderr and exit with code 1 if any dependency is missing. Do not start the TUI.

---

## 3. Repository Structure

```
gotune/
├── cmd/
│   └── gotune/
│       └── main.go                  # Entrypoint
├── internal/
│   ├── config/
│   │   └── config.go                # TOML config load/save/defaults
│   ├── provider/
│   │   ├── provider.go              # Track type + Provider interface
│   │   ├── ytmusic.go               # YouTube Music HTTP implementation
│   │   └── ytmusic_test.go
│   ├── extractor/
│   │   ├── extractor.go             # StreamInfo type + Extractor interface
│   │   ├── ytdlp.go                 # yt-dlp subprocess wrapper
│   │   └── ytdlp_test.go
│   ├── player/
│   │   ├── player.go                # Player interface
│   │   ├── ffmpeg.go                # ffmpeg subprocess → PCM pipe
│   │   ├── output.go                # oto speaker management
│   │   └── player_test.go
│   ├── cache/
│   │   ├── cache.go                 # Cache interface + LRU eviction
│   │   ├── index.go                 # bbolt-backed metadata index
│   │   └── cache_test.go
│   ├── mpris/
│   │   ├── mpris.go                 # MPRIS2 D-Bus service registration
│   │   └── properties.go            # Property get/set + PropertiesChanged emit
│   └── ui/
│       ├── model.go                 # Root bubbletea model + messages
│       ├── search.go                # Search view
│       ├── queue.go                 # Queue view
│       ├── playerbar.go             # Bottom player bar
│       ├── cacheview.go             # Cache manager overlay
│       ├── toast.go                 # Temporary notification
│       ├── keys.go                  # All keybinding definitions
│       └── styles.go                # All lipgloss styles
├── testdata/
│   └── search_response.json         # Saved real API response for unit tests
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

All packages are under `internal/`. Nothing is exported as a public library. This is intentional — we are building an application, not a library, and `internal/` prevents accidental import by other projects.

---

## 4. Module Initialization and Go Dependencies

```bash
mkdir gotune && cd gotune
go mod init github.com/yourname/gotune
```

Add dependencies:

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/hajimehoshi/oto/v2@latest
go get go.etcd.io/bbolt@latest
go get github.com/BurntSushi/toml@latest
go get github.com/godbus/dbus/v5@latest
```

What each does:

- `bubbletea`: The Elm-Architecture TUI event loop. Model/Update/View.
- `lipgloss`: Terminal styling. Colors, borders, padding, alignment, layout composition.
- `bubbles`: Pre-built bubbletea components. We use `textinput` (search box) and `progress` (playback progress bar).
- `oto/v2`: Low-level PCM audio output for Linux (PulseAudio/PipeWire/ALSA). CGO required.
- `bbolt`: Embedded B-tree key-value store for the cache index. Pure Go, no daemon.
- `BurntSushi/toml`: TOML config file encoding/decoding.
- `godbus/dbus/v5`: D-Bus client for MPRIS2 media key integration. Pure Go.

There are no other dependencies. Do not add a YAML library, a logging framework, a CLI flag library beyond `flag` from stdlib, or any HTTP client beyond `net/http` from stdlib.

---

## 5. The Config Layer

### File: `internal/config/config.go`

Config file path: `~/.config/gotune/config.toml`

Created automatically on first run with defaults if it does not exist.

**The config struct:**

```go
package config

import (
    "os"
    "path/filepath"
    "github.com/BurntSushi/toml"
)

type Config struct {
    Cache      CacheConfig      `toml:"cache"`
    Playback   PlaybackConfig   `toml:"playback"`
    YtDlp      YtDlpConfig      `toml:"ytdlp"`
    FFmpeg     FFmpegConfig     `toml:"ffmpeg"`
    Appearance AppearanceConfig `toml:"appearance"`
}

type CacheConfig struct {
    MaxSizeMB int    `toml:"max_size_mb"`  // default: 2048
    Directory string `toml:"directory"`    // default: "" (use ~/.cache/gotune)
}

type PlaybackConfig struct {
    Volume        int  `toml:"volume"`          // default: 80, range 0-100
    BufferSizeKB  int  `toml:"buffer_size_kb"`  // default: 8, range 4-64
}

type YtDlpConfig struct {
    Path       string `toml:"path"`        // default: "yt-dlp"
    RateLimit  string `toml:"rate_limit"`  // default: "" (no limit), e.g. "500K"
}

type FFmpegConfig struct {
    Path       string `toml:"path"`        // default: "ffmpeg"
    Threads    int    `toml:"threads"`     // default: 1
    SampleRate int    `toml:"sample_rate"` // default: 44100
}

type AppearanceConfig struct {
    AccentColor string `toml:"accent_color"` // default: "#e94560"
}
```

**Default config values:**

```go
func defaults() *Config {
    return &Config{
        Cache: CacheConfig{
            MaxSizeMB: 2048,
            Directory: "",
        },
        Playback: PlaybackConfig{
            Volume:       80,
            BufferSizeKB: 8,
        },
        YtDlp: YtDlpConfig{
            Path:      "yt-dlp",
            RateLimit: "",
        },
        FFmpeg: FFmpegConfig{
            Path:       "ffmpeg",
            Threads:    1,
            SampleRate: 44100,
        },
        Appearance: AppearanceConfig{
            AccentColor: "#e94560",
        },
    }
}
```

**Load function:**

```go
func Load() (*Config, error) {
    cfg := defaults()
    path := configFilePath()

    if _, err := os.Stat(path); os.IsNotExist(err) {
        if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
            return cfg, nil // can't write config dir, use defaults silently
        }
        f, err := os.Create(path)
        if err != nil {
            return cfg, nil
        }
        defer f.Close()
        toml.NewEncoder(f).Encode(cfg)
        return cfg, nil
    }

    if _, err := toml.DecodeFile(path, cfg); err != nil {
        return nil, fmt.Errorf("config: parse error in %s: %w", path, err)
    }
    return cfg, nil
}

func configFilePath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "gotune", "config.toml")
}
```

**CacheDir resolution:**

```go
func (c *CacheConfig) ResolvedDir() string {
    if c.Directory != "" {
        return c.Directory
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".cache", "gotune")
}
```

---

## 6. The Provider Layer

### Purpose

Search YouTube Music. Return structured track metadata. Does not return stream URLs — that is the extractor's responsibility.

### File: `internal/provider/provider.go`

```go
package provider

type Track struct {
    VideoID  string
    Title    string
    Artist   string
    Album    string
    Duration int    // seconds, 0 if unknown
    ThumbURL string
}

type Provider interface {
    Search(query string) ([]Track, error)
    GetRadio(seedVideoID string) ([]Track, error)
}
```

### File: `internal/provider/ytmusic.go`

#### The HTTP client

Create one `http.Client` per `YTMusicProvider` instance and reuse it. This keeps TCP connections alive between requests (connection pooling):

```go
type YTMusicProvider struct {
    client       *http.Client
    lastRequest  time.Time
    minDelay     time.Duration // 300ms between requests
}

func New() *YTMusicProvider {
    return &YTMusicProvider{
        client: &http.Client{
            Timeout: 15 * time.Second,
        },
        minDelay: 300 * time.Millisecond,
    }
}
```

#### Rate limiting

Before every HTTP request, enforce the minimum delay:

```go
func (p *YTMusicProvider) throttle() {
    since := time.Since(p.lastRequest)
    if since < p.minDelay {
        time.Sleep(p.minDelay - since)
    }
    p.lastRequest = time.Now()
}
```

#### The search endpoint

URL:
```
https://music.youtube.com/youtubei/v1/search?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE&prettyPrint=false
```

The API key above is YouTube's public client key embedded in their web JS. It has not changed in years. If it stops working, open music.youtube.com in a browser, open DevTools → Network tab, search for something, inspect the POST request URL to find the current key.

Required headers:

```go
req.Header.Set("Content-Type", "application/json")
req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
req.Header.Set("X-YouTube-Client-Name", "67")
req.Header.Set("X-YouTube-Client-Version", "1.20231204.01.00")
req.Header.Set("Origin", "https://music.youtube.com")
req.Header.Set("Referer", "https://music.youtube.com/")
req.Header.Set("Accept-Language", "en-US,en;q=0.9")
```

Client name `67` is the `WEB_REMIX` client. Using client name `1` (WEB) will return general YouTube results, not music-formatted ones.

Request body:

```go
type ytContext struct {
    Client struct {
        ClientName    string `json:"clientName"`
        ClientVersion string `json:"clientVersion"`
        HL            string `json:"hl"`
        GL            string `json:"gl"`
    } `json:"client"`
}

type searchRequest struct {
    Context ytContext `json:"context"`
    Query   string    `json:"query"`
    Params  string    `json:"params"`
}

func buildContext() ytContext {
    var ctx ytContext
    ctx.Client.ClientName = "WEB_REMIX"
    ctx.Client.ClientVersion = "1.20231204.01.00"
    ctx.Client.HL = "en"
    ctx.Client.GL = "US"
    return ctx
}
```

The `params` field is a base64-encoded protobuf that filters results to the Songs tab:

```go
const searchParamsSongs = "EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D"
```

To get all results (songs + videos + albums): omit the `params` field entirely. For V1, always use `searchParamsSongs`.

#### Response parsing

The response JSON is ~50KB of deeply nested objects. Use `map[string]interface{}` with a safe recursive accessor. Do not use typed structs for the response — the shape has too many optional fields.

```go
func dig(v interface{}, keys ...string) interface{} {
    for _, k := range keys {
        switch node := v.(type) {
        case map[string]interface{}:
            val, ok := node[k]
            if !ok {
                return nil
            }
            v = val
        default:
            return nil
        }
    }
    return v
}

func digIndex(v interface{}, index int) interface{} {
    arr, ok := v.([]interface{})
    if !ok || index >= len(arr) {
        return nil
    }
    return arr[index]
}

func digStr(v interface{}, keys ...string) string {
    result := dig(v, keys...)
    if result == nil {
        return ""
    }
    s, _ := result.(string)
    return s
}
```

The path to individual track renderers in the search response:

```
root
  .contents
  .tabbedSearchResultsRenderer
  .tabs[0]
  .tabRenderer
  .content
  .sectionListRenderer
  .contents    <- slice, iterate all elements
    [N].musicShelfRenderer   <- only process elements that have this key
      .contents              <- slice of track renderers
        [M].musicResponsiveListItemRenderer  <- one track
```

From each `musicResponsiveListItemRenderer`:

```go
func parseTrack(renderer interface{}) *provider.Track {
    videoID := digStr(renderer,
        "overlay", "musicItemThumbnailOverlayRenderer",
        "content", "musicPlayButtonRenderer",
        "playNavigationEndpoint", "watchEndpoint", "videoId",
    )
    if videoID == "" {
        return nil // not a playable track (e.g. a podcast or ad unit)
    }

    title := digStr(renderer,
        "flexColumns", // note: dig handles string keys, not int indices
        // use digIndex for array access
    )
    // full path for title:
    // flexColumns[0].musicResponsiveListItemFlexColumnRenderer.text.runs[0].text

    artist := // flexColumns[1].musicResponsiveListItemFlexColumnRenderer.text.runs[0].text
    album  := // flexColumns[1].musicResponsiveListItemFlexColumnRenderer.text.runs[2].text (may be nil)

    durationStr := // fixedColumns[0].musicResponsiveListItemFixedColumnRenderer.text.runs[0].text

    thumbURL := "" // thumbnail.musicThumbnailRenderer.thumbnail.thumbnails[last].url
    if thumbs, ok := dig(renderer, "thumbnail", "musicThumbnailRenderer",
        "thumbnail", "thumbnails").([]interface{}); ok && len(thumbs) > 0 {
        thumbURL = digStr(thumbs[len(thumbs)-1], "url")
    }

    return &provider.Track{
        VideoID:  videoID,
        Title:    title,
        Artist:   artist,
        Album:    album,
        Duration: parseDuration(durationStr),
        ThumbURL: thumbURL,
    }
}
```

Since `dig` works with string keys and `digIndex` with integer indices, you need both for array access. The `flexColumns` path requires alternating between the two. Write it explicitly for each field rather than trying to make a generic deep-path accessor — clarity matters here because you will need to debug this when YouTube changes the response shape.

#### Duration parsing

Durations come as strings: `"3:45"`, `"1:03:22"`, `"0:45"`.

```go
func parseDuration(s string) int {
    if s == "" {
        return 0
    }
    parts := strings.Split(s, ":")
    total := 0
    for _, p := range parts {
        n, err := strconv.Atoi(strings.TrimSpace(p))
        if err != nil {
            return 0
        }
        total = total*60 + n
    }
    return total
}
```

#### Error detection

YouTube returns HTTP 200 even for some API errors. Check for the error field:

```go
var topLevel map[string]json.RawMessage
if err := json.Unmarshal(body, &topLevel); err != nil {
    return nil, fmt.Errorf("provider: invalid JSON response: %w", err)
}
if errVal, ok := topLevel["error"]; ok {
    var apiErr struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    }
    json.Unmarshal(errVal, &apiErr)
    if apiErr.Code == 429 {
        return nil, ErrRateLimited
    }
    return nil, fmt.Errorf("provider: API error %d: %s", apiErr.Code, apiErr.Message)
}
```

Define `ErrRateLimited` as a sentinel:

```go
var ErrRateLimited = errors.New("rate limited by YouTube")
```

The caller (UI layer) will check for this specifically to show a different toast message.

#### GetRadio implementation

Uses the `next` endpoint to fetch a continuous radio/watch playlist seeded by a video ID:

```
POST https://music.youtube.com/youtubei/v1/next?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE&prettyPrint=false
```

Request body:

```go
type radioRequest struct {
    Context     ytContext `json:"context"`
    VideoID     string    `json:"videoId"`
    IsAudioOnly bool      `json:"isAudioOnly"`
}
```

Set `IsAudioOnly: true`. This biases results toward audio tracks rather than music videos.

Response path to track list:

```
root
  .contents
  .singleColumnMusicWatchNextResultsRenderer
  .tabbedRenderer
  .watchNextTabbedResultsRenderer
  .tabs[0]
  .tabRenderer
  .content
  .musicQueueRenderer
  .content
  .playlistPanelRenderer
  .contents        <- slice of tracks
    [N].playlistPanelVideoRenderer
      .videoId     <- the video ID
      .title.runs[0].text
      .longBylineText.runs[0].text  <- artist
      .lengthText.runs[0].text      <- duration string
```

Return the first 25 tracks. Skip the first one (it is the seed track itself).

---

## 7. The Extractor Layer

### Purpose

Given a video ID, return a direct time-limited stream URL for audio-only playback, plus the confirmed duration in seconds.

### File: `internal/extractor/extractor.go`

```go
package extractor

type StreamInfo struct {
    URL      string
    Format   string // "webm" (opus) or "m4a" (aac)
    Duration float64 // seconds
}

type Extractor interface {
    Extract(videoID string) (*StreamInfo, error)
}
```

### File: `internal/extractor/ytdlp.go`

#### The command

```go
func (e *YtDlpExtractor) Extract(videoID string) (*StreamInfo, error) {
    ytURL := "https://www.youtube.com/watch?v=" + videoID

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, e.cfg.YtDlp.Path,
        "--no-playlist",
        "--no-warnings",
        "--format", "bestaudio[ext=webm]/bestaudio[ext=m4a]/bestaudio",
        "--print", "%(url)s\t%(duration)s\t%(ext)s",
        ytURL,
    )

    out, err := cmd.Output()
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("extractor: yt-dlp timed out after 30s for %s", videoID)
        }
        if exitErr, ok := err.(*exec.ExitError); ok {
            stderr := strings.TrimSpace(string(exitErr.Stderr))
            return nil, fmt.Errorf("extractor: yt-dlp failed: %s", stderr)
        }
        return nil, fmt.Errorf("extractor: yt-dlp exec error: %w", err)
    }

    line := strings.TrimSpace(string(out))
    parts := strings.SplitN(line, "\t", 3)
    if len(parts) != 3 {
        return nil, fmt.Errorf("extractor: unexpected yt-dlp output: %q", line)
    }

    streamURL := parts[0]
    duration, _ := strconv.ParseFloat(parts[1], 64)
    format := parts[2]

    if streamURL == "" || !strings.HasPrefix(streamURL, "http") {
        return nil, fmt.Errorf("extractor: invalid URL returned for %s", videoID)
    }

    return &StreamInfo{
        URL:      streamURL,
        Format:   format,
        Duration: duration,
    }, nil
}
```

Using `--print` with a tab-separated template gets all three values in one subprocess call. This is better than calling yt-dlp twice.

#### Format selection rationale

`bestaudio[ext=webm]` selects Opus audio in WebM container (itag 251, ~160kbps VBR). This is the highest-quality audio-only format YouTube provides. ffmpeg decodes Opus natively with no issues.

`bestaudio[ext=m4a]` is the fallback (itag 140, ~128kbps AAC). Also handled fine by ffmpeg.

`bestaudio` catches anything else if neither WebM nor M4A is available.

#### URL expiry

Stream URLs from yt-dlp are valid for approximately 6 hours. They are cryptographically signed and bound to the extraction time. Do not store them in the cache index. Cache the audio file instead.

#### RateLimit option

If configured, pass to yt-dlp:

```go
if e.cfg.YtDlp.RateLimit != "" {
    args = append(args, "--rate-limit", e.cfg.YtDlp.RateLimit)
}
```

---

## 8. The Player Layer

### Purpose

Decode and play audio. Control playback (pause, resume, seek, volume, stop). Report current position and playing state.

### Architecture

```
ffmpeg subprocess
   reads from: stream URL over HTTP
   writes to: stdout pipe (raw PCM s16le, 44100Hz, stereo)
       |
       v
Go goroutine reads PCM bytes from the pipe
       |
       v
oto/v2 player.Write(pcmData)
       |
       v
PipeWire/PulseAudio → speakers
```

### File: `internal/player/output.go`

The oto player is initialized once for the lifetime of the application. It is not recreated between tracks:

```go
package player

import "github.com/hajimehoshi/oto/v2"

type Speaker struct {
    ctx    *oto.Context
    player oto.Player
}

func NewSpeaker(sampleRate int, bufferSizeKB int) (*Speaker, error) {
    // oto.NewContext initializes the audio output device
    // channelCount=2 (stereo), bitDepthInBytes=2 (16-bit = s16le)
    ctx, readyCh, err := oto.NewContext(sampleRate, 2, 2)
    if err != nil {
        return nil, fmt.Errorf("player: failed to open audio device: %w", err)
    }
    // Wait for the audio context to be ready
    <-readyCh
    return &Speaker{ctx: ctx}, nil
}
```

The `bufferSizeBytes` for oto is `bufferSizeKB * 1024`. At 44100Hz, stereo, 16-bit (2 bytes/sample), 8KB buffer is 8192 / (44100 * 2 * 2) = ~46ms latency. This is imperceptible. Do not go below 4096 (glitches) or above 32768 (sluggish seek response).

### File: `internal/player/ffmpeg.go`

#### State

```go
type FFmpegPlayer struct {
    cfg     *config.Config
    speaker *Speaker

    mu          sync.Mutex
    cmd         *exec.Cmd
    pipe        io.ReadCloser   // ffmpeg stdout
    otoPlayer   oto.Player

    paused      bool
    volume      float64         // 0.0 to 1.0
    position    float64         // seconds, updated by read loop
    duration    float64         // seconds, from extractor
    sampleRate  int
    bytesPerSec int             // sampleRate * channels * bytesPerSample

    stopCh      chan struct{}    // close to stop the read loop
    doneCh      chan struct{}    // closed when read loop exits
    onEnd       func()          // called when track finishes naturally
}
```

#### Starting a track

```go
func (p *FFmpegPlayer) Play(streamURL string, duration float64, onEnd func()) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Stop any current playback first
    p.stopLocked()

    p.duration = duration
    p.position = 0
    p.onEnd = onEnd
    p.stopCh = make(chan struct{})
    p.doneCh = make(chan struct{})

    // Launch ffmpeg
    p.cmd = exec.Command(p.cfg.FFmpeg.Path,
        "-hide_banner",
        "-loglevel", "error",
        "-i", streamURL,
        "-vn",                              // no video
        "-sn",                              // no subtitles
        "-threads", strconv.Itoa(p.cfg.FFmpeg.Threads),
        "-f", "s16le",                      // raw signed 16-bit little-endian PCM
        "-ar", strconv.Itoa(p.sampleRate),  // sample rate
        "-ac", "2",                         // stereo
        "pipe:1",                           // output to stdout
    )
    // Capture stderr so it doesn't leak into the terminal
    var stderrBuf bytes.Buffer
    p.cmd.Stderr = &stderrBuf

    var err error
    p.pipe, err = p.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("player: stdout pipe: %w", err)
    }

    if err := p.cmd.Start(); err != nil {
        return fmt.Errorf("player: ffmpeg start: %w", err)
    }

    // Create a new oto player for this track
    p.otoPlayer = p.speaker.ctx.NewPlayer(p)

    p.otoPlayer.Play()

    go p.readLoop(p.stopCh, p.doneCh, &stderrBuf)
    return nil
}
```

#### The FFmpegPlayer as io.Reader

oto's `Context.NewPlayer` takes an `io.Reader`. Make `FFmpegPlayer` implement `io.Reader`. oto calls `Read` when it needs more audio data:

```go
func (p *FFmpegPlayer) Read(buf []byte) (int, error) {
    p.mu.Lock()
    paused := p.paused
    pipe := p.pipe
    p.mu.Unlock()

    if paused {
        // When paused, block until unpaused or stopped
        for {
            time.Sleep(10 * time.Millisecond)
            p.mu.Lock()
            paused = p.paused
            pipe = p.pipe
            p.mu.Unlock()
            if !paused || pipe == nil {
                break
            }
        }
    }

    if pipe == nil {
        return 0, io.EOF
    }

    n, err := pipe.Read(buf)

    // Update position counter
    if n > 0 {
        p.mu.Lock()
        p.position += float64(n) / float64(p.bytesPerSec)
        p.mu.Unlock()
    }

    return n, err
}
```

Note: `bytesPerSec = sampleRate * channels * bytesPerSample = 44100 * 2 * 2 = 176400`.

#### The read loop

The read loop monitors when ffmpeg exits and cleans up:

```go
func (p *FFmpegPlayer) readLoop(stopCh <-chan struct{}, doneCh chan<- struct{}, stderrBuf *bytes.Buffer) {
    defer close(doneCh)

    // Wait for ffmpeg to exit, or for a stop signal
    exitCh := make(chan error, 1)
    go func() { exitCh <- p.cmd.Wait() }()

    select {
    case err := <-exitCh:
        if err != nil {
            // ffmpeg exited with an error — log it but don't crash
            // The oto player will get io.EOF from Read and stop naturally
        }
        // Natural end of track
        if p.onEnd != nil {
            p.onEnd()
        }
    case <-stopCh:
        // Explicitly stopped — kill ffmpeg
        p.cmd.Process.Kill()
        <-exitCh
    }
}
```

#### Pause and Resume

```go
func (p *FFmpegPlayer) Pause() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.paused = true
    if p.otoPlayer != nil {
        p.otoPlayer.Pause()
    }
}

func (p *FFmpegPlayer) Resume() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.paused = false
    if p.otoPlayer != nil {
        p.otoPlayer.Play()
    }
}

func (p *FFmpegPlayer) TogglePause() {
    p.mu.Lock()
    paused := p.paused
    p.mu.Unlock()
    if paused {
        p.Resume()
    } else {
        p.Pause()
    }
}
```

#### Seek

Seeking requires restarting ffmpeg at a new position. oto will get a new `io.Reader` with audio starting from the seek point:

```go
func (p *FFmpegPlayer) Seek(seconds float64) error {
    p.mu.Lock()
    streamURL := p.currentURL // store this when Play() is called
    duration := p.duration
    onEnd := p.onEnd
    p.mu.Unlock()

    if seconds < 0 { seconds = 0 }
    if seconds >= duration { seconds = duration - 1 }

    // Stop current playback
    p.mu.Lock()
    p.stopLocked()
    p.mu.Unlock()

    // Restart from new position
    p.position = seconds
    p.stopCh = make(chan struct{})
    p.doneCh = make(chan struct{})

    cmd := exec.Command(p.cfg.FFmpeg.Path,
        "-hide_banner",
        "-loglevel", "error",
        "-ss", fmt.Sprintf("%.2f", seconds), // seek BEFORE -i for fast seek
        "-i", streamURL,
        "-vn", "-sn",
        "-threads", strconv.Itoa(p.cfg.FFmpeg.Threads),
        "-f", "s16le",
        "-ar", strconv.Itoa(p.sampleRate),
        "-ac", "2",
        "pipe:1",
    )

    // ... same setup as Play() from here
}
```

Important: placing `-ss` before `-i` in the ffmpeg command is a fast seek (seeks to the nearest keyframe before the target, then decodes forward). Placing it after `-i` is slow/accurate seek. For audio, fast seek is acceptable since audio keyframes are dense.

#### Volume

oto does not have a built-in volume control. Implement software volume in the `Read` function by scaling PCM samples:

```go
func applyVolume(buf []byte, volume float64) {
    if volume == 1.0 {
        return
    }
    for i := 0; i+1 < len(buf); i += 2 {
        sample := int16(buf[i]) | int16(buf[i+1])<<8
        scaled := int32(float64(sample) * volume)
        // Clamp to int16 range
        if scaled > 32767  { scaled = 32767  }
        if scaled < -32768 { scaled = -32768 }
        buf[i]   = byte(scaled)
        buf[i+1] = byte(scaled >> 8)
    }
}
```

Call this in `Read()` after reading from the pipe:

```go
n, err := pipe.Read(buf)
if n > 0 {
    p.mu.Lock()
    vol := p.volume
    p.mu.Unlock()
    applyVolume(buf[:n], vol)
}
```

```go
func (p *FFmpegPlayer) SetVolume(pct int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if pct < 0   { pct = 0   }
    if pct > 100 { pct = 100 }
    p.volume = float64(pct) / 100.0
}
```

#### Stop

```go
func (p *FFmpegPlayer) stopLocked() {
    // Must be called with p.mu held
    if p.stopCh != nil {
        close(p.stopCh)
        p.stopCh = nil
    }
    if p.doneCh != nil {
        p.mu.Unlock()
        <-p.doneCh // wait for read loop to exit
        p.mu.Lock()
        p.doneCh = nil
    }
    if p.otoPlayer != nil {
        p.otoPlayer.Close()
        p.otoPlayer = nil
    }
    if p.pipe != nil {
        p.pipe.Close()
        p.pipe = nil
    }
}
```

#### Exported state accessors

```go
func (p *FFmpegPlayer) Position() float64 {
    p.mu.Lock(); defer p.mu.Unlock()
    return p.position
}

func (p *FFmpegPlayer) Duration() float64 {
    p.mu.Lock(); defer p.mu.Unlock()
    return p.duration
}

func (p *FFmpegPlayer) IsPaused() bool {
    p.mu.Lock(); defer p.mu.Unlock()
    return p.paused
}

func (p *FFmpegPlayer) IsPlaying() bool {
    p.mu.Lock(); defer p.mu.Unlock()
    return p.pipe != nil && !p.paused
}

func (p *FFmpegPlayer) Volume() int {
    p.mu.Lock(); defer p.mu.Unlock()
    return int(p.volume * 100)
}
```

---

## 9. The Cache Layer

### Purpose

Store audio files locally so repeated plays of the same track avoid a network request and yt-dlp extraction overhead.

### Directories

```
~/.cache/gotune/
  audio/       ← completed audio files, named {videoID}.{ext}
  partial/     ← in-progress downloads only, named {videoID}.part
  index.db     ← bbolt database
```

On startup, delete everything in `partial/`. A partial file is always a failed or abandoned download.

### File: `internal/cache/index.go`

```go
package cache

import (
    "encoding/json"
    "time"
    bolt "go.etcd.io/bbolt"
)

var bucketName = []byte("tracks")

type Entry struct {
    VideoID       string    `json:"video_id"`
    FilePath      string    `json:"file_path"`
    Format        string    `json:"format"`
    FileSizeBytes int64     `json:"file_size_bytes"`
    LastAccessed  time.Time `json:"last_accessed"`
}

type Index struct {
    db *bolt.DB
}

func openIndex(path string) (*Index, error) {
    db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
    if err != nil {
        // Possibly corrupted — delete and retry once
        os.Remove(path)
        db, err = bolt.Open(path, 0600, nil)
        if err != nil {
            return nil, fmt.Errorf("cache: cannot open index: %w", err)
        }
    }
    err = db.Update(func(tx *bolt.Tx) error {
        _, err := tx.CreateBucketIfNotExists(bucketName)
        return err
    })
    return &Index{db: db}, err
}

func (idx *Index) Get(videoID string) (*Entry, error) {
    var entry Entry
    err := idx.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(bucketName)
        v := b.Get([]byte(videoID))
        if v == nil {
            return ErrNotFound
        }
        return json.Unmarshal(v, &entry)
    })
    if err != nil {
        return nil, err
    }
    return &entry, nil
}

func (idx *Index) Put(entry *Entry) error {
    return idx.db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket(bucketName)
        v, _ := json.Marshal(entry)
        return b.Put([]byte(entry.VideoID), v)
    })
}

func (idx *Index) Delete(videoID string) error {
    return idx.db.Update(func(tx *bolt.Tx) error {
        return tx.Bucket(bucketName).Delete([]byte(videoID))
    })
}

func (idx *Index) All() ([]*Entry, error) {
    var entries []*Entry
    err := idx.db.View(func(tx *bolt.Tx) error {
        return tx.Bucket(bucketName).ForEach(func(k, v []byte) error {
            var e Entry
            if err := json.Unmarshal(v, &e); err != nil {
                return nil // skip corrupt entries
            }
            entries = append(entries, &e)
            return nil
        })
    })
    return entries, err
}

var ErrNotFound = errors.New("cache: entry not found")
```

### File: `internal/cache/cache.go`

```go
type Cache struct {
    audioDir   string
    partialDir string
    index      *Index
    maxBytes   int64
    cfg        *config.Config
}

func New(cfg *config.Config) (*Cache, error) {
    base := cfg.Cache.ResolvedDir()
    audioDir   := filepath.Join(base, "audio")
    partialDir := filepath.Join(base, "partial")
    indexPath  := filepath.Join(base, "index.db")

    for _, dir := range []string{audioDir, partialDir} {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return nil, fmt.Errorf("cache: mkdir %s: %w", dir, err)
        }
    }

    idx, err := openIndex(indexPath)
    if err != nil {
        return nil, err
    }

    c := &Cache{
        audioDir:   audioDir,
        partialDir: partialDir,
        index:      idx,
        maxBytes:   int64(cfg.Cache.MaxSizeMB) * 1024 * 1024,
        cfg:        cfg,
    }

    c.cleanPartials()
    return c, nil
}

func (c *Cache) cleanPartials() {
    filepath.WalkDir(c.partialDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return nil
        }
        os.Remove(path)
        return nil
    })
}
```

#### Get

```go
func (c *Cache) Get(videoID string) (filePath string, hit bool) {
    entry, err := c.index.Get(videoID)
    if err != nil {
        return "", false
    }
    // Verify file actually exists (index can be stale after manual deletion)
    if _, err := os.Stat(entry.FilePath); err != nil {
        c.index.Delete(videoID) // repair stale index entry
        return "", false
    }
    // Update last accessed time
    entry.LastAccessed = time.Now()
    c.index.Put(entry)
    return entry.FilePath, true
}
```

#### DownloadAsync

Called after a track finishes playing (natural EOF). Downloads the track for future cache hits. Runs entirely in a goroutine.

```go
func (c *Cache) DownloadAsync(videoID string, format string) {
    go func() {
        // Check if already cached (could have been downloaded by a concurrent call)
        if _, hit := c.Get(videoID); hit {
            return
        }

        partPath  := filepath.Join(c.partialDir, videoID+".part")
        finalPath := filepath.Join(c.audioDir, videoID+"."+format)

        cmd := exec.Command(c.cfg.YtDlp.Path,
            "--no-playlist",
            "--no-warnings",
            "--format", "bestaudio[ext=webm]/bestaudio[ext=m4a]/bestaudio",
            "--output", partPath,
            "https://www.youtube.com/watch?v="+videoID,
        )
        if err := cmd.Run(); err != nil {
            os.Remove(partPath)
            return
        }

        if err := os.Rename(partPath, finalPath); err != nil {
            os.Remove(partPath)
            return
        }

        info, err := os.Stat(finalPath)
        if err != nil {
            return
        }

        c.index.Put(&Entry{
            VideoID:       videoID,
            FilePath:      finalPath,
            Format:        format,
            FileSizeBytes: info.Size(),
            LastAccessed:  time.Now(),
        })

        c.maybeEvict()
    }()
}
```

#### LRU Eviction

```go
func (c *Cache) maybeEvict() {
    entries, err := c.index.All()
    if err != nil {
        return
    }

    var total int64
    for _, e := range entries {
        total += e.FileSizeBytes
    }

    if total <= c.maxBytes {
        return
    }

    // Sort oldest-accessed first
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].LastAccessed.Before(entries[j].LastAccessed)
    })

    for _, e := range entries {
        if total <= c.maxBytes {
            break
        }
        os.Remove(e.FilePath)
        c.index.Delete(e.VideoID)
        total -= e.FileSizeBytes
    }
}
```

#### Stats and Clear

```go
type Stats struct {
    TrackCount    int
    UsedBytes     int64
    MaxBytes      int64
    Directory     string
}

func (c *Cache) Stats() (Stats, error) {
    entries, err := c.index.All()
    if err != nil {
        return Stats{}, err
    }
    s := Stats{
        MaxBytes:  c.maxBytes,
        Directory: c.audioDir,
    }
    for _, e := range entries {
        s.TrackCount++
        s.UsedBytes += e.FileSizeBytes
    }
    return s, nil
}

func (c *Cache) Clear() (int64, error) {
    entries, err := c.index.All()
    if err != nil {
        return 0, err
    }
    var freed int64
    for _, e := range entries {
        if err := os.Remove(e.FilePath); err == nil {
            freed += e.FileSizeBytes
        }
        c.index.Delete(e.VideoID)
    }
    return freed, nil
}
```

---

## 10. The MPRIS2 Layer

### Purpose

Register gotune as an MPRIS2-compliant media player on the D-Bus session bus. This makes media keys (play/pause, next, previous) work in GNOME, KDE, and any MPRIS2-aware desktop environment.

### How MPRIS2 works on Linux

When you press the play/pause media key, the desktop environment looks for a D-Bus service named `org.mpris.MediaPlayer2.*` on the session bus and sends a method call to it. Your application registers under this name and responds to those calls by controlling playback.

Additionally, GNOME's top bar and KDE's taskbar widget display now-playing info by reading `Metadata` and `PlaybackStatus` properties from your service.

### File: `internal/mpris/mpris.go`

```go
package mpris

import (
    "github.com/godbus/dbus/v5"
    "github.com/godbus/dbus/v5/prop"
)

const (
    serviceName   = "org.mpris.MediaPlayer2.gotune"
    objectPath    = "/org/mpris/MediaPlayer2"
    playerIface   = "org.mpris.MediaPlayer2.Player"
    rootIface     = "org.mpris.MediaPlayer2"
    propsIface    = "org.freedesktop.DBus.Properties"
)

type Service struct {
    conn    *dbus.Conn
    props   *prop.Properties
    program interface{ Send(interface{}) } // tea.Program or a channel
}

func Register(program interface{ Send(interface{}) }) (*Service, error) {
    conn, err := dbus.SessionBus()
    if err != nil {
        return nil, fmt.Errorf("mpris: D-Bus session bus unavailable: %w", err)
    }

    reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("mpris: RequestName failed: %w", err)
    }
    if reply != dbus.RequestNameReplyPrimaryOwner {
        conn.Close()
        return nil, fmt.Errorf("mpris: name already taken (another gotune instance running?)")
    }

    s := &Service{conn: conn, program: program}

    // Export the root interface
    conn.Export(s, dbus.ObjectPath(objectPath), rootIface)
    // Export the player interface
    conn.Export(s, dbus.ObjectPath(objectPath), playerIface)

    // Set up initial properties
    s.props = s.initProperties()
    conn.Export(s.props, dbus.ObjectPath(objectPath), propsIface)

    return s, nil
}

func (s *Service) Close() {
    if s.conn != nil {
        s.conn.ReleaseName(serviceName)
        s.conn.Close()
    }
}
```

#### Root interface methods (org.mpris.MediaPlayer2)

```go
// Raise brings the app to the foreground — not applicable for TUI, no-op.
func (s *Service) Raise() *dbus.Error { return nil }

// Quit exits the application.
func (s *Service) Quit() *dbus.Error {
    s.program.Send(QuitMsg{})
    return nil
}
```

Root interface properties (read-only, returned via the Properties interface):

```
CanQuit          = true
CanRaise         = false
HasTrackList     = false
Identity         = "GoTune"
SupportedUriSchemes = ["http", "https"]
SupportedMimeTypes  = ["audio/webm", "audio/mp4", "audio/mpeg"]
```

#### Player interface methods (org.mpris.MediaPlayer2.Player)

```go
func (s *Service) Play() *dbus.Error {
    s.program.Send(MediaKeyMsg{Action: ActionPlay})
    return nil
}

func (s *Service) Pause() *dbus.Error {
    s.program.Send(MediaKeyMsg{Action: ActionPause})
    return nil
}

func (s *Service) PlayPause() *dbus.Error {
    s.program.Send(MediaKeyMsg{Action: ActionPlayPause})
    return nil
}

func (s *Service) Stop() *dbus.Error {
    s.program.Send(MediaKeyMsg{Action: ActionStop})
    return nil
}

func (s *Service) Next() *dbus.Error {
    s.program.Send(MediaKeyMsg{Action: ActionNext})
    return nil
}

func (s *Service) Previous() *dbus.Error {
    s.program.Send(MediaKeyMsg{Action: ActionPrevious})
    return nil
}

func (s *Service) Seek(offset int64) *dbus.Error {
    // offset is in microseconds
    s.program.Send(MediaSeekMsg{OffsetSeconds: float64(offset) / 1e6})
    return nil
}

func (s *Service) SetPosition(trackID dbus.ObjectPath, position int64) *dbus.Error {
    // position is in microseconds
    s.program.Send(MediaSeekMsg{AbsoluteSeconds: float64(position) / 1e6})
    return nil
}

func (s *Service) OpenUri(uri string) *dbus.Error {
    // Not supported in V1
    return nil
}
```

#### Messages sent to bubbletea

```go
// In internal/mpris/mpris.go or internal/ui/model.go

type MediaKeyMsg struct {
    Action MediaAction
}

type MediaSeekMsg struct {
    OffsetSeconds   float64 // relative seek (from Seek method)
    AbsoluteSeconds float64 // absolute seek (from SetPosition method)
    IsAbsolute      bool
}

type QuitMsg struct{}

type MediaAction int
const (
    ActionPlay MediaAction = iota
    ActionPause
    ActionPlayPause
    ActionStop
    ActionNext
    ActionPrevious
)
```

#### Updating MPRIS properties when state changes

Call these from the UI layer whenever the relevant state changes:

```go
// File: internal/mpris/properties.go

func (s *Service) UpdatePlaybackStatus(playing bool, paused bool) {
    status := "Stopped"
    if playing && !paused {
        status = "Playing"
    } else if playing && paused {
        status = "Paused"
    }

    s.props.SetMust(playerIface, "PlaybackStatus", dbus.MakeVariant(status))
    s.emitPropertiesChanged(playerIface, map[string]dbus.Variant{
        "PlaybackStatus": dbus.MakeVariant(status),
    })
}

func (s *Service) UpdateMetadata(track *provider.Track) {
    trackID := dbus.ObjectPath(fmt.Sprintf("/org/mpris/MediaPlayer2/track/%s", track.VideoID))
    meta := map[string]dbus.Variant{
        "mpris:trackid":   dbus.MakeVariant(trackID),
        "xesam:title":     dbus.MakeVariant(track.Title),
        "xesam:artist":    dbus.MakeVariant([]string{track.Artist}),
        "xesam:album":     dbus.MakeVariant(track.Album),
        "mpris:length":    dbus.MakeVariant(int64(track.Duration) * 1e6), // microseconds
    }
    s.props.SetMust(playerIface, "Metadata", dbus.MakeVariant(meta))
    s.emitPropertiesChanged(playerIface, map[string]dbus.Variant{
        "Metadata": dbus.MakeVariant(meta),
    })
}

func (s *Service) UpdatePosition(seconds float64) {
    // Position is not emitted via PropertiesChanged — it's polled by clients.
    // Emit Seeked signal only when the position jumps non-linearly (on seek).
    s.props.SetMust(playerIface, "Position", dbus.MakeVariant(int64(seconds*1e6)))
}

func (s *Service) EmitSeeked(seconds float64) {
    s.conn.Emit(dbus.ObjectPath(objectPath), playerIface+".Seeked", int64(seconds*1e6))
}

func (s *Service) emitPropertiesChanged(iface string, changed map[string]dbus.Variant) {
    s.conn.Emit(
        dbus.ObjectPath(objectPath),
        propsIface+".PropertiesChanged",
        iface,
        changed,
        []string{}, // invalidated properties — empty
    )
}
```

#### Graceful MPRIS failure

MPRIS is optional. If D-Bus is unavailable, the app works without media keys:

```go
// In main.go
mprisService, err := mpris.Register(program)
if err != nil {
    log.Printf("MPRIS2 unavailable (media keys won't work): %v", err)
    mprisService = mpris.Noop() // returns a no-op implementation of the same interface
}
```

---

## 11. The TUI Layer

### Framework overview

Bubbletea uses the Elm Architecture:

- `Model`: All state. A Go struct. Immutable during Update — return new copies.
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)`: Handle messages. Return updated model and optional command.
- `View() string`: Render the model to a string. Called after every Update.
- `tea.Cmd`: A `func() tea.Msg`. Used to run async work (HTTP, subprocess) without blocking the loop.

**Critical rules**:
- Never do I/O inside `Update`. Spawn a `Cmd` instead.
- `View` is called very frequently. Keep it fast. No allocations beyond string building.
- Use `lipgloss.JoinVertical` and `lipgloss.JoinHorizontal` for layout.
- Always handle `tea.WindowSizeMsg` to reflow on terminal resize.

### File: `internal/ui/keys.go`

```go
package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
    FocusSearch  key.Binding
    SelectItem   key.Binding
    AddToQueue   key.Binding
    TogglePause  key.Binding
    SkipNext     key.Binding
    SkipPrev     key.Binding
    SeekForward  key.Binding
    SeekBackward key.Binding
    VolumeUp     key.Binding
    VolumeDown   key.Binding
    SwitchView   key.Binding
    CacheManager key.Binding
    CursorUp     key.Binding
    CursorDown   key.Binding
    DeleteItem   key.Binding
    Quit         key.Binding
    Help         key.Binding
    Escape       key.Binding
}

var Keys = KeyMap{
    FocusSearch:  key.NewBinding(key.WithKeys("/"),           key.WithHelp("/", "search")),
    SelectItem:   key.NewBinding(key.WithKeys("enter"),       key.WithHelp("enter", "play")),
    AddToQueue:   key.NewBinding(key.WithKeys("a"),           key.WithHelp("a", "add to queue")),
    TogglePause:  key.NewBinding(key.WithKeys(" "),           key.WithHelp("space", "pause/play")),
    SkipNext:     key.NewBinding(key.WithKeys("n"),           key.WithHelp("n", "next")),
    SkipPrev:     key.NewBinding(key.WithKeys("p"),           key.WithHelp("p", "prev")),
    SeekForward:  key.NewBinding(key.WithKeys("right", "l"),  key.WithHelp("→/l", "+10s")),
    SeekBackward: key.NewBinding(key.WithKeys("left",  "h"),  key.WithHelp("←/h", "-10s")),
    VolumeUp:     key.NewBinding(key.WithKeys("+", "="),      key.WithHelp("+", "vol up")),
    VolumeDown:   key.NewBinding(key.WithKeys("-"),           key.WithHelp("-", "vol down")),
    SwitchView:   key.NewBinding(key.WithKeys("tab"),         key.WithHelp("tab", "search/queue")),
    CacheManager: key.NewBinding(key.WithKeys("C"),           key.WithHelp("C", "cache")),
    CursorUp:     key.NewBinding(key.WithKeys("up",   "k"),   key.WithHelp("↑/k", "up")),
    CursorDown:   key.NewBinding(key.WithKeys("down", "j"),   key.WithHelp("↓/j", "down")),
    DeleteItem:   key.NewBinding(key.WithKeys("d"),           key.WithHelp("d", "remove")),
    Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
    Help:         key.NewBinding(key.WithKeys("?"),           key.WithHelp("?", "help")),
    Escape:       key.NewBinding(key.WithKeys("esc"),         key.WithHelp("esc", "back")),
}
```

### File: `internal/ui/styles.go`

```go
package ui

import "github.com/charmbracelet/lipgloss"

// Base palette
var (
    ColorBg       = lipgloss.AdaptiveColor{Dark: "#0d0d0d", Light: "#f5f5f5"}
    ColorSurface  = lipgloss.AdaptiveColor{Dark: "#1a1a1a", Light: "#e8e8e8"}
    ColorBorder   = lipgloss.AdaptiveColor{Dark: "#333333", Light: "#cccccc"}
    ColorAccent   = lipgloss.Color("#e94560")
    ColorText     = lipgloss.AdaptiveColor{Dark: "#e8e8e8", Light: "#1a1a1a"}
    ColorMuted    = lipgloss.AdaptiveColor{Dark: "#666666", Light: "#888888"}
    ColorPlaying  = lipgloss.Color("#4ade80")
    ColorError    = lipgloss.Color("#f87171")
    ColorInfo     = lipgloss.Color("#60a5fa")
)

var (
    StyleBase = lipgloss.NewStyle().
        Background(ColorBg).
        Foreground(ColorText)

    StyleTitle = lipgloss.NewStyle().
        Bold(true).
        Foreground(ColorAccent)

    StyleSelected = lipgloss.NewStyle().
        Bold(true).
        Foreground(ColorAccent).
        Background(ColorSurface)

    StyleNormal = lipgloss.NewStyle().
        Foreground(ColorText)

    StyleMuted = lipgloss.NewStyle().
        Foreground(ColorMuted)

    StyleBorderBox = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(ColorBorder)

    StylePlayerBar = lipgloss.NewStyle().
        Background(ColorSurface).
        Padding(0, 1)

    StyleToastError = lipgloss.NewStyle().
        Background(ColorError).
        Foreground(lipgloss.Color("#ffffff")).
        Padding(0, 2).
        Bold(true)

    StyleToastInfo = lipgloss.NewStyle().
        Background(ColorInfo).
        Foreground(lipgloss.Color("#000000")).
        Padding(0, 2)

    StyleCacheOverlay = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(ColorAccent).
        Padding(1, 2)
)
```

Provide a `SetAccentColor(hex string)` function that overrides `ColorAccent` and refreshes all styles that depend on it. Called during startup from config.

### File: `internal/ui/model.go`

#### Messages

All async events in the application arrive as these message types:

```go
package ui

// Sent when provider.Search completes
type SearchResultsMsg struct {
    Results []provider.Track
    Err     error
}

// Sent when extractor.Extract completes
type StreamReadyMsg struct {
    VideoID string
    Info    *extractor.StreamInfo
    Err     error
}

// Sent once per second by the tick command
type TickMsg struct {
    Position float64
    Duration float64
    IsPaused bool
}

// Sent when ffmpeg EOF is detected (track ended naturally)
type TrackEndedMsg struct {
    VideoID string
}

// Sent when cache stats are fetched
type CacheStatsMsg struct {
    Stats cache.Stats
}

// Sent when cache clear completes
type CacheClearedMsg struct {
    FreedBytes int64
    Err        error
}

// Temporary notification
type ToastMsg struct {
    Text    string
    IsError bool
}

// Internal: dismiss the toast
type dismissToastMsg struct{}

// From MPRIS2
type MediaKeyMsg  struct{ Action mpris.MediaAction }
type MediaSeekMsg struct{ OffsetSeconds float64; IsAbsolute bool; AbsoluteSeconds float64 }
type QuitMsg      struct{}
```

#### View states

```go
type ViewState int

const (
    ViewSearch ViewState = iota
    ViewQueue
    ViewCacheManager
)
```

#### Root model

```go
type Model struct {
    // Sub-models
    search    SearchModel
    queue     QueueModel
    playerBar PlayerBarModel
    cacheView CacheViewModel
    toast     ToastModel

    // Playback state
    currentTrack *provider.Track
    isPlaying    bool
    isPaused     bool
    position     float64
    duration     float64
    volume       int

    // Services
    prov      provider.Provider
    ext       extractor.Extractor
    player    *player.FFmpegPlayer
    cache     *cache.Cache
    mpris     *mpris.Service

    // Layout
    viewState ViewState
    width     int
    height    int
}
```

#### Update function

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.width  = msg.Width
        m.height = msg.Height
        m.search.width    = msg.Width
        m.search.height   = m.contentHeight()
        m.queue.width     = msg.Width
        m.queue.height    = m.contentHeight()
        return m, nil

    case tea.KeyMsg:
        // Global keys (active regardless of view)
        switch {
        case key.Matches(msg, Keys.Quit):
            return m, tea.Quit
        case key.Matches(msg, Keys.TogglePause):
            return m, m.togglePauseCmd()
        case key.Matches(msg, Keys.SkipNext):
            return m, m.skipNextCmd()
        case key.Matches(msg, Keys.SkipPrev):
            return m, m.skipPrevCmd()
        case key.Matches(msg, Keys.SeekForward):
            return m, m.seekCmd(10)
        case key.Matches(msg, Keys.SeekBackward):
            return m, m.seekCmd(-10)
        case key.Matches(msg, Keys.VolumeUp):
            m.volume = min(100, m.volume+5)
            m.player.SetVolume(m.volume)
            return m, nil
        case key.Matches(msg, Keys.VolumeDown):
            m.volume = max(0, m.volume-5)
            m.player.SetVolume(m.volume)
            return m, nil
        case key.Matches(msg, Keys.SwitchView):
            if m.viewState == ViewSearch {
                m.viewState = ViewQueue
            } else {
                m.viewState = ViewSearch
            }
            return m, nil
        case key.Matches(msg, Keys.CacheManager):
            m.viewState = ViewCacheManager
            return m, fetchCacheStatsCmd(m.cache)
        case key.Matches(msg, Keys.Escape):
            if m.viewState == ViewCacheManager {
                m.viewState = ViewSearch
            }
            return m, nil
        }

        // Delegate remaining keys to the focused view
        switch m.viewState {
        case ViewSearch:
            var cmd tea.Cmd
            m.search, cmd = m.search.Update(msg)
            return m, cmd
        case ViewQueue:
            var cmd tea.Cmd
            m.queue, cmd = m.queue.Update(msg)
            return m, cmd
        case ViewCacheManager:
            var cmd tea.Cmd
            m.cacheView, cmd = m.cacheView.Update(msg)
            return m, cmd
        }

    case SearchResultsMsg:
        if msg.Err != nil {
            if errors.Is(msg.Err, provider.ErrRateLimited) {
                return m, toastCmd("Rate limited by YouTube. Wait a moment.", true)
            }
            return m, toastCmd("Search failed: "+msg.Err.Error(), true)
        }
        m.search.results = msg.Results
        m.search.cursor = 0
        m.search.loading = false
        return m, nil

    case StreamReadyMsg:
        if msg.Err != nil {
            m.playerBar.loading = false
            return m, toastCmd("Cannot play: "+msg.Err.Error(), true)
        }
        m.playerBar.loading = false
        m.isPlaying = true
        m.isPaused = false
        m.duration = msg.Info.Duration
        m.position = 0
        cmd := playCmd(m.player, msg.Info.URL, msg.Info.Duration, func() {
            // This is the onEnd callback — runs in the ffmpeg goroutine
            // Do NOT touch model state here; send a message instead
        })
        m.mpris.UpdateMetadata(m.currentTrack)
        m.mpris.UpdatePlaybackStatus(true, false)
        return m, tea.Batch(cmd, tickCmd(m.player))

    case TrackEndedMsg:
        m.isPlaying = false
        m.cache.DownloadAsync(msg.VideoID, "webm") // fire-and-forget
        return m, m.playNextInQueue()

    case TickMsg:
        m.position = msg.Position
        m.duration = msg.Duration
        m.isPaused = msg.IsPaused
        m.playerBar.position = msg.Position
        m.playerBar.duration = msg.Duration
        m.mpris.UpdatePosition(msg.Position)
        return m, tickCmd(m.player)

    case CacheStatsMsg:
        m.cacheView.stats = msg.Stats
        return m, nil

    case CacheClearedMsg:
        if msg.Err != nil {
            return m, toastCmd("Cache clear failed: "+msg.Err.Error(), true)
        }
        m.viewState = ViewSearch
        freed := formatBytes(msg.FreedBytes)
        return m, toastCmd(fmt.Sprintf("Cache cleared (%s freed)", freed), false)

    case ToastMsg:
        m.toast = m.toast.show(msg.Text, msg.IsError)
        return m, dismissToastAfter(3 * time.Second)

    case dismissToastMsg:
        m.toast.visible = false
        return m, nil

    case MediaKeyMsg:
        switch msg.Action {
        case mpris.ActionPlayPause: return m, m.togglePauseCmd()
        case mpris.ActionNext:      return m, m.skipNextCmd()
        case mpris.ActionPrevious:  return m, m.skipPrevCmd()
        case mpris.ActionStop:
            m.player.Stop()
            m.isPlaying = false
            m.mpris.UpdatePlaybackStatus(false, false)
        }
        return m, nil

    case MediaSeekMsg:
        if msg.IsAbsolute {
            return m, m.seekAbsoluteCmd(msg.AbsoluteSeconds)
        }
        return m, m.seekCmd(msg.OffsetSeconds)

    case QuitMsg:
        return m, tea.Quit
    }

    return m, nil
}
```

#### Commands

```go
func searchCmd(prov provider.Provider, query string) tea.Cmd {
    return func() tea.Msg {
        results, err := prov.Search(query)
        return SearchResultsMsg{Results: results, Err: err}
    }
}

func extractCmd(ext extractor.Extractor, track *provider.Track) tea.Cmd {
    return func() tea.Msg {
        info, err := ext.Extract(track.VideoID)
        return StreamReadyMsg{VideoID: track.VideoID, Info: info, Err: err}
    }
}

func playCmd(p *player.FFmpegPlayer, url string, duration float64, program *tea.Program) tea.Cmd {
    return func() tea.Msg {
        err := p.Play(url, duration, func() {
            program.Send(TrackEndedMsg{VideoID: /* need to thread this through */})
        })
        if err != nil {
            return ToastMsg{Text: "Playback error: " + err.Error(), IsError: true}
        }
        return nil
    }
}

func tickCmd(p *player.FFmpegPlayer) tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg{
            Position: p.Position(),
            Duration: p.Duration(),
            IsPaused: p.IsPaused(),
        }
    })
}

func fetchCacheStatsCmd(c *cache.Cache) tea.Cmd {
    return func() tea.Msg {
        stats, err := c.Stats()
        if err != nil {
            return ToastMsg{Text: "Failed to read cache stats", IsError: true}
        }
        return CacheStatsMsg{Stats: stats}
    }
}

func clearCacheCmd(c *cache.Cache) tea.Cmd {
    return func() tea.Msg {
        freed, err := c.Clear()
        return CacheClearedMsg{FreedBytes: freed, Err: err}
    }
}

func toastCmd(text string, isError bool) tea.Cmd {
    return func() tea.Msg {
        return ToastMsg{Text: text, IsError: isError}
    }
}

func dismissToastAfter(d time.Duration) tea.Cmd {
    return tea.Tick(d, func(t time.Time) tea.Msg {
        return dismissToastMsg{}
    })
}
```

#### View function

```go
func (m Model) View() string {
    if m.width == 0 {
        return "Initializing..."
    }

    playerBarHeight := 4
    toastHeight := 0
    if m.toast.visible {
        toastHeight = 2
    }
    contentH := m.height - playerBarHeight - toastHeight

    var content string
    switch m.viewState {
    case ViewSearch:
        content = m.search.View(m.width, contentH)
    case ViewQueue:
        content = m.queue.View(m.width, contentH)
    case ViewCacheManager:
        bg := m.search.View(m.width, contentH)
        overlay := m.cacheView.View()
        content = lipgloss.Place(m.width, contentH,
            lipgloss.Center, lipgloss.Center,
            overlay,
            lipgloss.WithWhitespaceBackground(ColorBg),
        )
        _ = bg
    }

    parts := []string{}
    if m.toast.visible {
        parts = append(parts, m.toast.View(m.width))
    }
    parts = append(parts, content)
    parts = append(parts, m.playerBar.View(m.width))

    return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) contentHeight() int {
    h := m.height - 4 // player bar
    if m.toast.visible {
        h -= 2
    }
    return h
}
```

### File: `internal/ui/search.go`

```go
type SearchModel struct {
    input   textinput.Model
    results []provider.Track
    cursor  int
    offset  int   // scroll offset for long result lists
    loading bool
    width   int
    height  int
    prov    provider.Provider
}

func NewSearchModel(prov provider.Provider) SearchModel {
    ti := textinput.New()
    ti.Placeholder = "Search YouTube Music..."
    ti.CharLimit = 100
    return SearchModel{input: ti, prov: prov}
}
```

In `Update`, handle `Keys.FocusSearch` to call `ti.Focus()`. Handle `Keys.SelectItem` to either submit the search (if input is focused) or play the selected result (if results are focused).

In `View`:

```go
func (s SearchModel) View(width, height int) string {
    // Line 1: search input box
    inputBox := StyleBorderBox.Width(width - 2).Render(s.input.View())

    // Lines 2 to height-1: result rows
    resultHeight := height - lipgloss.Height(inputBox) - 1
    var rows []string
    visible := min(resultHeight, len(s.results))
    for i := s.offset; i < s.offset+visible && i < len(s.results); i++ {
        row := s.renderRow(s.results[i], i == s.cursor, width)
        rows = append(rows, row)
    }
    resultsPane := strings.Join(rows, "\n")

    return lipgloss.JoinVertical(lipgloss.Left, inputBox, resultsPane)
}

func (s SearchModel) renderRow(t provider.Track, selected bool, width int) string {
    indicator := "  "
    if selected {
        indicator = StyleTitle.Render("▶ ")
    }

    title  := truncate(t.Title,  35)
    artist := truncate(t.Artist, 20)
    album  := truncate(t.Album,  20)
    dur    := formatSeconds(t.Duration)

    // Fixed-width columns using lipgloss
    titleCol  := lipgloss.NewStyle().Width(35).Render(title)
    artistCol := lipgloss.NewStyle().Width(20).Render(artist)
    albumCol  := lipgloss.NewStyle().Width(20).Render(album)
    durCol    := lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Render(dur)

    row := indicator + titleCol + "  " + artistCol + "  " + albumCol + "  " + durCol

    if selected {
        return StyleSelected.Width(width).Render(row)
    }
    return StyleNormal.Render(row)
}
```

Implement scrolling: when `cursor` moves below `offset + visible`, increment `offset`. When it moves above `offset`, decrement it.

### File: `internal/ui/queue.go`

```go
type QueueModel struct {
    tracks  []provider.Track
    cursor  int
    width   int
    height  int
}
```

The queue supports:
- `a` from search view: append track
- `enter` on a queue item: jump to that track immediately
- `d` on a queue item: remove it
- `j/k` or `↑/↓`: move cursor

When a track ends and `queue.Next()` is called, remove the first item and return it. If the queue is empty, call `provider.GetRadio(currentTrack.VideoID)` and append the results.

```go
func (q *QueueModel) Next() *provider.Track {
    if len(q.tracks) == 0 {
        return nil
    }
    track := q.tracks[0]
    q.tracks = q.tracks[1:]
    return &track
}

func (q *QueueModel) Append(t provider.Track) {
    q.tracks = append(q.tracks, t)
}

func (q *QueueModel) Remove(index int) {
    if index < 0 || index >= len(q.tracks) {
        return
    }
    q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
    if q.cursor >= len(q.tracks) {
        q.cursor = max(0, len(q.tracks)-1)
    }
}
```

### File: `internal/ui/playerbar.go`

The player bar is always rendered at the bottom. Height is always 4 lines:

```
Line 1: blank
Line 2: [▶] Title • Artist                        [cached]   🔊 80
Line 3: 1:23 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 3:45
Line 4: blank
```

```go
type PlayerBarModel struct {
    track    *provider.Track
    position float64
    duration float64
    isPaused bool
    isCached bool
    loading  bool
    volume   int
    progress progress.Model
}

func (b PlayerBarModel) View(width int) string {
    if b.track == nil {
        empty := StyleMuted.Render("  No track playing  •  Press / to search")
        return StylePlayerBar.Width(width).Height(4).
            Align(lipgloss.Left, lipgloss.Center).Render(empty)
    }

    icon := "▶"
    if b.isPaused  { icon = "⏸" }
    if b.loading   { icon = "⏳" }

    cachedLabel := ""
    if b.isCached { cachedLabel = StyleMuted.Render("  [cached]") }

    titleLine := fmt.Sprintf("%s  %s  •  %s%s   🔊 %d",
        StyleTitle.Render(icon),
        StyleNormal.Render(truncate(b.track.Title, 40)),
        StyleMuted.Render(truncate(b.track.Artist, 25)),
        cachedLabel,
        b.volume,
    )

    pct := 0.0
    if b.duration > 0 {
        pct = b.position / b.duration
    }
    b.progress.Width = width - 14 // leave room for timestamps
    progressBar := fmt.Sprintf("%s %s %s",
        StyleMuted.Render(formatSeconds(int(b.position))),
        b.progress.ViewAs(pct),
        StyleMuted.Render(formatSeconds(int(b.duration))),
    )

    content := lipgloss.JoinVertical(lipgloss.Left, "", titleLine, progressBar, "")
    return StylePlayerBar.Width(width).Render(content)
}
```

Initialize the progress bar in `NewPlayerBarModel`:

```go
prog := progress.New(
    progress.WithDefaultGradient(),
    progress.WithoutPercentage(),
)
```

### File: `internal/ui/cacheview.go`

```go
type CacheViewModel struct {
    stats    cache.Stats
    loading  bool
}

func (v CacheViewModel) View() string {
    if v.loading {
        return StyleCacheOverlay.Render("Loading cache info...")
    }

    usedStr := formatBytes(v.stats.UsedBytes)
    maxStr  := formatBytes(v.stats.MaxBytes)
    pct     := 0.0
    if v.stats.MaxBytes > 0 {
        pct = float64(v.stats.UsedBytes) / float64(v.stats.MaxBytes) * 100
    }

    lines := []string{
        StyleTitle.Render("Cache Manager"),
        "",
        fmt.Sprintf("  Tracks cached:  %d", v.stats.TrackCount),
        fmt.Sprintf("  Used:           %s / %s  (%.1f%%)", usedStr, maxStr, pct),
        fmt.Sprintf("  Location:       %s", v.stats.Directory),
        "",
        StyleMuted.Render("  [x] Clear all cache"),
        StyleMuted.Render("  [esc] Close"),
    }
    return StyleCacheOverlay.Render(strings.Join(lines, "\n"))
}

func (v CacheViewModel) Update(msg tea.Msg) (CacheViewModel, tea.Cmd) {
    if km, ok := msg.(tea.KeyMsg); ok {
        if km.String() == "x" {
            return v, clearCacheCmd(/* need cache ref */)
        }
    }
    return v, nil
}
```

Pass the `*cache.Cache` into `CacheViewModel` during construction to make `clearCacheCmd` available.

### File: `internal/ui/toast.go`

```go
type ToastModel struct {
    text    string
    isError bool
    visible bool
}

func (t ToastModel) show(text string, isError bool) ToastModel {
    return ToastModel{text: text, isError: isError, visible: true}
}

func (t ToastModel) View(width int) string {
    style := StyleToastInfo
    if t.isError {
        style = StyleToastError
    }
    return style.Width(width).Render(t.text)
}
```

### Helper functions

Define these in a `helpers.go` file within the `ui` package:

```go
func truncate(s string, maxLen int) string {
    runes := []rune(s)
    if len(runes) <= maxLen {
        return s
    }
    return string(runes[:maxLen-1]) + "…"
}

func formatSeconds(total int) string {
    if total < 0 { total = 0 }
    h := total / 3600
    m := (total % 3600) / 60
    s := total % 60
    if h > 0 {
        return fmt.Sprintf("%d:%02d:%02d", h, m, s)
    }
    return fmt.Sprintf("%d:%02d", m, s)
}

func formatBytes(b int64) string {
    switch {
    case b >= 1<<30:
        return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
    case b >= 1<<20:
        return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
    case b >= 1<<10:
        return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
    default:
        return fmt.Sprintf("%d B", b)
    }
}

func min(a, b int) int { if a < b { return a }; return b }
func max(a, b int) int { if a > b { return a }; return b }
```

---

## 12. Concurrency Model

### All goroutines in the running application

1. **Main goroutine**: Runs `tea.Program.Run()` — the bubbletea event loop.
2. **ffmpeg process goroutine**: Launched by `player.Play()`. Monitors when ffmpeg exits and sends `TrackEndedMsg`.
3. **oto internal goroutine**: Created by oto internally. Calls our `Read()` method when it needs PCM data.
4. **Cache download goroutines** (one per track, short-lived): Launched by `cache.DownloadAsync()` after track end.
5. **MPRIS D-Bus dispatch goroutine**: Created internally by `godbus`. Dispatches incoming D-Bus method calls to our handlers.

### Synchronization rules

- The `FFmpegPlayer` struct uses a single `sync.Mutex` (`p.mu`) for all field access.
- `oto`'s call to `Read()` happens in goroutine 3. `Read()` acquires `p.mu` briefly to check pause state and update position. Do not hold `p.mu` while blocked on I/O — this would deadlock.
- The correct pattern for pause: set `p.paused = true` under `p.mu`, then in `Read()` spin-wait (10ms sleep) until `paused` is false. oto's goroutine blocks in `Read()` without holding any lock, so the main goroutine can freely toggle `p.paused`.
- `bbolt` is safe for concurrent use from multiple goroutines without external locking. Its transactions are serialized internally.
- The MPRIS D-Bus handlers (goroutine 5) must not modify model state directly. They call `program.Send(msg)` which is goroutine-safe. The message arrives in bubbletea's Update function on the main goroutine.
- The `onEnd` callback in `player.Play()` runs in goroutine 2 (the ffmpeg watcher). It must call `program.Send(TrackEndedMsg{})`, not modify anything directly.

### Why there is no deadlock risk

- The main goroutine (bubbletea) never blocks — it processes messages and returns.
- The only lock is `p.mu` in the player. It is held for microseconds (field reads/writes), never during I/O or sleeps.
- The oto goroutine blocks in `Read()` waiting for PCM data, but `Read()` only sleeps when paused and releases `p.mu` before sleeping.
- D-Bus and bbolt have their own internal serialization.

---

## 13. Error Handling

### Philosophy

Errors fall into three categories. Handle each differently.

**Fatal errors** (can't proceed): Missing system dependency, audio device unavailable, config parse error. Print to stderr and `os.Exit(1)`. Do this before starting the TUI so the error is visible.

**Recoverable errors** (single operation failed): Search failed, track extraction failed, cache write failed. Show a toast notification. Continue running.

**Ignored errors** (best-effort operations): Cache eviction failed, MPRIS property update failed, background download failed. Log to a debug log file (not terminal). Do not interrupt the user.

### Error wrapping convention

Always wrap with context:

```go
return fmt.Errorf("cache.Get: stat file: %w", err)
```

This makes errors traceable without a stack trace library.

### Specific error cases

**HTTP 429 from YouTube search**: Defined as `provider.ErrRateLimited`. Caught in Update, shows a specific toast with backoff advice.

**yt-dlp `This video is not available`**: Caught by string-matching stderr. Show toast "Track unavailable". Do not retry.

**yt-dlp `This video is age-restricted`**: Same pattern. Show toast "Age-restricted track — cannot play without login".

**yt-dlp timeout (30s)**: Context deadline exceeded. Show toast "Track took too long to load. Try again."

**ffmpeg EOF before duration**: Track ended early, possibly due to an expired URL (6-hour expiry). Re-extract the URL and seek to the last known position, then replay. Detect this by comparing `p.position` to `p.duration` when `TrackEndedMsg` arrives — if `position < duration * 0.95`, it was an early termination, not a natural end.

**bbolt open failure**: Delete the database and reopen. Log the incident. The audio files are unaffected.

**oto initialization failure**: Audio device unavailable. This is fatal — print an error and exit. Common causes: PulseAudio/PipeWire not running, another exclusive-access process holding the audio device.

---

## 14. Signal Handling and Clean Shutdown

A clean shutdown is important because:
- The ffmpeg process must be killed, not orphaned
- The oto audio device must be closed
- The bbolt database must be flushed
- The MPRIS service must be unregistered from D-Bus
- The partial cache directory must be cleaned

### In `cmd/gotune/main.go`

```go
func main() {
    if err := checkDependencies(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintln(os.Stderr, "Config error:", err)
        os.Exit(1)
    }

    // Initialize services
    prov  := ytmusic.New(cfg)
    ext   := ytdlp.New(cfg)
    cache, err := cache.New(cfg)
    if err != nil {
        fmt.Fprintln(os.Stderr, "Cache error:", err)
        os.Exit(1)
    }
    speaker, err := player.NewSpeaker(cfg.FFmpeg.SampleRate, cfg.Playback.BufferSizeKB)
    if err != nil {
        fmt.Fprintln(os.Stderr, "Audio device error:", err)
        os.Exit(1)
    }
    pl := player.New(cfg, speaker)

    // Build root model
    model := ui.NewModel(prov, ext, pl, cache, cfg)

    // Start bubbletea
    prog := tea.NewProgram(model,
        tea.WithAltScreen(),       // full-screen TUI
        tea.WithMouseCellMotion(), // optional mouse support
    )

    // Register MPRIS2 (optional)
    mprisSvc, err := mpris.Register(prog)
    if err != nil {
        log.Printf("MPRIS2 unavailable: %v", err)
        mprisSvc = mpris.NewNoop()
    }
    model = model.WithMPRIS(mprisSvc)

    // Handle OS signals in a separate goroutine
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigCh
        prog.Send(ui.QuitMsg{})
    }()

    // Run — blocks until quit
    if _, err := prog.Run(); err != nil {
        fmt.Fprintln(os.Stderr, "TUI error:", err)
    }

    // Cleanup (runs after prog.Run returns)
    pl.Stop()
    speaker.Close()
    mprisSvc.Close()
    cache.Close() // closes bbolt
}
```

`tea.WithAltScreen()` is important. It activates the terminal's alternate screen buffer, which means the TUI occupies the full terminal and is fully erased when the program exits — leaving the shell in a clean state.

---

## 15. Build System

### Makefile

```makefile
BINARY   := gotune
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-X main.version=$(VERSION) -s -w"

.PHONY: build run test lint clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/gotune

run: build
	./$(BINARY)

test:
	go test ./... -v -race -count=1

test-unit:
	go test ./... -v -race -count=1 -short

lint:
	go vet ./...
	staticcheck ./...  # go install honnef.co/go/tools/cmd/staticcheck@latest

clean:
	rm -f $(BINARY)
	go clean ./...

install: build
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
```

### Version flag

```go
// cmd/gotune/main.go
var version = "dev"

func main() {
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "--version", "-v":
            fmt.Printf("gotune %s\n", version)
            os.Exit(0)
        case "--config":
            fmt.Println(config.FilePath())
            os.Exit(0)
        case "--cache-dir":
            cfg, _ := config.Load()
            fmt.Println(cfg.Cache.ResolvedDir())
            os.Exit(0)
        }
    }
    // ... rest of main
}
```

---

## 16. Testing Strategy

### Unit tests

#### `provider/ytmusic_test.go`

Save a real search response to `testdata/search_response.json` by running:

```bash
curl -s -X POST \
  "https://music.youtube.com/youtubei/v1/search?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE" \
  -H "Content-Type: application/json" \
  -H "X-YouTube-Client-Name: 67" \
  -H "X-YouTube-Client-Version: 1.20231204.01.00" \
  -d '{"context":{"client":{"clientName":"WEB_REMIX","clientVersion":"1.20231204.01.00","hl":"en","gl":"US"}},"query":"test","params":"EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D"}' \
  > testdata/search_response.json
```

Then in the test:

```go
func TestParseSearchResponse(t *testing.T) {
    data, err := os.ReadFile("testdata/search_response.json")
    require.NoError(t, err)

    results, err := parseSearchResponse(data)
    require.NoError(t, err)
    require.NotEmpty(t, results)

    // Verify structure of first result
    first := results[0]
    assert.NotEmpty(t, first.VideoID, "VideoID should not be empty")
    assert.NotEmpty(t, first.Title,   "Title should not be empty")
    assert.NotEmpty(t, first.Artist,  "Artist should not be empty")
    assert.Greater(t,  first.Duration, 0, "Duration should be positive")
}
```

No HTTP calls in unit tests.

#### `cache/cache_test.go`

```go
func TestCacheGetMissOnPartialFile(t *testing.T) {
    c := newTestCache(t)
    // Write a file to partial dir, not audio dir
    partPath := filepath.Join(c.partialDir, "testvideo.part")
    os.WriteFile(partPath, []byte("incomplete data"), 0644)
    // Should not be a cache hit
    _, hit := c.Get("testvideo")
    assert.False(t, hit)
}

func TestLRUEviction(t *testing.T) {
    c := newTestCache(t)
    c.maxBytes = 100 // very small limit

    // Add two entries totaling > 100 bytes
    addFakeEntry(t, c, "track1", 60, time.Now().Add(-2*time.Hour))
    addFakeEntry(t, c, "track2", 60, time.Now().Add(-1*time.Hour))

    c.maybeEvict()

    // track1 (older) should be evicted
    _, hit1 := c.Get("track1")
    _, hit2 := c.Get("track2")
    assert.False(t, hit1, "track1 should be evicted (LRU)")
    assert.True(t,  hit2, "track2 should remain")
}

func TestStaleIndexRepair(t *testing.T) {
    c := newTestCache(t)
    // Write an index entry pointing to a non-existent file
    c.index.Put(&Entry{
        VideoID:      "missing",
        FilePath:     "/tmp/does-not-exist.webm",
        LastAccessed: time.Now(),
    })
    _, hit := c.Get("missing")
    assert.False(t, hit, "stale entry should not be a hit")
    // Index should be cleaned up
    _, err := c.index.Get("missing")
    assert.ErrorIs(t, err, ErrNotFound)
}
```

#### `player/player_test.go`

Test volume scaling:

```go
func TestVolumeScaling(t *testing.T) {
    // A maximum-amplitude sample at 50% volume should be half amplitude
    buf := make([]byte, 2)
    buf[0] = 0xFF // 0x7FFF = 32767 in little-endian signed
    buf[1] = 0x7F
    applyVolume(buf, 0.5)
    result := int16(buf[0]) | int16(buf[1])<<8
    assert.InDelta(t, 16383, int(result), 2) // 32767 * 0.5 = 16383
}
```

### Integration tests

Gate with build tag `integration`. These make real network calls and require `yt-dlp` and `ffmpeg`:

```go
//go:build integration

func TestRealSearch(t *testing.T) {
    prov := ytmusic.New(config.Defaults())
    results, err := prov.Search("never gonna give you up")
    require.NoError(t, err)
    require.NotEmpty(t, results)
    // Rick Astley should appear somewhere in results
    found := slices.ContainsFunc(results, func(r provider.Track) bool {
        return strings.Contains(strings.ToLower(r.Artist), "rick astley")
    })
    assert.True(t, found)
}

func TestRealExtraction(t *testing.T) {
    ext := ytdlp.New(config.Defaults())
    // Use a known stable video ID
    info, err := ext.Extract("dQw4w9WgXcQ")
    require.NoError(t, err)
    assert.NotEmpty(t, info.URL)
    assert.Greater(t, info.Duration, 0.0)
    assert.Contains(t, []string{"webm", "m4a"}, info.Format)
}
```

Run:

```bash
go test -tags integration ./... -v -run TestReal
```

---

## 17. Phase-by-Phase Execution Plan

### Phase 0: Spike — Prove Audio Pipeline (1 day)

Before writing any structure, create a single throwaway `spike/main.go`:

```go
package main

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"
    "github.com/hajimehoshi/oto/v2"
)

func main() {
    videoID := "dQw4w9WgXcQ" // hardcoded

    // Step 1: Get stream URL from yt-dlp
    out, err := exec.Command("yt-dlp",
        "--no-playlist", "--format", "bestaudio[ext=webm]/bestaudio",
        "--print", "%(url)s", "https://youtube.com/watch?v="+videoID,
    ).Output()
    if err != nil { panic(err) }
    streamURL := strings.TrimSpace(string(out))
    fmt.Println("URL:", streamURL[:60], "...")

    // Step 2: Start ffmpeg → PCM pipe
    ffmpeg := exec.Command("ffmpeg",
        "-hide_banner", "-loglevel", "error",
        "-i", streamURL,
        "-vn", "-f", "s16le", "-ar", "44100", "-ac", "2", "pipe:1",
    )
    pipe, _ := ffmpeg.StdoutPipe()
    ffmpeg.Stderr = os.Stderr
    ffmpeg.Start()

    // Step 3: Play through oto
    ctx, readyCh, err := oto.NewContext(44100, 2, 2)
    if err != nil { panic(err) }
    <-readyCh
    player := ctx.NewPlayer(pipe)
    player.Play()

    fmt.Println("Playing for 10 seconds...")
    time.Sleep(10 * time.Second)

    player.Close()
    ffmpeg.Process.Kill()
}
```

Run: `cd spike && go mod init spike && go get github.com/hajimehoshi/oto/v2 && go run main.go`

If you hear audio: proceed. If not, debug here. The only variable is whether `yt-dlp` and `ffmpeg` are correctly installed and the stream URL is valid.

Delete the `spike/` directory after success.

### Phase 1: Config and Structure (1 day)

1. `go mod init` the main project. Add all dependencies.
2. Create the full directory structure (all files can be empty stubs with correct package declarations).
3. Write `internal/config/config.go` completely. Write a `TestLoad` that verifies defaults are applied and the file is created.
4. Write `cmd/gotune/main.go` with just `checkDependencies()` and a config load. Verify it runs and creates `~/.config/gotune/config.toml`.

### Phase 2: Provider (2 days)

1. Implement `internal/provider/ytmusic.go` fully.
2. Test manually: write a temporary `main()` that calls `Search("test artist")` and prints results.
3. Write unit test using the saved `testdata/search_response.json` fixture.
4. Implement `GetRadio()`.
5. Verify parsing handles edge cases: podcast episodes, live streams (these should be skipped, `videoID` will be empty for non-standard items).

### Phase 3: Extractor (1 day)

1. Implement `internal/extractor/ytdlp.go`.
2. Test manually by extracting a URL for a video ID from the Phase 2 results.
3. Verify the format string, duration parsing, and timeout behavior.

### Phase 4: Player (2 days)

1. Implement `internal/player/output.go` (Speaker).
2. Implement `internal/player/ffmpeg.go`: `Play()`, `Read()`, `Pause()`, `Resume()`, `Seek()`, `SetVolume()`, `Stop()`.
3. Test manually: write a `main()` that chains provider → extractor → player and plays a track interactively with keyboard input via `bufio.Scanner` on stdin (pause on 'p', seek on 's', quit on 'q'). This validates all player functions before the TUI exists.
4. Write unit test for `applyVolume`.

### Phase 5: Cache (2 days)

1. Implement `internal/cache/index.go` and `internal/cache/cache.go`.
2. Write all unit tests for the cache layer.
3. Integrate into the Phase 4 `main()`: after a track plays, call `DownloadAsync`. On second play of the same track, verify it plays from the local file.
4. Test LRU eviction manually by setting `MaxSizeMB: 1` in the config and playing several long tracks.

### Phase 6: TUI (4 days)

1. Write `styles.go` and `keys.go` completely.
2. Write the root `model.go`: struct definition, all message types, empty `Update` and `View`.
3. Write `search.go`: input + result list, wire to provider.
4. Wire the full play flow: search → select → extract → play. Test it. At this point you have a functional (if ugly) TUI.
5. Write `playerbar.go`: progress bar, track info, volume indicator.
6. Write `queue.go`: append, remove, next.
7. Wire radio: when queue empties after `TrackEndedMsg`, call `GetRadio`.
8. Write `toast.go`.
9. Write `cacheview.go`.
10. Handle `tea.WindowSizeMsg` everywhere — test by resizing the terminal while playing.

### Phase 7: MPRIS2 (1 day)

1. Implement `internal/mpris/mpris.go` and `properties.go`.
2. Integrate into `main.go`.
3. Test: play a track, press the keyboard media key (Fn+F8 or dedicated key), verify playback pauses. Check GNOME's or KDE's media indicator shows the track title.

### Phase 8: Hardening (2 days)

1. Run `go test -race ./...`. Fix all races.
2. Test the expired URL scenario: extract a URL, wait (or modify the URL to be invalid), attempt playback, verify error handling.
3. Test killing ffmpeg externally while playing: `kill $(pgrep ffmpeg)`. Verify the application shows a toast and handles the `TrackEndedMsg`.
4. Test `yt-dlp -U` update flow — verify a version bump doesn't break the `--print` format string.
5. Run `go vet ./...` and `staticcheck ./...`. Fix all issues.
6. Final manual test: search → play → pause → seek → skip → queue multiple tracks → cache verify → clear cache → quit cleanly.

---

## 18. Known Failure Modes and Mitigations

### YouTube API payload change (search parser breaks)

**Symptom**: Search returns 0 results with no error, or a JSON parse error.

**Diagnosis**: Open DevTools in Chrome, go to music.youtube.com, search for something, inspect the POST request to `/youtubei/v1/search` and compare the response structure to your parser.

**Fix**: Update the field paths in `parseTrack()`. The `dig()` helper makes this mechanical — find the new path, update the string keys.

**Timeline**: This has happened approximately 3-4 times per year to the Python `ytmusicapi` project, which documents the same API.

### yt-dlp breaks on YouTube signature change

**Symptom**: `yt-dlp` exits with "ERROR: ... cipher" or "Sign in to confirm" errors.

**Fix**: `yt-dlp -U`. This self-updates yt-dlp to the latest version which patches the cipher.

**User action**: Document `yt-dlp -U` in the README as the first troubleshooting step.

### Expired stream URL (ffmpeg 403 mid-stream)

**Symptom**: Track stops after several hours without reaching EOF. `TrackEndedMsg` arrives with `position < duration * 0.95`.

**Detection in Update**:
```go
case TrackEndedMsg:
    if m.position < m.duration * 0.95 && m.duration > 30 {
        // Premature end — re-extract and seek
        return m, tea.Batch(
            extractCmd(m.ext, m.currentTrack),
            toastCmd("Reconnecting...", false),
        )
    }
    // Natural end
    return m, m.playNextInQueue()
```

**On StreamReadyMsg after re-extraction**: use `Seek(m.position)` on the player before playback begins.

### oto audio device unavailable

**Symptom**: `oto.NewContext` returns an error like "failed to initialize audio driver".

**Common causes**: PipeWire/PulseAudio not running, another application holding exclusive access.

**Fix**: This is fatal. Print a clear error message explaining that the audio device is unavailable and suggest checking PulseAudio/PipeWire status:

```
gotune: could not open audio device.
Is PulseAudio or PipeWire running? Try: systemctl --user status pipewire
```

### bbolt lock timeout

**Symptom**: `bolt.Open` returns "timeout" error if another gotune instance is running (bbolt is single-writer).

**Fix**: The `bolt.Options{Timeout: 1 * time.Second}` in `openIndex` causes it to fail fast rather than hang. Show a clear error:

```
Another instance of gotune may be running.
If not, delete ~/.cache/gotune/index.db and retry.
```

### Terminal too small

**Symptom**: Garbled rendering when the terminal width is below ~60 columns or height below ~10 rows.

**Fix**: In `View()`, if `m.width < 60 || m.height < 10`, render a single message:

```go
if m.width < 60 || m.height < 10 {
    return "Terminal too small. Please resize to at least 60x10."
}
```

### Race between cache DownloadAsync and application exit

**Symptom**: A background `DownloadAsync` goroutine is running when the user presses `q`. The bbolt database is closed while the goroutine is trying to write to it.

**Fix**: Track in-flight downloads with a `sync.WaitGroup`:

```go
type Cache struct {
    // ...
    wg sync.WaitGroup
}

func (c *Cache) DownloadAsync(videoID string, format string) {
    c.wg.Add(1)
    go func() {
        defer c.wg.Done()
        // ... download logic
    }()
}

func (c *Cache) Close() error {
    c.wg.Wait() // let in-flight downloads finish
    return c.index.db.Close()
}
```

`cache.Close()` is called in `main.go` after `prog.Run()` returns. The `wg.Wait()` here gives in-flight downloads up to a few seconds to complete before the database closes.

---

*End of plan.*
*Approximate final size: ~3,500 lines of Go across all packages, not counting tests.*
*All packages: internal only, no public API surface.*
*Single binary. Two runtime dependencies: ffmpeg, yt-dlp.*