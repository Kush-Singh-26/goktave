package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/Kush-Singh-26/goktave/internal/engine"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type FocusArea int

const (
	AreaSearch FocusArea = iota
	AreaQueue
	AreaContent
)

type ContentTab int

const (
	TabResults ContentTab = iota
	TabLyrics
	TabLibrary
)

type Model struct {
	engine engine.Engine

	focusArea FocusArea
	activeTab ContentTab

	search  textinput.Model
	results ResultsList
	queue   QueueView
	lyrics  viewport.Model

	statusBar StatusBar
	help      help.Model

	terminalWidth  int
	terminalHeight int

	err     error
	loading bool

	lastTrackID    string
	lastState      player.State
	lastLyricsText string

	cancel context.CancelFunc
}

func NewModel(e engine.Engine) Model {
	ti := textinput.New()
	ti.Placeholder = "Search songs, artists..."
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(30)

	h := help.New()
	h.Styles.ShortKey = StyleTitle
	h.Styles.ShortDesc = StyleMeta
	h.Styles.FullKey = StyleTitle
	h.Styles.FullDesc = StyleMeta

	return Model{
		engine:    e,
		focusArea: AreaSearch,
		activeTab: TabResults,
		search:    ti,
		results:   NewResultsList(),
		queue:     QueueView{},
		lyrics:    viewport.New(),
		statusBar: NewStatusBar(),
		help:      h,
		lastState: -1,
	}
}

type SearchResultsMsg struct {
	Results []provider.Track
	Err     error
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickCmd())
}

func (m Model) cancelRunningTask() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.search.SetWidth(msg.Width/2 - 10)
		m.lyrics.SetWidth(m.terminalWidth - int(float64(m.terminalWidth-2)*0.3) - 4)
		m.lyrics.SetHeight(m.getVisibleHeight() - 2)

	case tea.KeyMsg:
		// 1. Absolute Global Keys (Always work)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focusArea = (m.focusArea + 1) % 3
			if m.focusArea == AreaSearch {
				m.search.Focus()
				m.queue.Blur()
				m.results.Blur()
			} else if m.focusArea == AreaQueue {
				m.search.Blur()
				m.queue.Focus()
				m.results.Blur()
			} else {
				m.search.Blur()
				m.queue.Blur()
				m.results.Focus()
			}
			return m, nil
		}

		// 2. Focused Search Bar specific logic
		if m.search.Focused() {
			switch msg.String() {
			case "esc":
				m.search.Blur()
				m.focusArea = AreaContent
				m.results.Focus()
				return m, nil
			case "enter":
				if m.search.Value() != "" {
					m.loading = true
					m.err = nil
					m.statusBar.SetStatus("Searching...")
					m.search.Blur()
					m.focusArea = AreaContent
					m.cancelRunningTask()
					ctx, cancel := context.WithCancel(context.Background())
					m.cancel = cancel

					return m, func() tea.Msg {
						res, err := m.engine.Search(ctx, m.search.Value())
						return SearchResultsMsg{Results: res, Err: err}
					}
				}
			}
			// Let textinput handle all other keys
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

		// 3. Navigation & Jump keys (Only when NOT typing)
		switch msg.String() {
		case "q":
			m.focusArea = AreaQueue
			m.search.Blur()
			m.queue.Focus()
			m.results.Blur()
			return m, nil
		case "s", "/":
			m.focusArea = AreaSearch
			m.search.Focus()
			m.queue.Blur()
			m.results.Blur()
			return m, nil
		case "1":
			m.activeTab = TabResults
			m.focusArea = AreaContent
			m.search.Blur()
			m.queue.Blur()
			m.results.Focus()
			return m, nil
		case "2":
			m.activeTab = TabLyrics
			m.focusArea = AreaContent
			m.search.Blur()
			m.queue.Blur()
			m.results.Blur()
			return m, nil
		case "3":
			m.activeTab = TabLibrary
			m.focusArea = AreaContent
			m.search.Blur()
			m.queue.Blur()
			m.results.Blur()
			return m, nil

		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case "up", "k":
			if m.focusArea == AreaContent && m.activeTab == TabResults {
				m.results.Prev()
			} else if m.focusArea == AreaQueue {
				if m.queue.Cursor > 0 {
					m.queue.Cursor--
				}
			} else if m.focusArea == AreaContent && m.activeTab == TabLyrics {
				m.lyrics.ScrollUp(1)
			}
			return m, nil

		case "down", "j":
			if m.focusArea == AreaContent && m.activeTab == TabResults {
				m.results.Next()
			} else if m.focusArea == AreaQueue {
				if m.queue.Cursor < len(m.engine.GetQueue())-1 {
					m.queue.Cursor++
				}
			} else if m.focusArea == AreaContent && m.activeTab == TabLyrics {
				m.lyrics.ScrollDown(1)
			}
			return m, nil

		case "K": // Move Up in Queue
			if m.focusArea == AreaQueue {
				if m.queue.Cursor > 0 {
					m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor-1)
					m.queue.Cursor--
				}
				return m, nil
			}
		case "J": // Move Down in Queue
			if m.focusArea == AreaQueue {
				queueLen := len(m.engine.GetQueue())
				if m.queue.Cursor < queueLen-1 {
					m.engine.MoveInQueue(m.queue.Cursor, m.queue.Cursor+1)
					m.queue.Cursor++
				}
				return m, nil
			}
		case "c": // Clear Queue
			if m.focusArea == AreaQueue {
				m.engine.ClearQueue()
				m.queue.Cursor = 0
				m.statusBar.SetStatus("Queue cleared")
				return m, nil
			}
		case "a": // Add to Queue
			if m.focusArea == AreaContent && m.activeTab == TabResults {
				selected := m.results.GetSelected()
				if selected != nil {
					m.engine.Queue(*selected)
					m.statusBar.SetStatus("Added to queue: " + selected.Title)
				}
				return m, nil
			}

		case "space":
			paused := m.engine.TogglePause()
			track := m.engine.GetCurrentTrack()
			if track != nil {
				if paused {
					m.statusBar.SetStatus("|| Paused : " + track.Title)
				} else {
					m.statusBar.SetStatus("|> Playing : " + track.Title)
				}
			} else {
				m.statusBar.SetStatus("No track to pause/play")
			}
			return m, nil

		case "n":
			m.statusBar.SetStatus("Skipping...")
			m.engine.Next()
			return m, nil

		case "x":
			if m.focusArea == AreaQueue {
				m.engine.RemoveFromQueue(m.queue.Cursor)
				queueLen := len(m.engine.GetQueue())
				if m.queue.Cursor >= queueLen && queueLen > 0 {
					m.queue.Cursor = queueLen - 1
				} else if queueLen == 0 {
					m.queue.Cursor = 0
				}
				return m, nil
			}

		case "enter":
			if m.focusArea == AreaQueue {
				queue := m.engine.GetQueue()
				if m.queue.Cursor >= 0 && m.queue.Cursor < len(queue) {
					m.statusBar.SetStatus("Playing from queue...")
					if err := m.engine.PlayFromQueue(m.queue.Cursor); err != nil {
						m.err = err
					}
					return m, nil
				}
			} else if m.focusArea == AreaContent && m.activeTab == TabResults {
				selected := m.results.GetSelected()
				if selected != nil {
					m.statusBar.SetStatus("Playing: " + selected.Title + "...")
					if err := m.engine.Play(*selected); err != nil {
						m.err = err
					}
					return m, nil
				}
			}
		}

	case SearchResultsMsg:
		m.loading = false
		if msg.Err != nil {
			if !strings.Contains(msg.Err.Error(), "context canceled") {
				m.err = msg.Err
				m.statusBar.SetStatus("Search Error: " + msg.Err.Error())
			}
		} else {
			m.results.SetTracks(msg.Results)
			m.statusBar.SetStatus(fmt.Sprintf("Found %d results", len(msg.Results)))
			if len(msg.Results) == 0 {
				m.statusBar.SetStatus("No results found for '" + m.search.Value() + "'")
			} else {
				// Auto-focus results
				m.focusArea = AreaContent
				m.activeTab = TabResults
				m.search.Blur()
				m.results.Focus()
			}
		}

	case tickMsg:
		// Update lyrics only if they changed
		lyricsText := m.engine.GetLyrics()
		if lyricsText != m.lastLyricsText {
			m.lastLyricsText = lyricsText
			m.lyrics.SetContent(lyricsText)
		}

		track := m.engine.GetCurrentTrack()
		if track != nil {
			state := m.engine.GetState()

			if track.VideoID != m.lastTrackID {
				m.lastTrackID = track.VideoID
				m.statusBar.elapsed = 0
				m.statusBar.lastTick = time.Now()
				m.lastState = -1
			}

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

			if state != m.lastState {
				m.lastState = state
				switch state {
				case player.StateBuffering:
					m.statusBar.SetStatus("Buffering : " + track.Title + "...")
				case player.StatePlaying:
					m.statusBar.SetStatus("|> Now Playing : " + track.Title)
				case player.StatePaused:
					m.statusBar.SetStatus("|| Paused : " + track.Title)
				}
			}

			if pct > 0.90 && !m.engine.IsPreloading() {
				m.engine.Preload()
			}
			cmds = append(cmds, m.statusBar.progress.SetPercent(pct))
		}

		return m, tea.Batch(append(cmds, tickCmd())...)
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)

	m.results, cmd = m.results.Update(msg, m.getVisibleHeight())
	m.results.SyncScroll(m.getVisibleHeight())
	m.queue.SyncScroll(m.getVisibleHeight())
	cmds = append(cmds, cmd)

	m.statusBar, cmd = m.statusBar.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) getVisibleHeight() int {
	// Estimate for update logic (sync with View math)
	h := 14
	if m.help.ShowAll {
		h = 20
	}
	vh := m.terminalHeight - h
	if vh < 2 {
		vh = 2
	}
	return vh
}

func (m Model) View() tea.View {
	if m.terminalWidth < 20 || m.terminalHeight < 10 {
		return tea.View{Content: "Terminal too small"}
	}

	branding := "█▀▀▀█  █▀▀▀█  █  ▄▀  ▀▀█▀▀  █▀▀▀█  █   █  █▀▀▀█\n" +
		"█ ▀▀█  █   █  █▀▀▄     █    █▀▀▀█  █   █  █▀▀▀ \n" +
		"▀▀▀▀▀  ▀▀▀▀▀  ▀  ▀     ▀    ▀   ▀   ▀▀▀   ▀▀▀▀▀"

	brandingWidth := int(float64(m.terminalWidth-2) * 0.5)
	searchWidth := (m.terminalWidth - 2) - brandingWidth

	brandingBox := HeaderStyle.Copy().
		Width(brandingWidth).
		Height(7).
		Foreground(Terracotta).
		Padding(1, 0, 1, 2).
		Render(branding)

	searchBoxStyle := HeaderStyle.Copy().
		Width(searchWidth).
		Height(7).
		Padding(2, 2, 2, 2).
		Align(lipgloss.Right, lipgloss.Center)

	if m.focusArea == AreaSearch {
		searchBoxStyle = searchBoxStyle.BorderForeground(Terracotta)
	}

	searchBox := searchBoxStyle.Render(m.search.View())
	header := lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, searchBox)

	// Unified Footer
	footerWidth := m.terminalWidth - 2
	innerFooterWidth := footerWidth - 4
	playbarWidth := int(float64(innerFooterWidth) * 0.35)
	vizWidth := innerFooterWidth - playbarWidth - 5
	if vizWidth < 10 {
		vizWidth = 10
	}

	playbarView := m.statusBar.PlaybarView(m.engine.GetCurrentTrack(), m.engine.GetState() == player.StatePlaying, playbarWidth)
	playbarPane := lipgloss.NewStyle().
		Width(playbarWidth).
		MaxWidth(playbarWidth).
		Height(4).
		Align(lipgloss.Left, lipgloss.Center).
		Render(playbarView)

	vizView := m.statusBar.VisualizerView(m.engine.GetVisualizerBars(vizWidth), vizWidth, 4)
	separator := lipgloss.NewStyle().
		Foreground(BorderMid).
		Height(4).
		Width(1).
		Align(lipgloss.Center, lipgloss.Center).
		Render("│\n│\n│\n│")

	footerContent := lipgloss.JoinHorizontal(lipgloss.Top,
		playbarPane,
		"  ",
		separator,
		"  ",
		vizView,
	)

	footer := FooterStyle.Copy().
		Width(footerWidth).
		Height(6).
		Render(footerContent)

	helpHeight := 1
	if m.help.ShowAll {
		helpHeight = 7
	}
	helpView := lipgloss.NewStyle().
		Width(m.terminalWidth).
		Height(helpHeight).
		Render("  " + m.help.View(Keys))

	// Precise dynamic height calculation for the body
	hUsed := lipgloss.Height(header) + lipgloss.Height(footer) + lipgloss.Height(helpView)
	visibleHeight := m.terminalHeight - hUsed
	if visibleHeight < 2 {
		visibleHeight = 2
	}

	contentWidth := m.terminalWidth - 4
	queueWidth := int(float64(contentWidth) * 0.3)
	mainWidth := contentWidth - queueWidth

	queueStyle := PaneStyle.Copy().
		Width(queueWidth).
		Height(visibleHeight)
	if m.focusArea == AreaQueue {
		queueStyle = ActivePaneStyle.Copy().
			Width(queueWidth).
			Height(visibleHeight)
	}
	queuePane := queueStyle.Render(m.queue.View(m.engine.GetQueue(), visibleHeight-2, queueWidth))

	tabs := []string{"[1] 📜 Results", "[2] 📝 Lyrics", "[3] 📁 Library"}
	tabRow := ""
	for i, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if int(m.activeTab) == i {
			style = style.Foreground(Terracotta).Bold(true).Underline(true)
		}
		tabRow += style.Render(t)
	}
	tabRow = lipgloss.NewStyle().
		Width(mainWidth - 2).
		Render(tabRow)

	var contentBody string
	resultsHeight := visibleHeight - 2
	if resultsHeight < 0 {
		resultsHeight = 0
	}

	switch m.activeTab {
	case TabResults:
		contentBody = m.results.View(resultsHeight-1, mainWidth)
	case TabLyrics:
		contentBody = m.lyrics.View()
	case TabLibrary:
		contentBody = "\n\n  " + StyleMeta.Render("Library coming soon...")
	}

	contentStyle := PaneStyle.Copy().
		Width(mainWidth).
		Height(visibleHeight)
	if m.focusArea == AreaContent {
		contentStyle = ActivePaneStyle.Copy().
			Width(mainWidth).
			Height(visibleHeight)
	}
	contentPane := contentStyle.Render(tabRow + "\n" + contentBody)

	body := lipgloss.JoinHorizontal(lipgloss.Top, queuePane, contentPane)

	fullView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		footer,
		helpView,
	)

	return tea.View{
		Content:   fullView,
		AltScreen: true,
	}
}

// ResultsList focus helper
const (
	AreaResults = AreaContent
)
