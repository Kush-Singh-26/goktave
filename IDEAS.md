# GoKtave Future Feature Ideas

This document tracks potential enhancements and features for the GoKtave music player.

## High Priority (UX & Polish)

### 1. Gapless / Pre-emptive Extraction
- **Description**: Currently, there is a 2-3 second silence between tracks while the next one is extracted.
- **Idea**: Trigger `yt-dlp` extraction when the current track is at ~90% progress. Store the URL in memory so the transition is instant.

### 2. Advanced Queue Management
- **Keybinds**:
    - `x`: Remove selected track from queue.
    - `c`: Clear entire queue.
    - `j/k`: Move tracks up/down in the "Up Next" list.
- **Logic**: Allow users to prune the auto-generated radio results.

### 3. MPRIS2 Integration (Linux Desktop Standard)
- **Description**: Expose GoKtave to the system D-Bus.
- **Benefit**: Enables keyboard media keys (Play/Pause/Next), GNOME/KDE media widgets, and integration with phone-to-PC control apps (like KDE Connect).

## Search & Discovery

### 4. Search Autocomplete
- **Description**: Fetch suggestions from the `https://music.youtube.com/youtubei/v1/music/get_search_suggestions` endpoint as the user types.
- **UI**: Show top 3 suggestions as ghost text or a small dropdown.

### 5. "I'm Feeling Lucky" / Mood Radio
- **Description**: Start a radio session based on a mood (e.g., "Chill", "Workout", "Focus") instead of a specific song.

### 6. Lyrics Support
- **Description**: Fetch lyrics from the YouTube Music `Lyrics` tab (which we already see in the `/next` response keys).
- **UI**: A dedicated view (toggle with `l`) showing scrolling text.

## Performance & Audio

### 7. Persistent Local Cache
- **Description**: Save decoded PCM or the raw downloaded stream to a local cache directory (e.g., `~/.cache/goktave/`).
- **Benefit**: Zero-bandwidth replay of your favorite tracks and instant startup.

### 8. Volume Normalization
- **Description**: Analyze the peak volume of the upcoming stream and adjust the gain so that a quiet acoustic song and a loud rock song play at the same perceived level.

### 9. Visualizer (FFT)
- **Description**: Perform a Fast Fourier Transform on the PCM buffer.
- **UI**: Add a reactive 8-16 bar spectrum visualizer in the TUI.

## Integration & Social

### 10. Discord Rich Presence
- **Description**: Update your Discord status with "Listening to [Song] on GoKtave".

### 11. Playback History
- **Description**: Save a local log of played tracks to allow "Back" functionality and "Top Played" stats.
