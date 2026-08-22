# goktave

```
█▀▀▀█  █▀▀▀█  █  ▄▀  ▀▀█▀▀  █▀▀▀█  █   █  █▀▀▀█
█ ▀▀█  █   █  █▀▀▄     █    █▀▀▀█  █   █  █▀▀▀ 
▀▀▀▀▀  ▀▀▀▀▀  ▀  ▀     ▀    ▀   ▀   ▀▀▀   ▀▀▀▀▀
```

A terminal-based (TUI) music streaming system that plays audio from YouTube Music.

---

## Features

- **Search** — Search YouTube Music directly from the terminal
- **Streaming** — Stream audio via yt-dlp + ffmpeg; no YouTube Premium required
- **Queue** — View, add, remove, reorder, and clear the playback queue; auto-populated with "Up Next" radio tracks
- **Playback** — Play, pause, next, previous, volume control
- **Audio Visualizer** — Real-time FFT-based spectrum analyzer in the TUI
- **Lyrics** — Synced LRC lyrics via [lrclib.net](https://lrclib.net) with YouTube Music fallback; cached to disk with downloads
- **Preloading** — Next track is preloaded during playback for near-gapless transitions
- **Downloads** — Download tracks for offline playback with LRU cache eviction
- **Playlists** — Create, delete, and play custom playlists; add tracks from search results
- **Likes** — Like/unlike tracks; view all liked tracks
- **History** — Automatic playback history with navigation back to previous tracks
- **Search Suggestions** — Autocomplete suggestions from YouTube Music as you type
- **Album Art** — Truecolor half-block artwork rendered natively in the now-playing panel (aspect-correct, auto-sized to your terminal)
- **Themes** — 4 built-in color themes: Terracotta, Catppuccin Mocha, Nord, Everforest
- **MPRIS2** — Integrates with Linux desktop media keys and notification area (GNOME, KDE, etc.)
- **Database** — Persistent storage via bbolt for likes, history, playlists, tracks, and settings
- **Logging** — Built-in logging facility for debugging

## Dependencies

- **ffmpeg** — Audio decoding, PCM streaming and album art processing
- **yt-dlp** — YouTube stream URL extraction

Install on Fedora:
```bash
sudo dnf install ffmpeg
pip install yt-dlp
# or standalone binary:
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && sudo chmod +x /usr/local/bin/yt-dlp
```

## Build

```bash
git clone https://github.com/Kush-Singh-26/goktave
cd goktave
go build -o goktave ./cmd/goktave
```

Requires Go 1.26.2+ and a C compiler (oto uses CGO for audio output).

## Usage

```bash
./goktave
```

### Keybindings

| Key | Action |
|---|---|
| `s` / `/` | Focus search |
| `up` / `k`, `down` / `j` | Navigate lists |
| `enter` | Play selected track |
| `space` | Play / Pause |
| `n` | Next track |
| `z` | Previous track |
| `q` | Focus queue |
| `K`, `J` | Move item up/down in queue |
| `x` | Remove from queue |
| `c` | Clear queue |
| `a` | Add selected track to queue |
| `l` | Toggle like |
| `d` | Download track |
| `p` | Add to playlist |
| `C` | Create playlist |
| `D` | Delete playlist |
| `P` | Play entire playlist |
| `1`–`7` | Switch tabs (Results, Lyrics, Playlists, Liked, History, Downloads, Settings) |
| `?` | Toggle full help |
| `ctrl+c` | Quit |

## Themes

Switch themes in the Settings tab (accessible via `7`):

- **Terracotta** (default) — Warm earth tones
- **Catppuccin Mocha** — Popular pastel dark theme
- **Nord** — Arctic, bluish pastel theme
- **Everforest** — Green, warm forest tones

## Architecture

```
cmd/goktave/main.go      — Entry point, dependency wiring
internal/
  config/                — Configuration defaults and persistence
  provider/              — YouTube Music API client (search, suggestions, radio, lyrics)
  extractor/             — yt-dlp subprocess wrapper for stream URL extraction
  player/                — Audio playback via ffmpeg → PCM → oto, FFT visualizer
  engine/                — Core business logic, state, queue, history, downloads, cache, LrcLib lyrics
  ui/                    — Bubbletea TUI: model, update, view, components, themes
  db/                    — bbolt database for likes, history, playlists, tracks
  mpris/                 — MPRIS2 D-Bus service for media keys and desktop integration
  logger/                — Structured logging
  thumbnail/             — ASCII thumbnail rendering
```

---

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Lipgloss](https://github.com/charmbracelet/lipgloss), [oto](https://github.com/ebitengine/oto), [bbolt](https://go.etcd.io/bbolt), and [godbus](https://github.com/godbus/dbus).
