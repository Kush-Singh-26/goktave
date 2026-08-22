package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/Kush-Singh-26/goktave/internal/player"
)

func (m Model) View() tea.View {
	if m.terminalWidth < 20 || m.terminalHeight < 10 {
		return tea.View{Content: "Terminal too small"}
	}

	branding := gradientText("▅▆██▆▅\ngoktave")

	searchWidth := 40
	suggestWidth := 50
	wide := m.terminalWidth >= 120
	narrow := m.terminalWidth < 80
	headerBoxHeight := 4

	if !wide {
		suggestWidth = 0
	}
	if wide {
		needed := 11 + searchWidth + suggestWidth + 4
		if m.terminalWidth < needed {
			searchWidth = int(float64(m.terminalWidth-15) * 0.55)
			suggestWidth = int(float64(m.terminalWidth-15) * 0.40)
		}
	}
	if searchWidth < 15 {
		searchWidth = 15
	}
	if suggestWidth < 10 && suggestWidth > 0 {
		suggestWidth = 10
	}

	searchBoxView := StyleMeta.Render("Search:") + "\n" + m.search.View()

	var header string
	if wide {
		brandingBox := HeaderStyle.Copy().
			Width(11).
			Height(headerBoxHeight).
			Foreground(Accent).
			Padding(0, 1).
			Render(branding)

		searchBoxStyle := HeaderStyle.Copy().
			Width(searchWidth).
			Height(headerBoxHeight).
			Padding(0, 1).
			Align(lipgloss.Left, lipgloss.Center)
		if m.focusArea == AreaSearch {
			searchBoxStyle = searchBoxStyle.BorderForeground(Accent)
		}
		searchBox := searchBoxStyle.Render(searchBoxView)

		suggestLines := m.headerSuggestLines(suggestWidth)
		suggestBoxView := strings.Join(suggestLines, "\n")
		suggestBox := HeaderStyle.Copy().
			Width(suggestWidth).
			Height(headerBoxHeight).
			Padding(0, 1).
			Align(lipgloss.Left, lipgloss.Center).
			Render(suggestBoxView)

		gap := "  "
		header = lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, gap, searchBox, suggestBox)
	} else if !narrow {
		brandingBox := HeaderStyle.Copy().
			Width(11).
			Height(headerBoxHeight).
			Foreground(Accent).
			Padding(0, 1).
			Render(branding)

		searchBoxStyle := HeaderStyle.Copy().
			Width(searchWidth).
			Height(headerBoxHeight).
			Padding(0, 1).
			Align(lipgloss.Left, lipgloss.Center)
		if m.focusArea == AreaSearch {
			searchBoxStyle = searchBoxStyle.BorderForeground(Accent)
		}
		searchBox := searchBoxStyle.Render(searchBoxView)

		gap := "  "
		header = lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, gap, searchBox)
	} else {
		narrowSearchWidth := m.terminalWidth - 6
		if narrowSearchWidth < 20 {
			narrowSearchWidth = 20
		}
		searchBoxStyle := HeaderStyle.Copy().
			Width(narrowSearchWidth).
			Height(headerBoxHeight).
			Padding(0, 1).
			Align(lipgloss.Left, lipgloss.Center)
		if m.focusArea == AreaSearch {
			searchBoxStyle = searchBoxStyle.BorderForeground(Accent)
		}
		header = searchBoxStyle.Render(searchBoxView)
	}

	// Help line
	helpHeight := 1
	if m.help.ShowAll && m.terminalHeight >= 20 {
		helpHeight = 7
	}
	helpView := lipgloss.NewStyle().
		Width(m.terminalWidth).
		Height(helpHeight).
		Render("  " + m.help.View(Keys))

	// Dynamic height for body
	headerHeight := lipgloss.Height(header)
	helpRenderedHeight := lipgloss.Height(helpView)
	visibleHeight := m.terminalHeight - headerHeight - helpRenderedHeight
	if visibleHeight < 2 {
		visibleHeight = 2
	}
	paneFrameV := PaneStyle.GetVerticalFrameSize()

	compactNPHeight := 0
	if !wide && m.engine.GetCurrentTrack() != nil {
		compactNPHeight = 3
	}

	vizOuterHeight := 4 + paneFrameV
	bodyOverhead := vizOuterHeight + compactNPHeight
	topHeightOuter := visibleHeight - bodyOverhead
	minPaneContent := 4
	if topHeightOuter < 1 || visibleHeight-bodyOverhead < minPaneContent {
		vizOuterHeight = 0
		topHeightOuter = visibleHeight - compactNPHeight
	}
	if topHeightOuter < 1 {
		topHeightOuter = 1
	}
	topInnerHeight := topHeightOuter - paneFrameV
	if topInnerHeight < 1 {
		topInnerHeight = 1
	}

	contentWidth := m.terminalWidth - 4

	var body string
	if wide {
		// 3-pane layout: Queue | Content | NowPlaying
		queueWidth := int(float64(contentWidth) * 0.20)
		mainWidth := int(float64(contentWidth) * 0.55)
		mainColumnWidth := contentWidth - queueWidth
		nowPlayingWidth := mainColumnWidth - mainWidth
		minQueueWidth := PaneStyle.GetHorizontalFrameSize() + 6
		minNowPlayingWidth := PaneStyle.GetHorizontalFrameSize() + 12
		minMainWidth := PaneStyle.GetHorizontalFrameSize() + 12
		if queueWidth < minQueueWidth {
			queueWidth = minQueueWidth
		}
		if mainWidth < minMainWidth {
			mainWidth = minMainWidth
		}
		if nowPlayingWidth < minNowPlayingWidth {
			nowPlayingWidth = minNowPlayingWidth
		}
		if queueWidth+mainWidth+nowPlayingWidth > contentWidth && contentWidth > 0 {
			available := contentWidth - queueWidth - minNowPlayingWidth
			if available > minMainWidth {
				mainWidth = available
				nowPlayingWidth = contentWidth - queueWidth - mainWidth
			} else {
				mainWidth = minMainWidth
				nowPlayingWidth = contentWidth - queueWidth - mainWidth
				if nowPlayingWidth < minNowPlayingWidth {
					nowPlayingWidth = minNowPlayingWidth
				}
			}
		}

		nowPlayingInnerWidth := nowPlayingWidth - PaneStyle.GetHorizontalFrameSize()
		if nowPlayingInnerWidth < 1 {
			nowPlayingInnerWidth = 1
		}

		queueStyle := PaneStyle.Copy().Width(queueWidth).Height(topInnerHeight)
		if m.focusArea == AreaQueue {
			queueStyle = ActivePaneStyle.Copy().Width(queueWidth).Height(topInnerHeight)
		}
		queueInnerWidth := queueWidth - PaneStyle.GetHorizontalFrameSize()
		if queueInnerWidth < 1 {
			queueInnerWidth = 1
		}
		queueContent := m.queue.View(m.engine.GetQueue(), topInnerHeight, queueInnerWidth, m.engine.IsLiked)
		queueContent = lipgloss.Place(queueInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, queueContent)
		queuePane := queueStyle.Render(queueContent)

		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		tabRow := m.buildTabRow(contentInnerWidth)
		contentInnerHeight := topInnerHeight - 2
		if contentInnerHeight < 1 {
			contentInnerHeight = 1
		}
		contentBody := m.renderContentBody(contentInnerHeight, contentInnerWidth)
		contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topInnerHeight)
		if m.focusArea == AreaContent {
			contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topInnerHeight)
		}
		contentText := tabRow + "\n" + contentBody
		contentText = lipgloss.Place(contentInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, contentText)
		contentPane := contentStyle.Render(contentText)

		nowPlayingContent := m.renderNowPlaying(topInnerHeight, nowPlayingInnerWidth)
		nowPlayingPaneStyle := PaneStyle.Copy().Width(nowPlayingWidth).Height(topInnerHeight)
		nowPlayingPane := nowPlayingPaneStyle.Render(nowPlayingContent)

		vizWidth := queueWidth + mainWidth
		vizInnerWidth := vizWidth - PaneStyle.GetHorizontalFrameSize()
		if vizInnerWidth < 1 {
			vizInnerWidth = 1
		}
		visualizerView := m.statusBar.VisualizerView(m.engine.GetVisualizerBars(vizInnerWidth), vizInnerWidth, 4)
		leftTopRow := lipgloss.JoinHorizontal(lipgloss.Top, queuePane, contentPane)
		leftColumn := leftTopRow
		if vizOuterHeight > 0 {
			visualizerPane := PaneStyle.Copy().
				Width(vizWidth).
				Height(4).
				Render(visualizerView)
			leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftTopRow, visualizerPane)
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, nowPlayingPane)
	} else if !narrow {
		// 2-pane layout: Queue | Content (no now-playing pane)
		queueWidth := int(float64(contentWidth) * 0.25)
		mainWidth := contentWidth - queueWidth
		minQueueWidth := PaneStyle.GetHorizontalFrameSize() + 6
		minMainWidth := PaneStyle.GetHorizontalFrameSize() + 12
		if queueWidth < minQueueWidth {
			queueWidth = minQueueWidth
			mainWidth = contentWidth - queueWidth
		}
		if mainWidth < minMainWidth {
			mainWidth = minMainWidth
			queueWidth = contentWidth - mainWidth
		}
		if queueWidth < minQueueWidth {
			queueWidth = minQueueWidth
			mainWidth = contentWidth - queueWidth
		}

		queueStyle := PaneStyle.Copy().Width(queueWidth).Height(topInnerHeight)
		if m.focusArea == AreaQueue {
			queueStyle = ActivePaneStyle.Copy().Width(queueWidth).Height(topInnerHeight)
		}
		queueInnerWidth := queueWidth - PaneStyle.GetHorizontalFrameSize()
		if queueInnerWidth < 1 {
			queueInnerWidth = 1
		}
		queueContent := m.queue.View(m.engine.GetQueue(), topInnerHeight, queueInnerWidth, m.engine.IsLiked)
		queueContent = lipgloss.Place(queueInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, queueContent)
		queuePane := queueStyle.Render(queueContent)

		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		tabRow := m.buildTabRow(contentInnerWidth)
		contentInnerHeight := topInnerHeight - 2
		if contentInnerHeight < 1 {
			contentInnerHeight = 1
		}
		contentBody := m.renderContentBody(contentInnerHeight, contentInnerWidth)
		contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topInnerHeight)
		if m.focusArea == AreaContent {
			contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topInnerHeight)
		}
		contentText := tabRow + "\n" + contentBody
		contentText = lipgloss.Place(contentInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, contentText)
		contentPane := contentStyle.Render(contentText)

		vizWidth := contentWidth
		vizInnerWidth := vizWidth - PaneStyle.GetHorizontalFrameSize()
		if vizInnerWidth < 1 {
			vizInnerWidth = 1
		}
		visualizerView := m.statusBar.VisualizerView(m.engine.GetVisualizerBars(vizInnerWidth), vizInnerWidth, 4)
		leftTopRow := lipgloss.JoinHorizontal(lipgloss.Top, queuePane, contentPane)
		leftColumn := leftTopRow
		if vizOuterHeight > 0 {
			visualizerPane := PaneStyle.Copy().
				Width(vizWidth).
				Height(4).
				Render(visualizerView)
			leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftTopRow, visualizerPane)
		}
		body = leftColumn
	} else {
		// 1-pane layout: Just Content
		mainWidth := contentWidth
		minMainWidth := PaneStyle.GetHorizontalFrameSize() + 12
		if mainWidth < minMainWidth {
			mainWidth = minMainWidth
		}

		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		tabRow := m.buildTabRow(contentInnerWidth)
		contentInnerHeight := topInnerHeight - 2
		if contentInnerHeight < 1 {
			contentInnerHeight = 1
		}
		contentBody := m.renderContentBody(contentInnerHeight, contentInnerWidth)
		contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topInnerHeight)
		if m.focusArea == AreaContent {
			contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topInnerHeight)
		}
		contentText := tabRow + "\n" + contentBody
		contentText = lipgloss.Place(contentInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, contentText)
		contentPane := contentStyle.Render(contentText)

		body = contentPane
	}

	// Compact now-playing in medium/narrow modes
	if !wide {
		npCompact := m.renderCompactNowPlaying(contentWidth - PaneStyle.GetHorizontalFrameSize())
		if npCompact != "" {
			npCompactView := PaneStyle.Copy().
				Width(contentWidth).
				Height(1).
				Render(npCompact)
			body = lipgloss.JoinVertical(lipgloss.Left, body, npCompactView)
		}
	}

	fullView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		helpView,
	)

	fullView = m.renderOverlays(fullView)

	return tea.View{
		Content:   fullView,
		AltScreen: true,
	}
}

func (m Model) headerSuggestLines(suggestWidth int) []string {
	var suggestLines []string
	if m.showSuggest {
		title := "Suggestions:"
		if m.search.Value() == "" {
			title = "Recent Searches:"
		}
		suggestLines = append(suggestLines, StyleMeta.Render(title))

		for i, s := range m.suggestions {
			if i >= 4 { // Only show 4 suggestions
				break
			}
			line := s
			if i == m.suggestionIndex {
				line = StyleSelected.Render("> " + s)
			} else {
				line = StyleMeta.Render("  " + s)
			}
			suggestLines = append(suggestLines, line)
		}
	} else if len(m.downloadProgress) > 0 {
		suggestLines = append(suggestLines, StyleTitle.Render("Active Downloads:"))
		keys := make([]string, 0, len(m.downloadProgress))
		for k := range m.downloadProgress {
			keys = append(keys, k)
		}
		// Show up to 4 downloads
		for i := 0; i < 4 && i < len(keys); i++ {
			title := keys[i]
			pct := m.downloadProgress[title]
			barWidth := suggestWidth - 15
			if barWidth < 5 {
				barWidth = 5
			}

			filled := int(float64(barWidth) * pct)
			empty := barWidth - filled
			if empty < 0 {
				empty = 0
			}

			bar := lipgloss.NewStyle().Foreground(Accent).Render(strings.Repeat("█", filled)) +
				lipgloss.NewStyle().Foreground(SurfaceDeep).Render(strings.Repeat("░", empty))

			name := title
			if len(name) > 15 {
				name = name[:12] + "..."
			}

			suggestLines = append(suggestLines, fmt.Sprintf("%s %s", StyleMeta.Render(name), bar))
		}
	}

	// Always ensure exactly 5 lines (1 title + 4 items) to prevent jumping
	for len(suggestLines) < 5 {
		if len(suggestLines) == 0 {
			suggestLines = append(suggestLines, StyleMeta.Render("Suggestions:"))
		} else {
			suggestLines = append(suggestLines, "")
		}
	}

	return suggestLines
}

func (m Model) buildTabRow(width int) string {
	tabs := []struct {
		name string
		key  string
	}{
		{"Search", "1"},
		{"Lyrics", "2"},
		{"Playlists", "3"},
		{"Liked", "4"},
		{"History", "5"},
		{"Downloads", "6"},
		{"Settings", "7"},
	}

	var renderedTabs []string
	for i, t := range tabs {
		var s string
		if int(m.activeTab) == i {
			s = lipgloss.NewStyle().
				Foreground(BgBase).
				Background(Accent).
				Bold(true).
				Padding(0, 1).
				Render(fmt.Sprintf("%s %s", t.key, t.name))
		} else {
			s = lipgloss.NewStyle().
				Foreground(FgMuted).
				Padding(0, 1).
				Render(fmt.Sprintf("%s %s", t.key, t.name))
		}
		renderedTabs = append(renderedTabs, s)
	}

	tabButtons := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	divider := gradientLine(width)
	return tabButtons + "\n" + divider
}

// gradientLine renders a full-width horizontal rule using the theme's
// spectrum gradient.
func gradientLine(width int) string {
	var b strings.Builder
	for i := 0; i < width; i++ {
		c := GetGradientColor(float64(i) / float64(max(1, width-1)))
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("─"))
	}
	return b.String()
}

// gradientText renders each rune of s interpolated across the theme's
// spectrum gradient.
func gradientText(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if r == '\n' {
			b.WriteByte('\n')
			continue
		}
		t := float64(i) / float64(max(1, len(runes)-1))
		c := GetGradientColor(t)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(r)))
	}
	return b.String()
}

func (m Model) renderContentBody(height, width int) string {
	switch m.activeTab {
	case TabResults:
		return m.results.View(height, width, m.engine.IsLiked)
	case TabLyrics:
		return m.lyrics.View()
	case TabPlaylists, TabLiked, TabHistory, TabDownloads:
		return m.renderLibrary(height, width)
	case TabSettings:
		return m.settings.View(width, height, m.engine.GetConfig().MaxCacheSizeGB, m.engine.GetCacheSize())
	}
	return ""
}

func (m Model) renderVinyl(innerHeight, innerWidth int) string {
	isPlaying := m.engine.GetState() == player.StatePlaying
	frame := m.vinylFrame % 4

	var lines []string
	switch frame {
	case 0:
		lines = []string{
			"       .--- - - ---.       ",
			"     .-'   .---.   '-.     ┬─┐",
			"   .-'   .-' | '-.   '-.   │ │",
			"  /     /  ( @ )  \\     \\  │ │",
			" |     |    / \\    |     | █ │",
			"  \\     \\         /     /    ▼",
			"   '-.   '-. _ .-'   '-.   ",
			"     '-.   '---'   .-'     ",
			"       '--- - - ---'       ",
		}
	case 1:
		lines = []string{
			"       .--- - - ---.       ",
			"     .-'   .---.   '-.     ┬─┐",
			"   .-'   .-' / '-.   '-.   │ │",
			"  /     /  ( @ )  \\     \\  │ │",
			" |     |    ─ ─    |     | █ │",
			"  \\     \\         /     /    ▼",
			"   '-.   '-. _ .-'   '-.   ",
			"     '-.   '---'   .-'     ",
			"       '--- - - ---'       ",
		}
	case 2:
		lines = []string{
			"       .--- - - ---.       ",
			"     .-'   .---.   '-.     ┬─┐",
			"   .-'   .-' \\ '-.   '-.   │ │",
			"  /     /  ( @ )  \\     \\  │ │",
			" |     |    \\ /    |     | █ │",
			"  \\     \\         /     /    ▼",
			"   '-.   '-. _ .-'   '-.   ",
			"     '-.   '---'   .-'     ",
			"       '--- - - ---'       ",
		}
	default:
		lines = []string{
			"       .--- - - ---.       ",
			"     .-'   .---.   '-.     ┬─┐",
			"   .-'   .-' | '-.   '-.   │ │",
			"  /     /  ( @ )  \\     \\  │ │",
			" |     |    │ │    |     | █ │",
			"  \\     \\         /     /    ▼",
			"   '-.   '-. _ .-'   '-.   ",
			"     '-.   '---'   .-'     ",
			"       '--- - - ---'       ",
		}
	}

	if !isPlaying {
		lines = []string{
			"       .--- - - ---.       ",
			"     .-'   .---.   '-.       ┬─┐",
			"   .-'   .-' | '-.   '-.     │ │",
			"  /     /  ( @ )  \\     \\    │ │",
			" |     |    / \\    |     |   █ │",
			"  \\     \\         /     /    │",
			"   '-.   '-. _ .-'   '-.     ▼",
			"     '-.   '---'   .-'     ",
			"       '--- - - ---'       ",
		}
	}

	vinylStyle := lipgloss.NewStyle().Foreground(FgSub)
	stylusStyle := lipgloss.NewStyle().Foreground(AccentBright)
	centerStyle := lipgloss.NewStyle().Foreground(Accent).Bold(true)

	var coloredLines []string
	for _, l := range lines {
		if len(l) > 27 {
			leftPart := l[:27]
			rightPart := l[27:]
			leftPart = strings.Replace(leftPart, "( @ )", centerStyle.Render("(@)"), 1)
			coloredLines = append(coloredLines, vinylStyle.Render(leftPart)+stylusStyle.Render(rightPart))
		} else {
			coloredLine := strings.Replace(l, "( @ )", centerStyle.Render("(@)"), 1)
			coloredLines = append(coloredLines, vinylStyle.Render(coloredLine))
		}
	}

	return strings.Join(coloredLines, "\n")
}

// renderVolumeBar was removed

// thumbTargetCols computes the artwork width (in terminal cells) that
// the now-playing pane will use at the current terminal size. It mirrors
// the wide-layout math in View() and must stay in sync with it.
func (m Model) thumbTargetCols() int {
	headerHeight := 4 + HeaderStyle.GetVerticalFrameSize()
	helpHeight := 1
	if m.help.ShowAll && m.terminalHeight >= 20 {
		helpHeight = 7
	}
	visibleHeight := m.terminalHeight - headerHeight - helpHeight
	if visibleHeight < 2 {
		visibleHeight = 2
	}
	paneFrameV := PaneStyle.GetVerticalFrameSize()
	vizOuter := 4 + paneFrameV
	bodyOverhead := vizOuter + 3 // compact now-playing bar
	topOuter := visibleHeight - bodyOverhead
	if topOuter < 1 || visibleHeight-bodyOverhead < 4 {
		topOuter = visibleHeight - 3
	}
	if topOuter < 1 {
		topOuter = 1
	}
	topInner := topOuter - paneFrameV
	if topInner < 1 {
		topInner = 1
	}

	contentWidth := m.terminalWidth - 4
	queueWidth := int(float64(contentWidth) * 0.20)
	mainWidth := int(float64(contentWidth) * 0.55)
	minQueueWidth := PaneStyle.GetHorizontalFrameSize() + 6
	minNowPlayingWidth := PaneStyle.GetHorizontalFrameSize() + 12
	minMainWidth := PaneStyle.GetHorizontalFrameSize() + 12
	if queueWidth < minQueueWidth {
		queueWidth = minQueueWidth
	}
	if mainWidth < minMainWidth {
		mainWidth = minMainWidth
	}
	nowPlayingWidth := contentWidth - queueWidth - mainWidth
	if nowPlayingWidth < minNowPlayingWidth {
		nowPlayingWidth = minNowPlayingWidth
	}
	innerWidth := nowPlayingWidth - PaneStyle.GetHorizontalFrameSize()
	if innerWidth < 1 {
		innerWidth = 1
	}

	_, artMaxCols := artBoxSize(topInner, innerWidth)
	return artMaxCols
}

// artBoxSize returns the framed-art box height (including frame) and the
// maximum art column count for a given inner pane size.
func artBoxSize(innerHeight, innerWidth int) (boxHeight, artMaxCols int) {
	const metaHeight = 6 // blank + title + artist line + blank + playbar
	boxHeight = innerHeight - metaHeight
	if boxHeight < 7 {
		boxHeight = 7
	}
	artMaxCols = innerWidth - 2 // rounded frame border
	artMaxRows := boxHeight - 2 // rounded frame border
	if artMaxRows < 2 {
		artMaxRows = 2
	}
	if artMaxCols > artMaxRows*2 {
		artMaxCols = artMaxRows * 2
	}
	if artMaxCols < 4 {
		artMaxCols = 4
	}
	return boxHeight, artMaxCols
}

// marqueeText scrolls s horizontally within avail cells, wrapping around.
// If it fits, s is returned unchanged.
func marqueeText(s string, avail int, offset int) string {
	if lipgloss.Width(s) <= avail {
		return s
	}
	runes := []rune(s + "   •   ")
	n := len(runes)
	off := offset % n
	shifted := make([]rune, 0, n)
	shifted = append(shifted, runes[off:]...)
	shifted = append(shifted, runes[:off]...)

	var out []rune
	w := 0
	for _, r := range shifted {
		rw := lipgloss.Width(string(r))
		if w+rw > avail {
			break
		}
		out = append(out, r)
		w += rw
	}
	return string(out)
}

func (m Model) renderNowPlaying(innerHeight, innerWidth int) string {
	nowPlayingContent := ""
	track := m.engine.GetCurrentTrack()
	if track != nil {
		title := StyleTitle.Render(marqueeText(track.Title, innerWidth, int(m.statusBar.elapsed.Seconds()*3)))

		artistLine := StyleMeta.Render(truncateText(track.Artist, innerWidth))
		if m.engine.IsLiked(track.VideoID) {
			artistLine += " " + StyleBadgeLiked.Render("♥")
		}
		if track.LocalPath != "" {
			artistLine += " " + StyleBadgeCached.Render("✓ cached")
		}

		playbar := m.statusBar.PlaybarView(track, m.engine.GetState(), innerWidth)

		noticeLine := ""
		if m.notice != "" {
			style := lipgloss.NewStyle().Foreground(Ok)
			if m.noticeErr {
				style = lipgloss.NewStyle().Foreground(Danger)
			}
			noticeLine = style.Render(truncateText(m.notice, innerWidth))
		}

		boxHeight, _ := artBoxSize(innerHeight, innerWidth)

		var artBlock string
		switch {
		case m.thumbnail != "":
			framed := ArtFrameStyle.Render(m.thumbnail)
			artBlock = lipgloss.Place(innerWidth, boxHeight, lipgloss.Center, lipgloss.Top, framed)
		case m.thumbError != "":
			errView := lipgloss.NewStyle().
				Foreground(FgSub).
				Render(truncateText("⚠ "+m.thumbError, innerWidth))
			vinyl := m.renderVinyl(boxHeight-2, innerWidth)
			artBlock = lipgloss.Place(
				innerWidth, boxHeight, lipgloss.Center, lipgloss.Center,
				lipgloss.JoinVertical(lipgloss.Center, vinyl, "", errView),
			)
		default:
			// Art still loading: spin the vinyl
			vinyl := m.renderVinyl(boxHeight, innerWidth)
			artBlock = lipgloss.Place(innerWidth, boxHeight, lipgloss.Center, lipgloss.Center, vinyl)
		}

		nowPlayingContent = lipgloss.JoinVertical(lipgloss.Center,
			artBlock,
			"",
			title,
			artistLine,
			"",
			playbar,
			noticeLine,
		)
	} else {
		nowPlayingContent = "\n\n  " + StyleMeta.Render("Nothing playing")
	}

	nowPlayingContent = lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Top, nowPlayingContent)
	return nowPlayingContent
}

func (m Model) renderCompactNowPlaying(width int) string {
	track := m.engine.GetCurrentTrack()
	if track == nil {
		return ""
	}
	state := m.engine.GetState()

	titleText := truncateText(track.Title, 28)
	if m.engine.IsLiked(track.VideoID) {
		titleText += " " + StyleBadgeLiked.Render("♥")
	}
	title := StyleTitle.Render(titleText)
	artist := StyleMeta.Render(truncateText(track.Artist, 28))

	prefix := lipgloss.JoinHorizontal(lipgloss.Center,
		StyleMeta.Render("♪ "),
		title,
		StyleMeta.Render(" • "),
		artist,
		StyleMeta.Render("  "),
	)

	prefixWidth := lipgloss.Width(prefix)
	playbarWidth := width - prefixWidth
	if playbarWidth < 10 {
		playbarWidth = 10
	}

	playbar := m.statusBar.PlaybarView(track, state, playbarWidth)

	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, playbar)
}

func truncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

func (m Model) renderOverlays(fullView string) string {
	if m.showPlaylistPrompt {
		promptView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			Padding(1, 2).
			Render("Create New Playlist\n\n" + m.playlistPrompt.View() + "\n\n(Enter to create, Esc to cancel)")

		fullView = m.placeOverlay(fullView, promptView)
	}

	if m.showPlaylistSelector {
		var lines []string
		lines = append(lines, StyleTitle.Render("Add to Playlist"))
		lines = append(lines, "")
		for i, p := range m.allPlaylists {
			if i == m.playlistSelectorIndex {
				lines = append(lines, StyleSelected.Render("> "+p.Name))
			} else {
				lines = append(lines, StyleNormal.Render("  "+p.Name))
			}
		}
		lines = append(lines, "")
		lines = append(lines, StyleMeta.Render("(Enter to select, Esc to cancel)"))

		selectorView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			Padding(1, 2).
			Render(strings.Join(lines, "\n"))

		fullView = m.placeOverlay(fullView, selectorView)
	}

	return fullView
}
