package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	progress "charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type tickMsg time.Time

type SearchResultsMsg struct {
	Results []provider.Track
	Err     error
}

type Model struct {
	input   textinput.Model
	results []provider.Track
	cursor  int
	loading bool
	err     error

	listFocused  bool
	playingTrack *provider.Track
	statusMsg    string
	isPaused     bool

	progress progress.Model
	elapsed  time.Duration
	lastTick time.Time
	queue    []provider.Track
	prov     provider.Provider
	ext      extractor.Extractor
	player   *player.Player

	cancel context.CancelFunc

	terminalHeight int
	scrollOffset   int
}

func NewModel(prov provider.Provider, ext extractor.Extractor, pl *player.Player) Model {
	ti := textinput.New()
	ti.Placeholder = "Search YouTube Music..."
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(50)

	prog := progress.New()
	prog.SetWidth(50)

	return Model{
		input:    ti,
		progress: prog,
		prov:     prov,
		ext:      ext,
		player:   pl,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) cancelRunningTask() {
	if m.cancel != nil {
		m.cancel()
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		return m, nil

	// Handle Keystrokes
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelRunningTask()
			return m, tea.Quit

		case "+", "=":
			if m.listFocused {
				if m.player != nil {
					v := m.player.GetVolume()
					m.player.SetVolume(v + 0.1)
					m.statusMsg = fmt.Sprintf("Volume : %d", int(m.player.GetVolume()*100))
				}
				return m, nil
			}
		case "-":
			if m.listFocused {
				if m.player != nil {
					v := m.player.GetVolume()
					m.player.SetVolume(v - 0.1)
					m.statusMsg = fmt.Sprintf("Volume : %d", int(m.player.GetVolume()*100))
				}
				return m, nil
			}
		case "n":
			if m.listFocused {
				if m.playingTrack != nil {
					m.player.Stop()
					return m, func() tea.Msg {
						return PlaybackEndedMsg{VideoID: m.playingTrack.VideoID}
					}
				}
				return m, nil
			}
		case "space":
			if m.listFocused {
				if m.playingTrack != nil {
					m.isPaused = m.player.TogglePause()
					if m.isPaused {
						m.statusMsg = "|| Paused : " + m.playingTrack.Title
					} else {
						m.lastTick = time.Now()
						m.statusMsg = "|> Now Playing : " + m.playingTrack.Title
					}
				}
				return m, nil
			}

		case "q":
			if m.listFocused {
				if len(m.results) > 0 {
					selected := m.results[m.cursor]
					m.queue = append(m.queue, selected)
					m.statusMsg = "Added to Queue : " + selected.Title
				}
				return m, nil
			}

		case "tab":
			m.listFocused = !m.listFocused
			if m.listFocused {
				m.input.Blur()
			} else {
				m.input.Focus()
			}
			return m, nil

		case "up":
			if m.listFocused {
				if m.cursor > 0 {
					m.cursor--
					if m.cursor < m.scrollOffset {
						m.scrollOffset = m.cursor
					}
				}
				return m, nil
			}

		case "down":
			if m.listFocused {
				if m.cursor < len(m.results)-1 {
					m.cursor++
					// Calculate visible items. Search box (4) + Status (1) + Footer (2) = ~7 lines
					visibleHeight := m.terminalHeight - 10
					if visibleHeight < 1 {
						visibleHeight = 1
					}
					if m.cursor >= m.scrollOffset+visibleHeight {
						m.scrollOffset = m.cursor - visibleHeight + 1
					}
				}
				return m, nil
			}

		case "enter":
			if m.listFocused && len(m.results) > 0 {
				selected := m.results[m.cursor]
				m.statusMsg = "Extracting audio for : " + selected.Title + "..."
				m.playingTrack = &selected
				m.elapsed = 0
				m.progress.SetPercent(0)

				m.cancelRunningTask()
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel

				return m, extractCmd(ctx, m.ext, selected.VideoID)
			} else if !m.listFocused && m.input.Value() != "" {
				m.loading = true
				m.err = nil
				m.results = nil
				m.statusMsg = ""
				m.scrollOffset = 0

				m.cancelRunningTask()
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel

				return m, searchCmd(ctx, m.prov, m.input.Value())
			}
		}

	// Handle Background Search Completion
	case SearchResultsMsg:
		m.loading = false
		if msg.Err != nil {
			if strings.Contains(msg.Err.Error(), "context canceled") {
				return m, nil
			}
			m.err = msg.Err
			return m, nil
		}
		m.results = msg.Results
		m.cursor = 0
		m.scrollOffset = 0
		if len(m.results) == 0 {
			m.statusMsg = "No results found for '" + m.input.Value() + "'"
		}
		return m, nil

	case StreamReadyMsg:
		if msg.Err != nil {
			if strings.Contains(msg.Err.Error(), "context canceled") {
				return m, nil
			}
			m.err = msg.Err
			m.statusMsg = ""
			return m, nil
		}
		m.statusMsg = "Buffering Audio..."
		m.elapsed = 0
		return m, playCmd(m.player, msg.URL)

	case PlaybackMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.statusMsg = ""
		} else if m.playingTrack != nil {
			m.statusMsg = "|> Now Playing : " + m.playingTrack.Title + " . " + m.playingTrack.Artist
			m.lastTick = time.Now()
			// Start waiting for track to end and start the progress tick
			return m, tea.Batch(tickCmd(), waitForEndCmd(m.player, m.playingTrack.VideoID))
		}
		return m, nil

	case tickMsg:
		if m.playingTrack == nil {
			return m, nil // Don't tick if nothing is playing
		}

		// Only advance the timer if the music is NOT paused
		if !m.isPaused {
			now := time.Now()
			m.elapsed += now.Sub(m.lastTick)
			m.lastTick = now
		}

		// Calculate the percentage (elapsed / total duration)
		pct := float64(m.elapsed.Seconds()) / float64(m.playingTrack.Duration)
		if pct > 1.0 {
			pct = 1.0
		}

		// Set the percentage on the progress bar component
		cmd := m.progress.SetPercent(pct)

		// Return a Batch containing the progress bar's internal update command,
		// AND our custom tickCmd to keep the infinite loop going.
		return m, tea.Batch(tickCmd(), cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel
		return m, cmd

	case PlaybackEndedMsg:
		if msg.Err != nil && !strings.Contains(msg.Err.Error(), "signal: killed") && !strings.Contains(msg.Err.Error(), "context canceled") {
			// Resiliency: If playback fails due to network/transient error, retry extracting & playing
			if m.playingTrack != nil && m.playingTrack.VideoID == msg.VideoID {
				m.statusMsg = "Connection lost, retrying..."
				m.cancelRunningTask()
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel
				return m, extractCmd(ctx, m.ext, m.playingTrack.VideoID)
			}
			m.err = msg.Err
			m.statusMsg = ""
		} else {
			// Track ended naturally or was stopped by us
			if msg.Err == nil && m.playingTrack != nil && m.playingTrack.VideoID == msg.VideoID {
				m.statusMsg = "Finished: " + m.playingTrack.Title
				m.playingTrack = nil
				m.elapsed = 0
				m.progress.SetPercent(0)

				if len(m.queue) > 0 {
					nextTrack := m.queue[0]
					m.queue = m.queue[1:]

					m.statusMsg = "Extracting next track : " + nextTrack.Title + "..."
					m.playingTrack = &nextTrack

					m.cancelRunningTask()
					ctx, cancel := context.WithCancel(context.Background())
					m.cancel = cancel
					return m, extractCmd(ctx, m.ext, nextTrack.VideoID)
				}
			}
		}
		return m, nil
	}

	// Always update the text input component
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func formatDuration(totalSeconds int) string {
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func (m Model) View() tea.View {
	// 1. Render the Search Box
	s := StyleTitle.Render("GoKtave") + "\n\n"
	if m.listFocused {
		s += StyleInput.Copy().BorderForeground(lipgloss.Color("#444444")).Render(m.input.View()) + "\n\n"
	} else {
		s += StyleInput.Copy().BorderForeground(lipgloss.Color("#e94560")).Render(m.input.View()) + "\n\n"
	}

	// 2. Render Status
	if m.loading {
		s += StyleMuted.Render("Searching YouTube Music...") + "\n"
	} else if m.err != nil {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171")).Render("Error: "+m.err.Error()) + "\n"
	}

	// 3. Render Results (with scrolling)
	visibleHeight := m.terminalHeight - 10
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	end := m.scrollOffset + visibleHeight
	if end > len(m.results) {
		end = len(m.results)
	}

	for i := m.scrollOffset; i < end; i++ {
		track := m.results[i]
		cursor := "  " // unselected
		trackStr := fmt.Sprintf("%s • %s", track.Title, track.Artist)

		if m.cursor == i {
			if m.listFocused {
				cursor = StyleTitle.Render("▶ ")
				s += cursor + StyleSelected.Render(trackStr) + "\n"
			} else {
				cursor = StyleMuted.Render("▶ ")
				s += cursor + StyleNormal.Render(trackStr) + "\n"
			}
		} else {
			s += cursor + StyleNormal.Render(trackStr) + "\n"
		}
	}

	if m.statusMsg != "" {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80")).Bold(true).Render(m.statusMsg) + "\n"
	}

	if m.playingTrack != nil {
		// Format the times (e.g., "1:05 / 3:45")
		elapsedStr := formatDuration(int(m.elapsed.Seconds()))
		totalStr := formatDuration(m.playingTrack.Duration)

		s += "\n" + m.progress.View() + "\n"
		s += StyleMuted.Render(fmt.Sprintf("%s / %s", elapsedStr, totalStr)) + "\n"
	}
	if len(m.queue) > 0 {
		s += "\n" + StyleTitle.Render(fmt.Sprintf("📋 Up Next: %d tracks in queue", len(m.queue))) + "\n"
	}
	s += "\n" + StyleMuted.Render("Tab: focus • Enter: play • Space: pause • n: next • +/-: volume • q: queue • Esc: quit")
	
	return tea.View{
		Content:   s,
		AltScreen: true,
	}
}

// searchCmd runs the HTTP request in a goroutine and returns a message
func searchCmd(ctx context.Context, prov provider.Provider, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := prov.Search(ctx, query)
		return SearchResultsMsg{Results: results, Err: err}
	}
}

// --- NEW MESSAGES ---
type StreamReadyMsg struct {
	URL string
	Err error
}

type PlaybackMsg struct {
	Err error
}

type PlaybackEndedMsg struct {
	VideoID string
	Err     error
}

func extractCmd(ctx context.Context, ext extractor.Extractor, videoID string) tea.Cmd {
	return func() tea.Msg {
		info, err := ext.Extract(ctx, videoID)
		if err != nil {
			return StreamReadyMsg{Err: err}
		}
		return StreamReadyMsg{URL: info.URL}
	}
}

func playCmd(pl *player.Player, url string) tea.Cmd {
	return func() tea.Msg {
		err := pl.Play(url)
		return PlaybackMsg{Err: err}
	}
}

func waitForEndCmd(pl *player.Player, videoID string) tea.Cmd {
	return func() tea.Msg {
		err := pl.Wait()
		return PlaybackEndedMsg{VideoID: videoID, Err: err}
	}
}
