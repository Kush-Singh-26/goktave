# GoTune: Engineering Plan

> Platform: Fedora Linux only.
> Audio backend: ffmpeg (already installed) + oto.
> Stream resolution: InnerTube API (direct Go HTTP) with Piped API fallback.
> Search: YouTube Music InnerTube (WEB_REMIX client).
> Token refresh: automatic, sourced from yt-dlp upstream.
> Media keys: MPRIS2 over D-Bus.
> No yt-dlp runtime dependency.

---

## Table of Contents

1. [Project Overview and Decisions](#1-project-overview-and-decisions)
2. [System Dependencies](#2-system-dependencies)
3. [Repository Structure](#3-repository-structure)
4. [Module Initialization and Go Dependencies](#4-module-initialization-and-go-dependencies)
5. [The Config Layer](#5-the-config-layer)
6. [The Source Layer — Search + Stream Resolution](#6-the-source-layer)
7. [The Token Layer — Automatic InnerTube Token Refresh](#7-the-token-layer)
8. [The Player Layer — ffmpeg + oto](#8-the-player-layer)
9. [The Queue Layer](#9-the-queue-layer)
10. [The Cache Layer — bbolt + LRU](#10-the-cache-layer)
11. [The MPRIS2 Layer — Media Keys via D-Bus](#11-the-mpris2-layer)
12. [The TUI Layer — Bubbletea](#12-the-tui-layer)
13. [Concurrency Model](#13-concurrency-model)
14. [Error Handling](#14-error-handling)
15. [Signal Handling and Clean Shutdown](#15-signal-handling-and-clean-shutdown)
16. [Build System](#16-build-system)
17. [Testing Strategy](#17-testing-strategy)
18. [Phase-by-Phase Execution Plan](#18-phase-by-phase-execution-plan)
19. [Known Failure Modes and Mitigations](#19-known-failure-modes-and-mitigations)

---

## 1. Project Overview and Decisions

### What it is

A terminal-based music player that streams audio from YouTube Music. Single compiled binary. No Electron. No ads. No YouTube Premium required. No Python runtime dependency.

### Finalized decisions

| Concern | Decision | Reason |
|---|---|---|
| Language | Go 1.22+ | Chosen |
| OS | Fedora Linux only | No Windows/macOS conditionals |
| TUI | bubbletea + lipgloss + bubbles | Standard for Go TUIs |
| Audio decoding | ffmpeg subprocess → PCM pipe | Handles every codec, already installed |
| Audio output | oto v3 | Low-level PCM, PulseAudio/PipeWire/ALSA |
| Stream resolution | InnerTube ANDROID_MUSIC client (direct Go HTTP) | ~150–250ms, no subprocess, no Python |
| Stream fallback | Piped public API | Free fallback if InnerTube tokens rotate |
| Token refresh | Auto-fetched from yt-dlp source on GitHub | Tokens stay current without manual updates |
| YT Music search | InnerTube WEB_REMIX client (direct Go HTTP) | Same API, no third-party library |
| Queue | Dedicated `internal/queue` package | Business logic separated from UI |
| Cache index | bbolt | Crash-safe, no CGO, no daemon |
| Config | TOML via BurntSushi/toml | Human-editable |
| Media keys | MPRIS2 over D-Bus via godbus | Linux standard, GNOME/KDE/etc |

### What this will not do

- No Windows or macOS support.
- No YouTube Premium features.
- No lyrics.
- No album art in terminal (V1).
- No offline-only mode.
- No yt-dlp runtime dependency (used only as a token reference source via HTTP, not executed).

### Architecture overview

```
Search query
    ↓
source.YouTube (WEB_REMIX InnerTube POST)
    ↓ []Track
queue.Queue
    ↓ Track
source.YouTube.StreamURL (ANDROID_MUSIC InnerTube POST)
    → on failure: source.Piped.StreamURL
    ↓ url string
player.FFmpegPlayer (ffmpeg subprocess → PCM pipe → oto)
    ↓ audio
PipeWire/PulseAudio → speakers
```

### Maintenance reality

The InnerTube ANDROID_MUSIC client version and API key change occasionally. The token auto-refresh system (Section 7) handles this automatically by reading from yt-dlp's source on GitHub. The WEB_REMIX search client version changes independently — it is configured in `config.toml` and in the token refresh path. The search response JSON shape can change without notice; this is the most fragile part and requires a human to inspect DevTools traffic when it breaks.

---

## 2. System Dependencies

### Required at runtime

**ffmpeg** — decodes audio stream to raw PCM, pipes to Go process.

```bash
sudo dnf install ffmpeg
```

If restricted on your Fedora version, enable RPM Fusion first:

```bash
sudo dnf install \
  https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm \
  https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm
sudo dnf install ffmpeg
```

That is the only required runtime binary. yt-dlp is **not** required at runtime.

### Required system libraries (for oto audio output)

oto on Linux outputs to PipeWire (via PulseAudio compatibility) or ALSA. Standard Fedora desktop has PipeWire active by default. On a minimal install:

```bash
sudo dnf install pulseaudio-libs-devel alsa-lib-devel
```

oto uses CGO to bind to the audio library. You need a C compiler:

```bash
sudo dnf install gcc
```

This is the only CGO dependency in the entire project.

### D-Bus (for MPRIS2)

Present on any Fedora desktop. No installation needed. `godbus` is pure Go.

### Startup dependency check

At startup, before initializing anything else:

```go
func checkDependencies() error {
    if _, err := exec.LookPath("ffmpeg"); err != nil {
        return fmt.Errorf(
            "ffmpeg not found in PATH.\n" +
            "Install with: sudo dnf install ffmpeg\n" +
            "(Enable RPM Fusion first if ffmpeg is unavailable)",
        )
    }
    return nil
}
```

Print to stderr and exit code 1 if missing. Do not start the TUI.

---

## 3. Repository Structure

```
gotune/
├── cmd/
│   └── gotune/
│       └── main.go                  # Entrypoint
├── internal/
│   ├── config/
│   │   └── config.go                # TOML config load/save/defaults
│   ├── source/
│   │   ├── source.go                # Track type + Source interface
│   │   ├── youtube.go               # InnerTube search + stream resolution
│   │   ├── piped.go                 # Piped API fallback for stream resolution
│   │   └── youtube_test.go
│   ├── tokens/
│   │   ├── tokens.go                # InnerTubeTokens type + LoadTokens
│   │   └── tokens_test.go
│   ├── player/
│   │   ├── player.go                # FFmpegPlayer struct + Play/Pause/Seek/Volume
│   │   ├── decoder.go               # ffmpeg subprocess → PCM pipe
│   │   ├── speaker.go               # oto singleton, owns context lifetime
│   │   └── player_test.go
│   ├── queue/
│   │   ├── queue.go                 # Enqueue/Next/Prev/Shuffle/Current
│   │   └── queue_test.go
│   ├── cache/
│   │   ├── cache.go                 # Cache struct + Get/DownloadAsync/Evict
│   │   ├── index.go                 # bbolt-backed metadata index
│   │   └── cache_test.go
│   ├── mpris/
│   │   ├── mpris.go                 # MPRIS2 D-Bus service  [//go:build linux]
│   │   └── properties.go            # Property get/set + PropertiesChanged emit
│   └── ui/
│       ├── model.go                 # Root bubbletea model — Init/Update/View only
│       ├── msgs.go                  # All tea.Msg type definitions
│       ├── search.go                # Search sub-model
│       ├── queue.go                 # Queue view sub-model
│       ├── playerbar.go             # Bottom player bar component
│       ├── cacheview.go             # Cache manager overlay
│       ├── toast.go                 # Temporary notification component
│       ├── keys.go                  # All keybinding definitions
│       ├── styles.go                # All lipgloss styles
│       └── helpers.go               # truncate, formatSeconds, formatBytes
├── testdata/
│   └── search_response.json         # Saved real API response for unit tests
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

All packages are under `internal/`. Nothing is exported as a public library.

**Dependency direction** (strict, no cycles):

```
config → (nothing internal)
tokens → config
source → config, tokens
queue  → source
player → config
cache  → config, source
mpris  → source (for Track type)
ui     → source, queue, player, cache, mpris, config
cmd    → ui, config (only)
```

Nothing imports `ui` except `cmd`. Nothing imports `player` except `ui` and `cmd`.

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
go get github.com/ebitengine/oto/v3@latest
go get go.etcd.io/bbolt@latest
go get github.com/BurntSushi/toml@latest
go get github.com/godbus/dbus/v5@latest
```

What each does:

- `bubbletea`: Elm-Architecture TUI event loop. Model/Update/View.
- `lipgloss`: Terminal styling. Colors, borders, padding, layout.
- `bubbles`: Pre-built bubbletea components. We use `textinput` and `progress`.
- `oto/v3`: Low-level PCM audio output for Linux. CGO required.
- `bbolt`: Embedded B-tree key-value store for cache index. Pure Go, no daemon.
- `BurntSushi/toml`: TOML config encoding/decoding.
- `godbus/dbus/v5`: D-Bus client for MPRIS2. Pure Go.

**No other dependencies.** Do not add a YAML library, a logging framework, a CLI flag library beyond stdlib `flag`, or any HTTP client beyond stdlib `net/http`.

Note: the original plan used `oto/v2`. This plan uses **oto/v3** throughout. The API differs:
- v3 `NewContext` takes `*oto.NewContextOptions` struct instead of positional args.
- v3 `NewContextOptions` has a `BufferSizeInBytes` field.
- Everything else (Player, Play, Pause, Close) is the same.

---

## 5. The Config Layer

### File: `internal/config/config.go`

Config file path: `~/.config/gotune/config.toml`

Created automatically on first run with defaults if it does not exist.

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/BurntSushi/toml"
)

type Config struct {
    Cache      CacheConfig      `toml:"cache"`
    Playback   PlaybackConfig   `toml:"playback"`
    FFmpeg     FFmpegConfig     `toml:"ffmpeg"`
    InnerTube  InnerTubeConfig  `toml:"innertube"`
    Piped      PipedConfig      `toml:"piped"`
    Appearance AppearanceConfig `toml:"appearance"`
}

type CacheConfig struct {
    MaxSizeMB int    `toml:"max_size_mb"` // default: 2048
    Directory string `toml:"directory"`   // default: "" → ~/.cache/gotune
}

type PlaybackConfig struct {
    Volume           int  `toml:"volume"`             // default: 80, range 0–100
    BufferSizeInBytes int `toml:"buffer_size_bytes"`  // default: 4096
}

type FFmpegConfig struct {
    Path       string `toml:"path"`        // default: "ffmpeg"
    Threads    int    `toml:"threads"`     // default: 1
    SampleRate int    `toml:"sample_rate"` // default: 44100
}

// InnerTubeConfig holds the ANDROID_MUSIC client identity used for stream
// resolution. These values are auto-refreshed by the tokens package, but
// can be overridden here if auto-refresh ever breaks.
type InnerTubeConfig struct {
    ClientName    string `toml:"client_name"`    // default: "ANDROID_MUSIC"
    ClientVersion string `toml:"client_version"` // default: "5.28.1"
    APIKey        string `toml:"api_key"`         // default: "AIzaSyA..."
}

// PipedConfig configures the Piped API fallback for stream resolution.
type PipedConfig struct {
    InstanceURL string `toml:"instance_url"` // default: "https://pipedapi.kavin.rocks"
    Enabled     bool   `toml:"enabled"`       // default: true
}

type AppearanceConfig struct {
    AccentColor string `toml:"accent_color"` // default: "#e94560"
}

func defaults() *Config {
    return &Config{
        Cache: CacheConfig{MaxSizeMB: 2048},
        Playback: PlaybackConfig{
            Volume:            80,
            BufferSizeInBytes: 4096,
        },
        FFmpeg: FFmpegConfig{
            Path:       "ffmpeg",
            Threads:    1,
            SampleRate: 44100,
        },
        InnerTube: InnerTubeConfig{
            ClientName:    "ANDROID_MUSIC",
            ClientVersion: "5.28.1",
            APIKey:        "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8",
        },
        Piped: PipedConfig{
            InstanceURL: "https://pipedapi.kavin.rocks",
            Enabled:     true,
        },
        Appearance: AppearanceConfig{AccentColor: "#e94560"},
    }
}

func Load() (*Config, error) {
    cfg := defaults()
    path := FilePath()

    if _, err := os.Stat(path); os.IsNotExist(err) {
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
            return cfg, nil
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

func FilePath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "gotune", "config.toml")
}

func (c *CacheConfig) ResolvedDir() string {
    if c.Directory != "" {
        return c.Directory
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".cache", "gotune")
}
```

---

## 6. The Source Layer

### Purpose

Two responsibilities in one package:
1. **Search** — query YouTube Music, return structured track metadata.
2. **StreamURL** — given a video ID, return a directly playable HTTPS URL.

These are unified because every provider we will ever support needs both. Splitting them added interface indirection with no benefit.

### File: `internal/source/source.go`

```go
package source

import "context"

// Track is the canonical track type used throughout the application.
type Track struct {
    VideoID  string
    Title    string
    Artist   string
    Album    string
    Duration int    // seconds, 0 if unknown
    ThumbURL string
}

// Source can search for tracks and resolve stream URLs.
type Source interface {
    Search(ctx context.Context, query string) ([]Track, error)
    StreamURL(ctx context.Context, videoID string) (string, error)
    Radio(ctx context.Context, seedVideoID string) ([]Track, error)
}

var ErrRateLimited = errors.New("rate limited by YouTube")
var ErrNoStream    = errors.New("no audio stream found for video")
```

### File: `internal/source/youtube.go`

#### The YouTube struct

```go
type YouTube struct {
    tokens  *tokens.InnerTubeTokens
    http    *http.Client
    cfg     *config.Config
    piped   *Piped // fallback, nil if disabled
}

func NewYouTube(cfg *config.Config, tok *tokens.InnerTubeTokens) *YouTube {
    var piped *Piped
    if cfg.Piped.Enabled {
        piped = NewPiped(cfg)
    }
    return &YouTube{
        tokens: tok,
        http:   &http.Client{Timeout: 15 * time.Second},
        cfg:    cfg,
        piped:  piped,
    }
}
```

#### Search — WEB_REMIX client

Search uses the `WEB_REMIX` InnerTube client (client name `67`). This is separate from stream resolution and uses different, more stable tokens — the `X-YouTube-Client-Version` header for search has not required updates as frequently as the ANDROID_MUSIC stream tokens.

Endpoint:
```
POST https://music.youtube.com/youtubei/v1/search?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE&prettyPrint=false
```

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

Request body:
```go
type searchRequest struct {
    Context ytContext `json:"context"`
    Query   string    `json:"query"`
    Params  string    `json:"params"`
}

// Songs filter — base64-encoded protobuf
const searchParamsSongs = "EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D"
```

Response parsing uses `map[string]interface{}` with a safe recursive `dig` accessor. Do not use typed structs for the response — too many optional fields.

```go
func dig(v interface{}, keys ...string) interface{} {
    for _, k := range keys {
        switch node := v.(type) {
        case map[string]interface{}:
            val, ok := node[k]
            if !ok { return nil }
            v = val
        case []interface{}:
            idx, err := strconv.Atoi(k)
            if err != nil || idx < 0 || idx >= len(node) { return nil }
            v = node[idx]
        default:
            return nil
        }
    }
    return v
}

func digStr(v interface{}, keys ...string) string {
    r := dig(v, keys...)
    if r == nil { return "" }
    s, _ := r.(string)
    return s
}
```

Path to track renderers:
```
root
  .contents.tabbedSearchResultsRenderer
  .tabs[0].tabRenderer.content
  .sectionListRenderer.contents
    [N].musicShelfRenderer.contents
      [M].musicResponsiveListItemRenderer  ← one track
```

Parse each renderer with an explicit `parseTrack` function. Return `nil` for items without a `videoId` (podcasts, ad units, etc.).

Duration strings arrive as `"3:45"`, `"1:03:22"`, `"0:45"`:

```go
func parseDuration(s string) int {
    if s == "" { return 0 }
    parts := strings.Split(s, ":")
    total := 0
    for _, p := range parts {
        n, err := strconv.Atoi(strings.TrimSpace(p))
        if err != nil { return 0 }
        total = total*60 + n
    }
    return total
}
```

#### StreamURL — ANDROID_MUSIC client

The ANDROID_MUSIC client returns **pre-signed URLs** that ffmpeg can open directly. No JS cipher deciphering needed. This is the key advantage over using yt-dlp.

```go
const innerTubePlayerURL = "https://www.youtube.com/youtubei/v1/player"

func (yt *YouTube) StreamURL(ctx context.Context, videoID string) (string, error) {
    body, _ := json.Marshal(map[string]interface{}{
        "videoId": videoID,
        "context": map[string]interface{}{
            "client": map[string]interface{}{
                "clientName":    yt.tokens.ClientName,
                "clientVersion": yt.tokens.ClientVersion,
                "hl":            "en",
            },
        },
    })

    reqURL := innerTubePlayerURL + "?key=" + yt.tokens.APIKey
    req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", yt.tokens.UserAgent())

    resp, err := yt.http.Do(req)
    if err != nil {
        return "", fmt.Errorf("innertube: request failed: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        StreamingData struct {
            AdaptiveFormats []struct {
                URL      string `json:"url"`
                MimeType string `json:"mimeType"`
                Bitrate  int    `json:"bitrate"`
            } `json:"adaptiveFormats"`
        } `json:"streamingData"`
        PlayabilityStatus struct {
            Status string `json:"status"`
            Reason string `json:"reason"`
        } `json:"playabilityStatus"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("innertube: decode failed: %w", err)
    }

    if result.PlayabilityStatus.Status == "ERROR" ||
       result.PlayabilityStatus.Status == "LOGIN_REQUIRED" {
        // Tokens likely rotated — fall through to Piped fallback
        return "", fmt.Errorf("innertube: %s: %s",
            result.PlayabilityStatus.Status,
            result.PlayabilityStatus.Reason)
    }

    url := pickBestAudio(result.StreamingData.AdaptiveFormats)
    if url != "" {
        return url, nil
    }

    // Fallback to Piped if InnerTube returned no streams
    if yt.piped != nil {
        return yt.piped.StreamURL(ctx, videoID)
    }

    return "", ErrNoStream
}

func pickBestAudio(formats []struct {
    URL      string `json:"url"`
    MimeType string `json:"mimeType"`
    Bitrate  int    `json:"bitrate"`
}) string {
    var bestURL string
    var bestBitrate int
    for _, f := range formats {
        if !strings.Contains(f.MimeType, "audio") {
            continue
        }
        if f.Bitrate > bestBitrate {
            bestBitrate = f.Bitrate
            bestURL = f.URL
        }
    }
    return bestURL
}
```

#### Radio

Uses the `next` InnerTube endpoint to fetch a continuous radio seeded by a video ID:

```
POST https://music.youtube.com/youtubei/v1/next?key=...&prettyPrint=false
```

Body includes `"videoId"` and `"isAudioOnly": true`. Returns the first 25 tracks, skipping index 0 (the seed track itself).

Response path:
```
root
  .contents.singleColumnMusicWatchNextResultsRenderer
  .tabbedRenderer.watchNextTabbedResultsRenderer
  .tabs[0].tabRenderer.content
  .musicQueueRenderer.content
  .playlistPanelRenderer.contents
    [N].playlistPanelVideoRenderer
      .videoId / .title.runs[0].text / .longBylineText.runs[0].text / .lengthText.runs[0].text
```

### File: `internal/source/piped.go`

The Piped fallback only needs `StreamURL` — search always uses InnerTube WEB_REMIX.

```go
type Piped struct {
    http        *http.Client
    instanceURL string
}

func NewPiped(cfg *config.Config) *Piped {
    return &Piped{
        http:        &http.Client{Timeout: 10 * time.Second},
        instanceURL: cfg.Piped.InstanceURL,
    }
}

func (p *Piped) StreamURL(ctx context.Context, videoID string) (string, error) {
    url := p.instanceURL + "/streams/" + videoID
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return "", err
    }

    resp, err := p.http.Do(req)
    if err != nil {
        return "", fmt.Errorf("piped: request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("piped: HTTP %d", resp.StatusCode)
    }

    var result struct {
        AudioStreams []struct {
            URL     string `json:"url"`
            Bitrate int    `json:"bitrate"`
            Format  string `json:"format"`
        } `json:"audioStreams"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("piped: decode failed: %w", err)
    }

    // Prefer opus/webm, then pick highest bitrate
    var bestURL string
    var bestBitrate int
    for _, s := range result.AudioStreams {
        if s.Bitrate > bestBitrate {
            bestBitrate = s.Bitrate
            bestURL = s.URL
        }
    }
    if bestURL == "" {
        return "", ErrNoStream
    }
    return bestURL, nil
}
```

---

## 7. The Token Layer

### Purpose

Automatically keep InnerTube ANDROID_MUSIC client tokens current. Tokens are fetched from yt-dlp's source on GitHub (the authoritative reference), cached to disk, and refreshed in the background when stale. Falls back to hardcoded values if network is unavailable.

yt-dlp maintainers update their token values within hours of YouTube rotating them. Reading from their source means gotune stays current without manual intervention.

### File: `internal/tokens/tokens.go`

```go
package tokens

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"
)

const (
    ytdlpSourceURL = "https://raw.githubusercontent.com/yt-dlp/yt-dlp/master/yt_dlp/extractor/youtube.py"

    // Fallback values — used when both disk cache and network are unavailable.
    // Update these manually if the auto-refresh regex ever breaks.
    fallbackClientName    = "ANDROID_MUSIC"
    fallbackClientVersion = "5.28.1"
    fallbackAPIKey        = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
)

type InnerTubeTokens struct {
    ClientName    string    `json:"client_name"`
    ClientVersion string    `json:"client_version"`
    APIKey        string    `json:"api_key"`
    FetchedAt     time.Time `json:"fetched_at"`
}

func (t *InnerTubeTokens) UserAgent() string {
    return fmt.Sprintf(
        "com.google.android.apps.youtube.music/%s (Linux; U; Android 11)",
        t.ClientVersion,
    )
}

var (
    // These regexes match the ANDROID_MUSIC block in yt-dlp's youtube.py.
    // The structure has been stable for years; update if it ever changes.
    reClientVersion = regexp.MustCompile(
        `'ANDROID_MUSIC':[^}]{0,500}?'clientVersion':\s*'([^']+)'`)
    reAPIKey = regexp.MustCompile(
        `'ANDROID_MUSIC':[^}]{0,500}?'innertube_key':\s*'([^']+)'`)
)

// Load returns current InnerTube tokens using the following precedence:
//  1. Disk cache (if < 24h old) — instant, no network
//  2. Live fetch from yt-dlp source — ~300ms one-time cost
//  3. Hardcoded fallback — always works, may be stale
//
// If disk cache is 12–24h old, a background refresh is triggered so the
// next startup will be instant with fresh tokens.
func Load(ctx context.Context, cacheDir string) *InnerTubeTokens {
    cachePath := filepath.Join(cacheDir, "innertube_tokens.json")

    if t := loadFromDisk(cachePath); t != nil {
        if time.Since(t.FetchedAt) > 12*time.Hour {
            go func() {
                if fresh, err := fetchFromSource(context.Background()); err == nil {
                    _ = saveToDisk(cachePath, fresh)
                }
            }()
        }
        return t
    }

    if t, err := fetchFromSource(ctx); err == nil {
        _ = saveToDisk(cachePath, t)
        return t
    }

    return &InnerTubeTokens{
        ClientName:    fallbackClientName,
        ClientVersion: fallbackClientVersion,
        APIKey:        fallbackAPIKey,
        FetchedAt:     time.Time{},
    }
}

func fetchFromSource(ctx context.Context) (*InnerTubeTokens, error) {
    reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(reqCtx, "GET", ytdlpSourceURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // The ANDROID_MUSIC block is near the top; 64KB is more than enough.
    body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
    if err != nil {
        return nil, err
    }

    src := string(body)
    version := extractFirst(reClientVersion, src)
    apiKey  := extractFirst(reAPIKey, src)

    if version == "" || apiKey == "" {
        return nil, fmt.Errorf("tokens: parse failed — yt-dlp source structure may have changed")
    }

    return &InnerTubeTokens{
        ClientName:    fallbackClientName, // name never changes
        ClientVersion: version,
        APIKey:        apiKey,
        FetchedAt:     time.Now(),
    }, nil
}

func extractFirst(re *regexp.Regexp, src string) string {
    m := re.FindStringSubmatch(src)
    if len(m) < 2 {
        return ""
    }
    return strings.TrimSpace(m[1])
}

func loadFromDisk(path string) *InnerTubeTokens {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil
    }
    var t InnerTubeTokens
    if err := json.Unmarshal(data, &t); err != nil {
        return nil
    }
    if time.Since(t.FetchedAt) > 24*time.Hour {
        return nil
    }
    return &t
}

func saveToDisk(path string, t *InnerTubeTokens) error {
    _ = os.MkdirAll(filepath.Dir(path), 0o755)
    data, err := json.MarshalIndent(t, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0o644)
}
```

### Wiring into main

```go
// In cmd/gotune/main.go, before any source is created:
tok := tokens.Load(ctx, cfg.Cache.ResolvedDir())

// Override with config values if the user has set them manually
// (non-empty config values take precedence over auto-fetched tokens)
if cfg.InnerTube.ClientVersion != "" {
    tok.ClientVersion = cfg.InnerTube.ClientVersion
}
if cfg.InnerTube.APIKey != "" {
    tok.APIKey = cfg.InnerTube.APIKey
}

src := source.NewYouTube(cfg, tok)
```

---

## 8. The Player Layer

### Purpose

Decode and play audio via ffmpeg → PCM pipe → oto. Control playback: pause, resume, seek, volume, stop. Report current position and state.

### Architecture

```
ffmpeg subprocess
  reads: stream URL over HTTPS
  writes: stdout pipe (raw PCM s16le, 44100Hz, stereo)
      ↓
FFmpegPlayer.Read() — called by oto, applies volume scaling
      ↓
oto/v3 Player — writes to audio device
      ↓
PipeWire/PulseAudio → speakers
```

### File: `internal/player/speaker.go`

oto context is initialized once for the process lifetime:

```go
package player

import "github.com/ebitengine/oto/v3"

type Speaker struct {
    ctx *oto.Context
}

func NewSpeaker(sampleRate int, bufferSizeInBytes int) (*Speaker, error) {
    ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
        SampleRate:        sampleRate,
        ChannelCount:      2,
        Format:            oto.FormatSignedInt16LE,
        BufferSizeInBytes: bufferSizeInBytes,
    })
    if err != nil {
        return nil, fmt.Errorf("player: failed to open audio device: %w", err)
    }
    <-ready
    return &Speaker{ctx: ctx}, nil
}

func (s *Speaker) Close() {
    // oto context has no explicit close in v3; GC handles it
}
```

Buffer size guidance:
- `4096` bytes = ~23ms latency at 44100Hz stereo s16le. Good default.
- Do not go below `2048` (glitches on slow systems).
- Do not go above `16384` (seek response feels sluggish).

### File: `internal/player/decoder.go`

ffmpeg subprocess management, factored out of the player:

```go
package player

type decoder struct {
    cmd    *exec.Cmd
    pipe   io.ReadCloser
    stderr *strings.Builder
}

func startDecoder(ffmpegPath string, url string, seekSecs float64, threads int, sampleRate int) (*decoder, error) {
    args := []string{
        "-hide_banner",
        "-loglevel", "error",
        "-probesize", "65536",
        "-analyzeduration", "100000",
        "-fflags", "nobuffer",
        "-flags", "low_delay",
    }

    // Place -ss before -i for fast keyframe seek
    if seekSecs > 0 {
        args = append(args, "-ss", fmt.Sprintf("%.2f", seekSecs))
    }

    args = append(args,
        "-i", url,
        "-vn", "-sn",
        "-threads", strconv.Itoa(threads),
        "-f", "s16le",
        "-ar", strconv.Itoa(sampleRate),
        "-ac", "2",
        "-flush_packets", "1",
        "pipe:1",
    )

    cmd := exec.Command(ffmpegPath, args...)

    var stderrBuf strings.Builder
    cmd.Stderr = &stderrBuf

    pipe, err := cmd.StdoutPipe()
    if err != nil {
        return nil, fmt.Errorf("decoder: stdout pipe: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("decoder: ffmpeg start: %w", err)
    }

    return &decoder{cmd: cmd, pipe: pipe, stderr: &stderrBuf}, nil
}

// stop kills ffmpeg in the correct order:
// kill → close pipe → wait (never the other order)
func (d *decoder) stop() {
    if d.cmd.Process != nil {
        _ = d.cmd.Process.Kill()
    }
    _ = d.pipe.Close()
    _ = d.cmd.Wait()
}
```

### File: `internal/player/player.go`

```go
package player

type FFmpegPlayer struct {
    cfg     *config.Config
    speaker *Speaker

    mu          sync.Mutex
    dec         *decoder
    otoPlayer   *oto.Player

    paused      bool
    volume      float64  // 0.0–1.0
    position    float64  // seconds
    duration    float64  // seconds, from caller
    currentURL  string
    bytesPerSec int      // sampleRate * 2 channels * 2 bytes

    stopCh  chan struct{}
    doneCh  chan struct{}
    onEnd   func()
}

func New(cfg *config.Config, speaker *Speaker) *FFmpegPlayer {
    bps := cfg.FFmpeg.SampleRate * 2 * 2
    return &FFmpegPlayer{
        cfg:         cfg,
        speaker:     speaker,
        volume:      float64(cfg.Playback.Volume) / 100.0,
        bytesPerSec: bps,
    }
}
```

#### Play

```go
func (p *FFmpegPlayer) Play(url string, duration float64, onEnd func()) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.stopLocked()

    dec, err := startDecoder(
        p.cfg.FFmpeg.Path, url, 0,
        p.cfg.FFmpeg.Threads, p.cfg.FFmpeg.SampleRate,
    )
    if err != nil {
        return err
    }

    // Prime: read one buffer's worth before handing to oto so playback
    // starts on real audio immediately rather than silence.
    primeBuf := make([]byte, p.cfg.Playback.BufferSizeInBytes)
    if _, err := io.ReadFull(dec.pipe, primeBuf); err != nil {
        dec.stop()
        return fmt.Errorf("player: prime read failed: %w", err)
    }
    primedReader := io.MultiReader(bytes.NewReader(primeBuf), dec.pipe)

    p.dec        = dec
    p.currentURL = url
    p.duration   = duration
    p.position   = 0
    p.paused     = false
    p.onEnd      = onEnd
    p.stopCh     = make(chan struct{})
    p.doneCh     = make(chan struct{})

    p.otoPlayer = p.speaker.ctx.NewPlayer(p.readerFor(primedReader))
    p.otoPlayer.Play()

    go p.watchLoop(p.dec, p.stopCh, p.doneCh)
    return nil
}
```

#### io.Reader implementation (oto calls this)

```go
// readerFor wraps the PCM source with position tracking and volume.
// Returns an io.Reader that oto calls to fill its audio buffer.
func (p *FFmpegPlayer) readerFor(src io.Reader) io.Reader {
    return &pcmReader{player: p, src: src}
}

type pcmReader struct {
    player *FFmpegPlayer
    src    io.Reader
}

func (r *pcmReader) Read(buf []byte) (int, error) {
    // Spin on pause — sleep 10ms, recheck.
    for {
        r.player.mu.Lock()
        paused := r.player.paused
        alive  := r.player.dec != nil
        r.player.mu.Unlock()

        if !paused || !alive {
            break
        }
        time.Sleep(10 * time.Millisecond)
    }

    r.player.mu.Lock()
    alive := r.player.dec != nil
    r.player.mu.Unlock()
    if !alive {
        return 0, io.EOF
    }

    n, err := r.src.Read(buf)
    if n > 0 {
        r.player.mu.Lock()
        vol := r.player.volume
        r.player.position += float64(n) / float64(r.player.bytesPerSec)
        r.player.mu.Unlock()
        applyVolume(buf[:n], vol)
    }
    return n, err
}
```

#### Volume scaling

```go
func applyVolume(buf []byte, volume float64) {
    if volume == 1.0 {
        return
    }
    for i := 0; i+1 < len(buf); i += 2 {
        sample := int16(buf[i]) | int16(buf[i+1])<<8
        scaled := int32(float64(sample) * volume)
        if scaled >  32767 { scaled =  32767 }
        if scaled < -32768 { scaled = -32768 }
        buf[i]   = byte(scaled)
        buf[i+1] = byte(uint16(scaled) >> 8)
    }
}
```

#### Seek

Seeking restarts ffmpeg at the target position with `-ss` placed before `-i` (fast keyframe seek):

```go
func (p *FFmpegPlayer) Seek(seconds float64) error {
    p.mu.Lock()
    url      := p.currentURL
    duration := p.duration
    onEnd    := p.onEnd
    p.mu.Unlock()

    if seconds < 0         { seconds = 0 }
    if seconds >= duration { seconds = duration - 1 }

    p.mu.Lock()
    p.stopLocked()
    p.mu.Unlock()

    dec, err := startDecoder(
        p.cfg.FFmpeg.Path, url, seconds,
        p.cfg.FFmpeg.Threads, p.cfg.FFmpeg.SampleRate,
    )
    if err != nil {
        return err
    }

    primeBuf := make([]byte, p.cfg.Playback.BufferSizeInBytes)
    if _, err := io.ReadFull(dec.pipe, primeBuf); err != nil {
        dec.stop()
        return fmt.Errorf("player: seek prime failed: %w", err)
    }
    primedReader := io.MultiReader(bytes.NewReader(primeBuf), dec.pipe)

    p.mu.Lock()
    p.dec        = dec
    p.position   = seconds
    p.paused     = false
    p.onEnd      = onEnd
    p.stopCh     = make(chan struct{})
    p.doneCh     = make(chan struct{})
    p.otoPlayer  = p.speaker.ctx.NewPlayer(p.readerFor(primedReader))
    p.otoPlayer.Play()
    p.mu.Unlock()

    go p.watchLoop(dec, p.stopCh, p.doneCh)
    return nil
}
```

#### Stop, Pause, Resume

```go
func (p *FFmpegPlayer) stopLocked() {
    // Must be called with p.mu held.
    if p.stopCh != nil {
        close(p.stopCh)
        p.stopCh = nil
    }
    if p.doneCh != nil {
        <-p.doneCh
        p.doneCh = nil
    }
    if p.otoPlayer != nil {
        _ = p.otoPlayer.Close()
        p.otoPlayer = nil
    }
    if p.dec != nil {
        p.dec.stop()
        p.dec = nil
    }
}

func (p *FFmpegPlayer) Stop() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.stopLocked()
}

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

func (p *FFmpegPlayer) SetVolume(pct int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if pct < 0   { pct = 0 }
    if pct > 100 { pct = 100 }
    p.volume = float64(pct) / 100.0
}
```

#### Getters (called by tick command, thread-safe)

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
    return p.dec != nil && !p.paused
}
```

#### watchLoop

```go
func (p *FFmpegPlayer) watchLoop(dec *decoder, stopCh <-chan struct{}, doneCh chan<- struct{}) {
    defer close(doneCh)

    exitCh := make(chan error, 1)
    go func() { exitCh <- dec.cmd.Wait() }()

    select {
    case ffmpegErr := <-exitCh:
        if ffmpegErr != nil {
            stderr := strings.TrimSpace(dec.stderr.String())
            if stderr != "" {
                // Log to stderr — TUI suppresses it, but it goes to journal
                fmt.Fprintf(os.Stderr, "ffmpeg: %s\n", stderr)
            }
        }
        if p.onEnd != nil {
            p.onEnd()
        }
    case <-stopCh:
        // Explicitly stopped — decoder already killed by stopLocked
        <-exitCh
    }
}
```

---

## 9. The Queue Layer

### Purpose

Manage the ordered list of upcoming tracks. This is business logic, not UI state. Separated so it can be tested without bubbletea, and so the UI queue view is a renderer of `queue.Queue`, not the queue itself.

### File: `internal/queue/queue.go`

```go
package queue

import (
    "sync"
    "github.com/yourname/gotune/internal/source"
)

type Queue struct {
    mu      sync.Mutex
    tracks  []source.Track
    history []source.Track  // last N played tracks for Prev()
    maxHist int
}

func New() *Queue {
    return &Queue{maxHist: 50}
}

func (q *Queue) Enqueue(t source.Track) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.tracks = append(q.tracks, t)
}

func (q *Queue) EnqueueFront(t source.Track) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.tracks = append([]source.Track{t}, q.tracks...)
}

// Next removes and returns the first track. Returns nil if empty.
func (q *Queue) Next() *source.Track {
    q.mu.Lock()
    defer q.mu.Unlock()
    if len(q.tracks) == 0 {
        return nil
    }
    t := q.tracks[0]
    q.tracks = q.tracks[1:]
    return &t
}

// Prev returns the most recently played track, if any.
func (q *Queue) Prev() *source.Track {
    q.mu.Lock()
    defer q.mu.Unlock()
    if len(q.history) == 0 {
        return nil
    }
    t := q.history[len(q.history)-1]
    q.history = q.history[:len(q.history)-1]
    return &t
}

func (q *Queue) PushHistory(t source.Track) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.history = append(q.history, t)
    if len(q.history) > q.maxHist {
        q.history = q.history[1:]
    }
}

func (q *Queue) Remove(index int) {
    q.mu.Lock()
    defer q.mu.Unlock()
    if index < 0 || index >= len(q.tracks) {
        return
    }
    q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
}

func (q *Queue) Len() int {
    q.mu.Lock()
    defer q.mu.Unlock()
    return len(q.tracks)
}

func (q *Queue) Snapshot() []source.Track {
    q.mu.Lock()
    defer q.mu.Unlock()
    out := make([]source.Track, len(q.tracks))
    copy(out, q.tracks)
    return out
}

func (q *Queue) Clear() {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.tracks = nil
}
```

---

## 10. The Cache Layer

### What is cached

Only **metadata + audio files**. Stream URLs are *not* cached — they expire in ~6 hours. The cache stores:
- The downloaded audio file (webm/opus or m4a/aac)
- A bbolt index entry with videoID, file path, size, last-accessed time

When a track is requested:
1. Check cache → if hit, pass the file path to ffmpeg instead of a URL
2. If miss, stream from the URL as normal
3. After natural playback end, trigger `DownloadAsync` to cache it for next time

### File: `internal/cache/index.go`

```go
package cache

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"
    bolt "go.etcd.io/bbolt"
)

var bucketName = []byte("tracks")

type Entry struct {
    VideoID       string    `json:"video_id"`
    FilePath      string    `json:"file_path"`
    Format        string    `json:"format"`
    FileSizeBytes int64     `json:"file_size_bytes"`
    LastAccessed  time.Time `json:"last_accessed"`
}

type Index struct {
    db *bolt.DB
}

func openIndex(path string) (*Index, error) {
    db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 1 * time.Second})
    if err != nil {
        os.Remove(path)
        db, err = bolt.Open(path, 0o600, nil)
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
    var e Entry
    err := idx.db.View(func(tx *bolt.Tx) error {
        v := tx.Bucket(bucketName).Get([]byte(videoID))
        if v == nil { return ErrNotFound }
        return json.Unmarshal(v, &e)
    })
    if err != nil { return nil, err }
    return &e, nil
}

func (idx *Index) Put(e *Entry) error {
    return idx.db.Update(func(tx *bolt.Tx) error {
        v, _ := json.Marshal(e)
        return tx.Bucket(bucketName).Put([]byte(e.VideoID), v)
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
            if json.Unmarshal(v, &e) == nil {
                entries = append(entries, &e)
            }
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
    audioDir   string
    partialDir string
    index      *Index
    maxBytes   int64
    cfg        *config.Config
    wg         sync.WaitGroup // tracks in-flight DownloadAsync calls
}

func New(cfg *config.Config) (*Cache, error) {
    base       := cfg.Cache.ResolvedDir()
    audioDir   := filepath.Join(base, "audio")
    partialDir := filepath.Join(base, "partial")
    indexPath  := filepath.Join(base, "index.db")

    for _, dir := range []string{audioDir, partialDir} {
        if err := os.MkdirAll(dir, 0o755); err != nil {
            return nil, fmt.Errorf("cache: mkdir %s: %w", dir, err)
        }
    }

    idx, err := openIndex(indexPath)
    if err != nil { return nil, err }

    c := &Cache{
        audioDir:   audioDir,
        partialDir: partialDir,
        index:      idx,
        maxBytes:   int64(cfg.Cache.MaxSizeMB) * 1024 * 1024,
        cfg:        cfg,
    }
    c.cleanPartials()
    return c, nil
}

// Close waits for in-flight downloads then closes bbolt.
// Call this after the TUI exits.
func (c *Cache) Close() error {
    c.wg.Wait()
    return c.index.db.Close()
}

func (c *Cache) Get(videoID string) (filePath string, hit bool) {
    e, err := c.index.Get(videoID)
    if err != nil { return "", false }
    if _, err := os.Stat(e.FilePath); err != nil {
        _ = c.index.Delete(videoID)
        return "", false
    }
    e.LastAccessed = time.Now()
    _ = c.index.Put(e)
    return e.FilePath, true
}

// DownloadAsync downloads videoID in the background using a direct URL
// from the source. Called after natural track end. Fire-and-forget.
func (c *Cache) DownloadAsync(videoID string, streamURL string, format string) {
    c.wg.Add(1)
    go func() {
        defer c.wg.Done()
        if _, hit := c.Get(videoID); hit { return }

        partPath  := filepath.Join(c.partialDir, videoID+".part")
        finalPath := filepath.Join(c.audioDir, videoID+"."+format)

        // Use ffmpeg to download the stream to a file (no yt-dlp needed)
        cmd := exec.Command(c.cfg.FFmpeg.Path,
            "-hide_banner", "-loglevel", "error",
            "-i", streamURL,
            "-vn", "-c", "copy",   // copy codec, no re-encode
            partPath,
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
        if err != nil { return }

        _ = c.index.Put(&Entry{
            VideoID:       videoID,
            FilePath:      finalPath,
            Format:        format,
            FileSizeBytes: info.Size(),
            LastAccessed:  time.Now(),
        })

        c.maybeEvict()
    }()
}

func (c *Cache) maybeEvict() {
    entries, err := c.index.All()
    if err != nil { return }

    var total int64
    for _, e := range entries { total += e.FileSizeBytes }
    if total <= c.maxBytes { return }

    sort.Slice(entries, func(i, j int) bool {
        return entries[i].LastAccessed.Before(entries[j].LastAccessed)
    })

    for _, e := range entries {
        if total <= c.maxBytes { break }
        os.Remove(e.FilePath)
        _ = c.index.Delete(e.VideoID)
        total -= e.FileSizeBytes
    }
}

type Stats struct {
    TrackCount int
    UsedBytes  int64
    MaxBytes   int64
    Directory  string
}

func (c *Cache) Stats() (Stats, error) {
    entries, err := c.index.All()
    if err != nil { return Stats{}, err }
    s := Stats{MaxBytes: c.maxBytes, Directory: c.audioDir}
    for _, e := range entries {
        s.TrackCount++
        s.UsedBytes += e.FileSizeBytes
    }
    return s, nil
}

func (c *Cache) Clear() (int64, error) {
    entries, err := c.index.All()
    if err != nil { return 0, err }
    var freed int64
    for _, e := range entries {
        if err := os.Remove(e.FilePath); err == nil {
            freed += e.FileSizeBytes
        }
        _ = c.index.Delete(e.VideoID)
    }
    return freed, nil
}

func (c *Cache) cleanPartials() {
    _ = filepath.WalkDir(c.partialDir, func(path string, d fs.DirEntry, err error) error {
        if err == nil && !d.IsDir() { os.Remove(path) }
        return nil
    })
}
```

Note: `DownloadAsync` now uses **ffmpeg** to save the stream to disk (with `-c copy` to avoid re-encoding). This removes the last runtime use of yt-dlp. The stream URL passed in is the same one that was just used for playback — it is still valid at download time since downloads happen immediately after play starts.

---

## 11. The MPRIS2 Layer

All files in this package carry `//go:build linux` at the top.

### Purpose

Register gotune on the D-Bus session bus as an MPRIS2 media player. Makes media keys (play/pause, next, previous) work in GNOME, KDE, and any MPRIS2-aware environment. Shows now-playing info in the desktop's media widget.

### File: `internal/mpris/mpris.go`

```go
//go:build linux

package mpris

import (
    "github.com/godbus/dbus/v5"
    "github.com/godbus/dbus/v5/prop"
)

const (
    serviceName = "org.mpris.MediaPlayer2.gotune"
    objectPath  = "/org/mpris/MediaPlayer2"
    playerIface = "org.mpris.MediaPlayer2.Player"
    rootIface   = "org.mpris.MediaPlayer2"
    propsIface  = "org.freedesktop.DBus.Properties"
)

type Service struct {
    conn    *dbus.Conn
    props   *prop.Properties
    program interface{ Send(interface{}) }
}

func Register(program interface{ Send(interface{}) }) (*Service, error) {
    conn, err := dbus.SessionBus()
    if err != nil {
        return nil, fmt.Errorf("mpris: D-Bus unavailable: %w", err)
    }
    reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
    if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
        conn.Close()
        return nil, fmt.Errorf("mpris: could not acquire service name")
    }
    s := &Service{conn: conn, program: program}
    conn.Export(s, dbus.ObjectPath(objectPath), rootIface)
    conn.Export(s, dbus.ObjectPath(objectPath), playerIface)
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

// Noop returns a no-op Service for when MPRIS is unavailable.
func Noop() *Service { return &Service{} }

// Root interface
func (s *Service) Raise() *dbus.Error { return nil }
func (s *Service) Quit() *dbus.Error {
    if s.program != nil { s.program.Send(QuitMsg{}) }
    return nil
}

// Player interface
func (s *Service) Play()      *dbus.Error { s.send(MediaKeyMsg{Action: ActionPlay}); return nil }
func (s *Service) Pause()     *dbus.Error { s.send(MediaKeyMsg{Action: ActionPause}); return nil }
func (s *Service) PlayPause() *dbus.Error { s.send(MediaKeyMsg{Action: ActionPlayPause}); return nil }
func (s *Service) Stop()      *dbus.Error { s.send(MediaKeyMsg{Action: ActionStop}); return nil }
func (s *Service) Next()      *dbus.Error { s.send(MediaKeyMsg{Action: ActionNext}); return nil }
func (s *Service) Previous()  *dbus.Error { s.send(MediaKeyMsg{Action: ActionPrevious}); return nil }

func (s *Service) Seek(offset int64) *dbus.Error {
    s.send(MediaSeekMsg{OffsetSeconds: float64(offset) / 1e6})
    return nil
}
func (s *Service) SetPosition(_ dbus.ObjectPath, pos int64) *dbus.Error {
    s.send(MediaSeekMsg{AbsoluteSeconds: float64(pos) / 1e6, IsAbsolute: true})
    return nil
}
func (s *Service) OpenUri(_ string) *dbus.Error { return nil }

func (s *Service) send(msg interface{}) {
    if s.program != nil { s.program.Send(msg) }
}
```

### File: `internal/mpris/properties.go`

```go
//go:build linux

package mpris

import "github.com/yourname/gotune/internal/source"

func (s *Service) UpdatePlaybackStatus(playing, paused bool) {
    if s.props == nil { return }
    status := "Stopped"
    if playing && !paused { status = "Playing" }
    if playing && paused  { status = "Paused" }
    s.props.SetMust(playerIface, "PlaybackStatus", dbus.MakeVariant(status))
    s.emitChanged(playerIface, map[string]dbus.Variant{
        "PlaybackStatus": dbus.MakeVariant(status),
    })
}

func (s *Service) UpdateMetadata(t *source.Track) {
    if s.props == nil || t == nil { return }
    trackID := dbus.ObjectPath("/org/mpris/MediaPlayer2/track/" + t.VideoID)
    meta := map[string]dbus.Variant{
        "mpris:trackid": dbus.MakeVariant(trackID),
        "xesam:title":   dbus.MakeVariant(t.Title),
        "xesam:artist":  dbus.MakeVariant([]string{t.Artist}),
        "xesam:album":   dbus.MakeVariant(t.Album),
        "mpris:length":  dbus.MakeVariant(int64(t.Duration) * 1e6),
    }
    s.props.SetMust(playerIface, "Metadata", dbus.MakeVariant(meta))
    s.emitChanged(playerIface, map[string]dbus.Variant{"Metadata": dbus.MakeVariant(meta)})
}

func (s *Service) UpdatePosition(seconds float64) {
    if s.props == nil { return }
    s.props.SetMust(playerIface, "Position", dbus.MakeVariant(int64(seconds*1e6)))
}

func (s *Service) EmitSeeked(seconds float64) {
    if s.conn == nil { return }
    s.conn.Emit(dbus.ObjectPath(objectPath), playerIface+".Seeked", int64(seconds*1e6))
}

func (s *Service) emitChanged(iface string, changed map[string]dbus.Variant) {
    if s.conn == nil { return }
    s.conn.Emit(dbus.ObjectPath(objectPath), propsIface+".PropertiesChanged",
        iface, changed, []string{})
}
```

MPRIS failure is non-fatal. If `Register` fails, `mpris.Noop()` is used and media keys simply don't work.

---

## 12. The TUI Layer

### Framework rules

- Never do I/O inside `Update`. Spawn a `tea.Cmd` instead.
- `View` is called after every `Update`. Keep it fast — no allocations beyond string building.
- Always handle `tea.WindowSizeMsg` everywhere to reflow on resize.
- Use `lipgloss.JoinVertical` / `lipgloss.JoinHorizontal` for layout.

### File: `internal/ui/msgs.go`

All message types live here. Nothing else defines `tea.Msg` types.

```go
package ui

type SearchResultsMsg struct {
    Results []source.Track
    Err     error
}

type StreamReadyMsg struct {
    VideoID string
    URL     string
    Format  string
    Err     error
}

type TrackEndedMsg struct {
    VideoID  string
    URL      string   // the URL that was playing (passed to DownloadAsync)
    Format   string
    Natural  bool     // true = end of track, false = error/kill
}

type TickMsg struct {
    Position float64
    Duration float64
    IsPaused bool
}

type CacheStatsMsg struct { Stats cache.Stats }
type CacheClearedMsg struct { FreedBytes int64; Err error }
type ToastMsg struct { Text string; IsError bool }
type dismissToastMsg struct{}
type QuitMsg struct{}

// From MPRIS
type MediaKeyMsg struct { Action mpris.MediaAction }
type MediaSeekMsg struct {
    OffsetSeconds   float64
    AbsoluteSeconds float64
    IsAbsolute      bool
}
```

### File: `internal/ui/model.go`

Root model — `Init`, `Update`, `View` only. No msg type definitions here.

```go
type ViewState int
const (
    ViewSearch ViewState = iota
    ViewQueue
    ViewCacheManager
)

type Model struct {
    // Sub-models
    search    SearchModel
    queueView QueueViewModel
    playerBar PlayerBarModel
    cacheView CacheViewModel
    toast     ToastModel

    // Playback state
    currentTrack *source.Track
    currentURL   string
    currentFmt   string
    isPlaying    bool
    isPaused     bool
    position     float64
    duration     float64
    volume       int

    // Services
    src     source.Source
    pl      *player.FFmpegPlayer
    q       *queue.Queue
    cache   *cache.Cache
    mpris   *mpris.Service
    program *tea.Program  // set via WithProgram after tea.NewProgram

    // Layout
    viewState ViewState
    width     int
    height    int
}
```

`program` is needed to pass to `onEnd` callbacks that call `program.Send(TrackEndedMsg{...})`. Wire it after `tea.NewProgram` returns:

```go
prog := tea.NewProgram(model, tea.WithAltScreen())
model = model.WithProgram(prog)
```

### File: `internal/ui/keys.go`

```go
var Keys = KeyMap{
    FocusSearch:  key.NewBinding(key.WithKeys("/"),            key.WithHelp("/",      "search")),
    SelectItem:   key.NewBinding(key.WithKeys("enter"),        key.WithHelp("enter",  "play")),
    AddToQueue:   key.NewBinding(key.WithKeys("a"),            key.WithHelp("a",      "queue")),
    TogglePause:  key.NewBinding(key.WithKeys(" "),            key.WithHelp("space",  "pause")),
    SkipNext:     key.NewBinding(key.WithKeys("n"),            key.WithHelp("n",      "next")),
    SkipPrev:     key.NewBinding(key.WithKeys("p"),            key.WithHelp("p",      "prev")),
    SeekForward:  key.NewBinding(key.WithKeys("right", "l"),   key.WithHelp("→/l",   "+10s")),
    SeekBackward: key.NewBinding(key.WithKeys("left",  "h"),   key.WithHelp("←/h",   "-10s")),
    VolumeUp:     key.NewBinding(key.WithKeys("+", "="),       key.WithHelp("+",      "vol+")),
    VolumeDown:   key.NewBinding(key.WithKeys("-"),            key.WithHelp("-",      "vol-")),
    SwitchView:   key.NewBinding(key.WithKeys("tab"),          key.WithHelp("tab",    "queue")),
    CacheManager: key.NewBinding(key.WithKeys("c"),            key.WithHelp("c",      "cache")),
    CursorUp:     key.NewBinding(key.WithKeys("up",   "k"),    key.WithHelp("↑/k",   "up")),
    CursorDown:   key.NewBinding(key.WithKeys("down", "j"),    key.WithHelp("↓/j",   "down")),
    DeleteItem:   key.NewBinding(key.WithKeys("d"),            key.WithHelp("d",      "remove")),
    Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"),  key.WithHelp("q",      "quit")),
    Help:         key.NewBinding(key.WithKeys("?"),            key.WithHelp("?",      "help")),
    Escape:       key.NewBinding(key.WithKeys("esc"),          key.WithHelp("esc",    "back")),
}
```

### Commands

```go
func searchCmd(src source.Source, query string) tea.Cmd {
    return func() tea.Msg {
        results, err := src.Search(context.Background(), query)
        return SearchResultsMsg{Results: results, Err: err}
    }
}

func resolveCmd(src source.Source, track source.Track) tea.Cmd {
    return func() tea.Msg {
        url, err := src.StreamURL(context.Background(), track.VideoID)
        if err != nil {
            return StreamReadyMsg{VideoID: track.VideoID, Err: err}
        }
        return StreamReadyMsg{
            VideoID: track.VideoID,
            URL:     url,
            Format:  "webm", // inferred; refine if Piped returns format info
        }
    }
}

func tickCmd(p *player.FFmpegPlayer) tea.Cmd {
    return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
        return TickMsg{
            Position: p.Position(),
            Duration: p.Duration(),
            IsPaused: p.IsPaused(),
        }
    })
}

func radioCmd(src source.Source, seedID string) tea.Cmd {
    return func() tea.Msg {
        tracks, err := src.Radio(context.Background(), seedID)
        if err != nil {
            return ToastMsg{Text: "Radio failed: " + err.Error(), IsError: true}
        }
        return SearchResultsMsg{Results: tracks}
    }
}
```

### `Update` — key cases

```go
case StreamReadyMsg:
    if msg.Err != nil {
        return m, toastCmd("Cannot play: "+msg.Err.Error(), true)
    }
    m.currentURL = msg.URL
    m.currentFmt = msg.Format
    m.isPlaying  = true
    m.isPaused   = false
    err := m.pl.Play(msg.URL, float64(m.currentTrack.Duration), func() {
        m.program.Send(TrackEndedMsg{
            VideoID: m.currentTrack.VideoID,
            URL:     m.currentURL,
            Format:  m.currentFmt,
            Natural: true,
        })
    })
    if err != nil {
        return m, toastCmd("Playback error: "+err.Error(), true)
    }
    m.mpris.UpdateMetadata(m.currentTrack)
    m.mpris.UpdatePlaybackStatus(true, false)
    return m, tickCmd(m.pl)

case TrackEndedMsg:
    m.isPlaying = false
    if msg.Natural && msg.URL != "" {
        m.cache.DownloadAsync(msg.VideoID, msg.URL, msg.Format)
    }
    // Premature end detection
    if msg.Natural && m.position < m.duration*0.95 && m.duration > 30 {
        return m, tea.Batch(
            resolveCmd(m.src, *m.currentTrack),
            toastCmd("Reconnecting...", false),
        )
    }
    return m, m.playNextCmd()

func (m Model) playNextCmd() tea.Cmd {
    next := m.q.Next()
    if next == nil {
        if m.currentTrack != nil {
            return radioCmd(m.src, m.currentTrack.VideoID)
        }
        return nil
    }
    m.q.PushHistory(*m.currentTrack)
    m.currentTrack = next
    return resolveCmd(m.src, *next)
}
```

### View structure

```
┌─────────────────────────────────────────┐
│  toast (optional, 1–2 lines)            │
├─────────────────────────────────────────┤
│                                         │
│  content area (search or queue view)    │
│                                         │
├─────────────────────────────────────────┤
│  player bar (always 4 lines)            │
│  [▶] Title • Artist         [cached] 🔊80│
│  0:00 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 3:45 │
│                                         │
└─────────────────────────────────────────┘
```

If `m.width < 60 || m.height < 10`, render only:
```
Terminal too small. Please resize to at least 60×10.
```

---

## 13. Concurrency Model

```
main goroutine
  └─ tea.Program.Run() — bubbletea event loop

Signal goroutine
  └─ os.Signal → program.Send(QuitMsg{})

Per-track goroutines (managed by player package)
  ├─ watchLoop — waits for ffmpeg exit, calls onEnd
  └─ pcmReader.Read — called by oto's internal goroutine

Background goroutines (fire-and-forget with WaitGroup)
  └─ cache.DownloadAsync — ffmpeg -c copy download

Token refresh goroutine (one per startup if cache is stale)
  └─ tokens.fetchFromSource → saveToDisk
```

**Rules:**
- All player state mutation goes through `p.mu` (sync.Mutex).
- The oto goroutine calls `pcmReader.Read` concurrently with UI goroutine. Never access player fields from Read without holding `p.mu`.
- `onEnd` callback fires from `watchLoop` goroutine — it must only call `program.Send(...)`, never touch model state directly.
- `cache.Close()` blocks on `wg.Wait()` — always call it after `prog.Run()` returns, before `os.Exit`.
- The bubbletea `Update` function is single-threaded. All state mutation in Update is safe without locks.

---

## 14. Error Handling

**Sentinel errors** (check with `errors.Is`):
```go
source.ErrRateLimited  // show "rate limited" toast, backoff
source.ErrNoStream     // show "cannot play" toast
cache.ErrNotFound      // cache miss — normal, not an error
```

**Error display:**
- Non-fatal errors → `ToastMsg{IsError: true}` → red toast, 3s auto-dismiss
- Fatal errors (audio device, config parse, bbolt lock) → `fmt.Fprintln(os.Stderr, ...)` + `os.Exit(1)` before TUI starts

**Never panic** in any package except `cmd/gotune/main.go` initialization before the TUI starts. After the TUI is running, all errors become messages.

---

## 15. Signal Handling and Clean Shutdown

```go
// cmd/gotune/main.go

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigCh
    prog.Send(ui.QuitMsg{})
}()

if _, err := prog.Run(); err != nil {
    fmt.Fprintln(os.Stderr, err)
}

// Ordered cleanup after prog.Run() returns:
pl.Stop()           // kills ffmpeg, closes oto player
speaker.Close()     // releases oto context
mprisSvc.Close()    // releases D-Bus name
cache.Close()       // wg.Wait() then bbolt close
```

`tea.Quit` command triggers bubbletea's shutdown, which causes `prog.Run()` to return, and cleanup runs sequentially in the correct order.

---

## 16. Build System

```makefile
BINARY  := gotune
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

.PHONY: build run test lint clean install

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/gotune

run: build
	./$(BINARY)

test:
	go test ./... -v -race -count=1

test-unit:
	go test ./... -v -race -count=1 -short

test-integration:
	go test -tags integration ./... -v -run TestReal

lint:
	go vet ./...
	staticcheck ./...

clean:
	rm -f $(BINARY)
	go clean ./...

install: build
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
```

---

## 17. Testing Strategy

### Unit tests

**`source/youtube_test.go`**

Save a real search response:
```bash
curl -s -X POST \
  "https://music.youtube.com/youtubei/v1/search?key=AIzaSyC9XL3ZjWddXya6X74dJoCTL-KLET5YdCE" \
  -H "Content-Type: application/json" \
  -H "X-YouTube-Client-Name: 67" \
  -H "X-YouTube-Client-Version: 1.20231204.01.00" \
  -d '{"context":{"client":{"clientName":"WEB_REMIX","clientVersion":"1.20231204.01.00","hl":"en","gl":"US"}},"query":"test","params":"EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D"}' \
  > testdata/search_response.json
```

Test parses the fixture — no network calls:
```go
func TestParseSearchResponse(t *testing.T) {
    data, _ := os.ReadFile("testdata/search_response.json")
    results, err := parseSearchResponse(data)
    require.NoError(t, err)
    require.NotEmpty(t, results)
    assert.NotEmpty(t, results[0].VideoID)
    assert.NotEmpty(t, results[0].Title)
    assert.Greater(t, results[0].Duration, 0)
}
```

**`tokens/tokens_test.go`**

```go
func TestFallbackOnNetworkFailure(t *testing.T) {
    // Pass a context that's already cancelled to force network failure
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    tok := Load(ctx, t.TempDir())
    assert.Equal(t, fallbackClientName, tok.ClientName)
    assert.NotEmpty(t, tok.ClientVersion)
}
```

**`player/player_test.go`**

```go
func TestApplyVolume(t *testing.T) {
    buf := []byte{0xFF, 0x7F} // 0x7FFF = 32767 in s16le
    applyVolume(buf, 0.5)
    result := int16(buf[0]) | int16(buf[1])<<8
    assert.InDelta(t, 16383, int(result), 2)
}
```

**`cache/cache_test.go`**

```go
func TestLRUEviction(t *testing.T) { ... }
func TestStaleIndexRepair(t *testing.T) { ... }
func TestGetMissOnPartialFile(t *testing.T) { ... }
func TestDownloadAsyncWaitOnClose(t *testing.T) { ... } // ensures wg.Wait works
```

**`queue/queue_test.go`**

```go
func TestNextReturnsNilOnEmpty(t *testing.T) { ... }
func TestHistoryLimit(t *testing.T) { ... }
func TestRemoveMiddle(t *testing.T) { ... }
```

### Integration tests (build tag: `integration`)

Require network, no mocks:

```go
//go:build integration

func TestRealSearch(t *testing.T) {
    src := source.NewYouTube(config.Defaults(), tokens.Load(context.Background(), t.TempDir()))
    results, err := src.Search(context.Background(), "never gonna give you up")
    require.NoError(t, err)
    require.NotEmpty(t, results)
    found := slices.ContainsFunc(results, func(r source.Track) bool {
        return strings.Contains(strings.ToLower(r.Artist), "rick astley")
    })
    assert.True(t, found)
}

func TestRealStreamURL(t *testing.T) {
    src := source.NewYouTube(config.Defaults(), tokens.Load(context.Background(), t.TempDir()))
    url, err := src.StreamURL(context.Background(), "dQw4w9WgXcQ")
    require.NoError(t, err)
    assert.HasPrefix(t, url, "https://")
}
```

Run with: `go test -tags integration ./... -v -run TestReal`

---

## 18. Phase-by-Phase Execution Plan

### Phase 0: Spike — Prove audio pipeline (done)

The POC from our sessions already proves the full pipeline:
- InnerTube ANDROID_MUSIC → stream URL (~200ms)
- ffmpeg with `-probesize 65536 -analyzeduration 100000 -fflags nobuffer`
- Prime buffer before oto player creation
- oto/v3 `NewContextOptions` with `BufferSizeInBytes: 4096`
- context-based shutdown, exponential backoff, buffered stderr

Delete the POC directory. Do not carry its code into the main project — rewrite it cleanly using the structures above.

### Phase 1: Config + Tokens + Structure (1 day)

1. `go mod init`, add all dependencies.
2. Create full directory structure with empty stub files (correct `package` declarations).
3. Write `internal/config/config.go` completely. Test: `TestLoad` verifies defaults are written and re-read.
4. Write `internal/tokens/tokens.go` completely. Test: `TestFallbackOnNetworkFailure`.
5. Write `cmd/gotune/main.go` stub: `checkDependencies()` + config load + token load. Verify it runs and creates `~/.config/gotune/config.toml`.

### Phase 2: Source — Search + StreamURL (2 days)

1. Implement `internal/source/youtube.go`: Search, StreamURL, Radio.
2. Implement `internal/source/piped.go`: StreamURL fallback.
3. Save `testdata/search_response.json` using the curl command in Section 17.
4. Write `source/youtube_test.go` with fixture-based parse test.
5. Manual test: temporary `main()` that searches and prints results.
6. Verify edge cases: podcast episodes and live streams return nil from `parseTrack`.

### Phase 3: Player (2 days)

1. Implement `internal/player/speaker.go`.
2. Implement `internal/player/decoder.go`.
3. Implement `internal/player/player.go`: Play, Seek, Pause, Resume, SetVolume, Stop, getters.
4. Write `player_test.go`: `TestApplyVolume`.
5. Manual test: `main()` that resolves a stream URL and plays for 10s, then seeks to 60s, then pauses, then quits. Validates all player functions before TUI exists.

### Phase 4: Queue (0.5 days)

1. Implement `internal/queue/queue.go`.
2. Write all queue unit tests.

### Phase 5: Cache (1.5 days)

1. Implement `internal/cache/index.go` and `cache.go`.
2. Write all cache unit tests including `TestDownloadAsyncWaitOnClose`.
3. Integration test: after a track plays, verify `DownloadAsync` writes a valid file and the index entry is retrievable.
4. Test LRU eviction with `maxBytes = 100`.

### Phase 6: TUI (4 days)

1. Write `styles.go` and `keys.go` completely.
2. Write `msgs.go` with all message types.
3. Write root `model.go`: struct + `Init` + empty `Update`/`View`.
4. Write `search.go`: input + result list, wire to source.
5. Wire the full play flow: search → select → resolve → play. At this point you have a working (ugly) TUI.
6. Write `playerbar.go`: progress bar, track info, volume.
7. Write `queue.go` view: append, remove, next.
8. Wire radio: when queue is empty after `TrackEndedMsg`, call `radioCmd`.
9. Write `toast.go`.
10. Write `cacheview.go`.
11. Handle `tea.WindowSizeMsg` everywhere. Test by resizing while playing.

### Phase 7: MPRIS2 (1 day)

1. Implement `internal/mpris/mpris.go` and `properties.go` (both with `//go:build linux`).
2. Wire into `main.go`.
3. Test: play a track, press media key, verify pause. Check GNOME media indicator shows title.

### Phase 8: Hardening (2 days)

1. `go test -race ./...` — fix all races.
2. Test token rotation: manually set wrong `ClientVersion` in config, verify fallback to Piped works.
3. Test Piped fallback: set `piped.instance_url` to an invalid URL, verify graceful error.
4. Test kill ffmpeg externally while playing: `kill $(pgrep ffmpeg)`. Verify `TrackEndedMsg` arrives and restart works.
5. `go vet ./...` and `staticcheck ./...` — fix all issues.
6. Final manual run: search → play → pause → seek → skip → queue multiple → cache check → clear cache → quit.

---

## 19. Known Failure Modes and Mitigations

### InnerTube token rotation

**Symptom**: `StreamURL` returns empty `adaptiveFormats` or a `LOGIN_REQUIRED` playability status.

**Automatic mitigation**: The token auto-refresh system detects a 12h-old cache and refreshes from yt-dlp's source in the background. Within one restart, tokens will be current.

**Immediate mitigation**: The Piped fallback kicks in automatically when InnerTube returns no streams.

**Manual override**: Update `[innertube]` section in `config.toml`. Find current values at:
```
https://github.com/yt-dlp/yt-dlp/blob/master/yt_dlp/extractor/youtube.py
```
Search for `ANDROID_MUSIC`.

### Search response shape change

**Symptom**: Search returns 0 results, no error. Or a JSON key error in `dig`.

**Diagnosis**: Open Chrome DevTools on `music.youtube.com`, search something, inspect the POST to `/youtubei/v1/search`. Compare response shape to `parseTrack`.

**Fix**: Update field paths in `parseTrack`. The `dig` helper makes this mechanical.

**Frequency**: ~3–4 times per year historically.

### Piped instance unavailable

**Symptom**: Piped fallback returns HTTP 5xx or connection refused.

**Fix**: Change `[piped] instance_url` in config to another instance from:
```
https://github.com/TeamPiped/Piped/wiki/Instances
```
Or set `[piped] enabled = false` to skip the fallback entirely.

### Expired stream URL mid-playback (ffmpeg 403)

**Symptom**: Track stops early. `TrackEndedMsg` arrives with `position < duration * 0.95`.

**Mitigation**: Already handled in `Update`:
```go
if msg.Natural && m.position < m.duration*0.95 && m.duration > 30 {
    return m, tea.Batch(resolveCmd(m.src, *m.currentTrack), toastCmd("Reconnecting...", false))
}
```
Re-resolve triggers a fresh URL from InnerTube.

### oto audio device unavailable

**Symptom**: `NewSpeaker` returns error.

**Fix**: Fatal. Print and exit before TUI starts:
```
gotune: could not open audio device.
Is PipeWire running? Try: systemctl --user status pipewire
```

### bbolt lock timeout

**Symptom**: `cache.New` returns timeout error (another gotune instance is running).

**Fix**: Non-fatal at cache level, fatal at startup. Print:
```
Another gotune instance may be running.
If not: rm ~/.cache/gotune/index.db
```

### Token regex breaks (yt-dlp restructures youtube.py)

**Symptom**: `tokens.fetchFromSource` logs "parse failed", falls back to hardcoded values.

**Fix**: Update the regex in `tokens/tokens.go`. Check the current structure at the yt-dlp URL. This has happened approximately twice in four years.

### Terminal too small

**Fix**: In `View()`:
```go
if m.width < 60 || m.height < 10 {
    return "Terminal too small. Resize to at least 60×10."
}
```

### Race between DownloadAsync and shutdown

**Mitigation**: Already handled. `cache.Close()` calls `wg.Wait()` before closing bbolt. Always call `cache.Close()` after `prog.Run()` returns.

---

*End of plan.*
*Approximate size: ~3,200 lines of Go across all packages, excluding tests.*
*Runtime dependencies: ffmpeg only.*
*Build dependencies: gcc (for oto CGO).*
*No Python. No yt-dlp binary at runtime.*