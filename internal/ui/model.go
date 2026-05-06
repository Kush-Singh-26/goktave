package ui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

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
	isPaused	bool

	prov   provider.Provider
	ext    extractor.Extractor
	player *player.Player

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

	return Model{
		input:  ti,
		prov:   prov,
		ext:    ext,
		player: pl,
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

		case "space":
			if m.playingTrack != nil {
				m.isPaused = m.player.TogglePause()
				if m.isPaused {
					m.statusMsg = "|| Paused : " + m.playingTrack.Title
				} else {
					m.statusMsg = "|> Now Playing : " + m.playingTrack.Title
				}
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
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollOffset {
					m.scrollOffset = m.cursor
				}
			}

		case "down":
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

		case "enter":
			if m.listFocused && len(m.results) > 0 {
				selected := m.results[m.cursor]
				m.statusMsg = "Extracting audio for : " + selected.Title + "..."
				m.playingTrack = &selected
				
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
		return m, playCmd(m.player, msg.URL)

	case PlaybackMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.statusMsg = ""
		} else {
			m.statusMsg = "|> Now Playing : " + m.playingTrack.Title + " . " + m.playingTrack.Artist
			// Start waiting for track to end
			return m, waitForEndCmd(m.player, m.playingTrack.VideoID)
		}
		return m, nil

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
			}
		}
		return m, nil
	}

	// Always update the text input component
	m.input, cmd = m.input.Update(msg)
	return m, cmd
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

	s += "\n" + StyleMuted.Render("Tab to switch focus • Enter to search/play • Esc to quit")
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
