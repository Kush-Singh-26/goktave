package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Kush-Singh-26/goktave/internal/engine"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type tickMsg time.Time

type SearchResultsMsg struct {
	Results []provider.Track
	Err     error
}

type Model struct {
	search    SearchBar
	results   ResultsList
	statusBar StatusBar
	queue     QueueView

	loading bool
	err     error

	engine engine.Engine

	cancel context.CancelFunc

	lastTrackID string
	lastState   player.State

	terminalHeight int
}

func NewModel(eng engine.Engine) Model {
	return Model{
		search:    NewSearchBar(),
		results:   NewResultsList(),
		statusBar: NewStatusBar(),
		engine:    eng,
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m *Model) cancelRunningTask() {
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
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			if m.queue.Focused {
				if m.queue.Cursor < len(m.engine.GetQueue())-1 {
					m.queue.Cursor++
				}
			}
		case "k":
			if m.queue.Focused {
				if m.queue.Cursor > 0 {
					m.queue.Cursor--
				}
			}
		case "J":
			if m.queue.Focused {
				if m.queue.Cursor < len(m.engine.GetQueue())-1 {
					m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor+1)
					m.queue.Cursor++
				}
			}
		case "K":
			if m.queue.Focused {
				if m.queue.Cursor > 0 {
					m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor-1)
					m.queue.Cursor--
				}
			}
		case "x":
			if m.queue.Focused {
				m.engine.RemoveFromQueue(m.queue.Cursor)
				if m.queue.Cursor >= len(m.engine.GetQueue()) && m.queue.Cursor > 0 {
					m.queue.Cursor--
				}
			}
		case "c":
			if m.queue.Focused {
				m.engine.ClearQueue()
				m.queue.Cursor = 0
			}

		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			if m.search.focused {
				m.search.Blur()
				m.results.Focus()
				m.queue.Focused = false
			} else if m.results.focused {
				m.results.Blur()
				m.queue.Focused = true
			} else {
				m.queue.Focused = false
				m.search.Focus()
			}
			return m, nil

		case "enter":
			if m.search.focused && m.search.Value() != "" {
				m.loading = true
				m.err = nil
				m.statusBar.SetStatus("")
				m.cancelRunningTask()
				ctx, cancel := context.WithCancel(context.Background())
				m.cancel = cancel

				return m, func() tea.Msg {
					res, err := m.engine.Search(ctx, m.search.Value())
					return SearchResultsMsg{Results: res, Err: err}
				}
			} else if m.queue.Focused {
				queue := m.engine.GetQueue()
				if m.queue.Cursor >= 0 && m.queue.Cursor < len(queue) {
					track := queue[m.queue.Cursor]
					m.statusBar.SetStatus("Extracting audio for : " + track.Title + "...")
					m.statusBar.elapsed = 0
					m.statusBar.progress.SetPercent(0)
					if err := m.engine.PlayFromQueue(m.queue.Cursor); err != nil {
						m.err = err
					} else {
						m.statusBar.lastTick = time.Now()
						return m, nil
					}
				}
			} else if m.results.focused {
				selected := m.results.GetSelected()
				if selected != nil {
					m.statusBar.SetStatus("Extracting audio for : " + selected.Title + "...")
					m.statusBar.elapsed = 0
					m.statusBar.progress.SetPercent(0)
					if err := m.engine.Play(*selected); err != nil {
						m.err = err
					} else {
						m.statusBar.lastTick = time.Now()
						return m, nil
					}
				}
			}

		case "n":
			if !m.search.focused {
				m.engine.Next()
				m.statusBar.elapsed = 0
			}
		case "space":
			if !m.search.focused {
				isPaused := m.engine.TogglePause()
				track := m.engine.GetCurrentTrack()
				if track != nil {
					if isPaused {
						m.statusBar.SetStatus("|| Paused : " + track.Title)
					} else {
						m.statusBar.lastTick = time.Now()
						m.statusBar.SetStatus("|> Now Playing : " + track.Title)
					}
				}
			}
		case "q":
			if m.results.focused {
				selected := m.results.GetSelected()
				if selected != nil {
					m.engine.Queue(*selected)
					m.statusBar.SetStatus("Added to Queue : " + selected.Title)
				}
			}
		case "+", "=":
			v := m.engine.GetVolume()
			m.engine.SetVolume(v + 0.1)
			m.statusBar.SetStatus(fmt.Sprintf("Volume : %d", int(m.engine.GetVolume()*100)))
		case "-":
			v := m.engine.GetVolume()
			m.engine.SetVolume(v - 0.1)
			m.statusBar.SetStatus(fmt.Sprintf("Volume : %d", int(m.engine.GetVolume()*100)))
		}

	case SearchResultsMsg:
		m.loading = false
		if msg.Err != nil {
			if !strings.Contains(msg.Err.Error(), "context canceled") {
				m.err = msg.Err
			}
		} else {
			m.results.SetTracks(msg.Results)
			if len(msg.Results) == 0 {
				m.statusBar.SetStatus("No results found for '" + m.search.Value() + "'")
			}
		}

	case tickMsg:
		track := m.engine.GetCurrentTrack()
		if track != nil {
			state := m.engine.GetState()

			// 1. Sync Track Change (Handles auto-advance)
			if track.VideoID != m.lastTrackID {
				m.lastTrackID = track.VideoID
				m.statusBar.elapsed = 0
				m.statusBar.lastTick = time.Now()

				// Force a state sync for the new track
				m.lastState = -1
			}

			// 2. Update Progress
			if state == player.StatePlaying {
				now := time.Now()
				m.statusBar.elapsed += now.Sub(m.statusBar.lastTick)
				m.statusBar.lastTick = now
			} else {
				m.statusBar.lastTick = time.Now()
			}

			pct := 0.0
			if track.Duration > 0 {
				pct = float64(m.statusBar.elapsed.Seconds()) / float64(track.Duration)
			}
			if pct > 1.0 {
				pct = 1.0
			}

			// 3. Sync Status Message based on Engine State
			if state != m.lastState {
				m.lastState = state
				switch state {
				case player.StateBuffering:
					m.statusBar.SetStatus("Buffering : " + track.Title + "...")
				case player.StatePlaying:
					m.statusBar.SetStatus("|> Now Playing : " + track.Title)
					m.statusBar.lastTick = time.Now()
				case player.StatePaused:
					m.statusBar.SetStatus("|| Paused : " + track.Title)
				case player.StateStopped:
					if pct > 0.9 {
						m.statusBar.SetStatus("Finished : " + track.Title)
					}
				}
			}

			if pct > 0.90 && !m.engine.IsPreloading() {
				m.engine.Preload()
			}
			cmds = append(cmds, m.statusBar.progress.SetPercent(pct))
		}

		return m, tea.Batch(append(cmds, tickCmd())...)
	}

	// Update components
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)

	visibleHeight := m.getVisibleHeight()
	m.results, cmd = m.results.Update(msg, visibleHeight)
	cmds = append(cmds, cmd)

	m.statusBar, cmd = m.statusBar.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) getVisibleHeight() int {
	reservedLines := 10
	if m.statusBar.status != "" {
		reservedLines += 2
	}
	if m.engine.GetCurrentTrack() != nil {
		reservedLines += 4
	}
	queue := m.engine.GetQueue()
	if len(queue) > 0 {
		displayCount := 5
		if len(queue) < displayCount {
			displayCount = len(queue)
		}
		reservedLines += displayCount + 2
	}

	vh := m.terminalHeight - reservedLines
	if vh < 1 {
		return 1
	}
	return vh
}

func (m Model) View() tea.View {
	s := StyleTitle.Render("GoKtave") + "\n\n"
	s += m.search.View() + "\n\n"

	if m.loading {
		s += StyleMeta.Render("Searching YouTube Music...") + "\n"
	} else if m.err != nil {
		s += StyleSelected.Copy().Foreground(Danger).Render("Error: "+m.err.Error()) + "\n"
	}

	s += m.results.View(m.getVisibleHeight())
	s += m.statusBar.View(m.engine.GetCurrentTrack(), m.engine.GetState() == player.StatePlaying)
	s += m.queue.View(m.engine.GetQueue())

	s += "\n" + StyleMeta.Render("Tab: focus • Enter: play • Space: pause • j/k: navigate • n: next • +/-: volume • q: queue • Esc: quit")

	return tea.View{
		Content:   s,
		AltScreen: true,
	}
}
