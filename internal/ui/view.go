package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	if m.terminalWidth < 20 || m.terminalHeight < 10 {
		return tea.View{Content: "Terminal too small"}
	}

	branding := "▅▆██▆▅\ngoktave"

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

		queueStyle := PaneStyle.Copy().Width(queueWidth).Height(topHeightOuter)
		if m.focusArea == AreaQueue {
			queueStyle = ActivePaneStyle.Copy().Width(queueWidth).Height(topHeightOuter)
		}
		queueInnerWidth := queueWidth - PaneStyle.GetHorizontalFrameSize()
		if queueInnerWidth < 1 {
			queueInnerWidth = 1
		}
		queueContent := m.queue.View(m.engine.GetQueue(), topInnerHeight, queueInnerWidth)
		queueContent = lipgloss.Place(queueInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, queueContent)
		queuePane := queueStyle.Render(queueContent)

		tabRow := m.buildTabRow()
		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		contentInnerHeight := topInnerHeight - 1
		if contentInnerHeight < 1 {
			contentInnerHeight = 1
		}
		contentBody := m.renderContentBody(contentInnerHeight, contentInnerWidth)
		contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
		if m.focusArea == AreaContent {
			contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
		}
		contentText := tabRow + "\n" + contentBody
		contentText = lipgloss.Place(contentInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, contentText)
		contentPane := contentStyle.Render(contentText)

		nowPlayingContent := m.renderNowPlaying(topInnerHeight, nowPlayingInnerWidth)
		nowPlayingPane := PaneStyle.Copy().
			Width(nowPlayingWidth).
			Height(topHeightOuter).
			Render(nowPlayingContent)

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
				Height(4 + PaneStyle.GetVerticalFrameSize()).
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

		queueStyle := PaneStyle.Copy().Width(queueWidth).Height(topHeightOuter)
		if m.focusArea == AreaQueue {
			queueStyle = ActivePaneStyle.Copy().Width(queueWidth).Height(topHeightOuter)
		}
		queueInnerWidth := queueWidth - PaneStyle.GetHorizontalFrameSize()
		if queueInnerWidth < 1 {
			queueInnerWidth = 1
		}
		queueContent := m.queue.View(m.engine.GetQueue(), topInnerHeight, queueInnerWidth)
		queueContent = lipgloss.Place(queueInnerWidth, topInnerHeight, lipgloss.Left, lipgloss.Top, queueContent)
		queuePane := queueStyle.Render(queueContent)

		tabRow := m.buildTabRow()
		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		contentInnerHeight := topInnerHeight - 1
		if contentInnerHeight < 1 {
			contentInnerHeight = 1
		}
		contentBody := m.renderContentBody(contentInnerHeight, contentInnerWidth)
		contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
		if m.focusArea == AreaContent {
			contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
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
				Height(4 + PaneStyle.GetVerticalFrameSize()).
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

		tabRow := m.buildTabRow()
		contentInnerWidth := mainWidth - PaneStyle.GetHorizontalFrameSize()
		if contentInnerWidth < 1 {
			contentInnerWidth = 1
		}
		contentInnerHeight := topInnerHeight - 1
		if contentInnerHeight < 1 {
			contentInnerHeight = 1
		}
		contentBody := m.renderContentBody(contentInnerHeight, contentInnerWidth)
		contentStyle := PaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
		if m.focusArea == AreaContent {
			contentStyle = ActivePaneStyle.Copy().Width(mainWidth).Height(topHeightOuter)
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
				Height(3).
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

func (m Model) buildTabRow() string {
	tabs := []string{
		"[1] Results",
		"[2] Lyrics",
		"[3] Playlists",
		"[4] Liked",
		"[5] History",
		"[6] Downloads",
		"[7] Settings",
	}
	row := ""
	for i, t := range tabs {
		style := TabStyle
		if int(m.activeTab) == i {
			style = ActiveTabStyle
		}
		row += style.Render(t)
	}
	return row
}

func (m Model) renderContentBody(height, width int) string {
	switch m.activeTab {
	case TabResults:
		return m.results.View(height, width)
	case TabLyrics:
		return m.lyrics.View()
	case TabPlaylists, TabLiked, TabHistory, TabDownloads:
		return m.renderLibrary(height, width)
	case TabSettings:
		return m.settings.View(width, height, m.engine.GetConfig().MaxCacheSizeGB, m.engine.GetCacheSize())
	}
	return ""
}

func (m Model) renderNowPlaying(innerHeight, innerWidth int) string {
	nowPlayingContent := ""
	track := m.engine.GetCurrentTrack()
	if track != nil {
		thumb := m.thumbnail
		if thumb == "" {
			thumb = "\n\n  No Thumbnail"
		}

		title := StyleTitle.Render(track.Title)
		artist := StyleMeta.Render(track.Artist)
		likeStatus := ""
		if m.engine.IsLiked(track.VideoID) {
			likeStatus = " ❤️"
		}
		offlineStatus := ""
		if track.LocalPath != "" {
			offlineStatus = " " + StyleMeta.Render("✔")
		}

		playbarWidth := innerWidth
		playbar := m.statusBar.PlaybarView(track, m.engine.GetState(), playbarWidth)

		// Ensure thumbnail doesn't overflow
		// Content area inside pane is inner height
		// title (1) + artist (1) + playbar (1) + 3 spacing (\n) = 6
		metadataHeight := 6
		innerNowPlayingHeight := innerHeight
		if innerNowPlayingHeight < 1 {
			innerNowPlayingHeight = 1
		}
		maxThumbHeight := innerNowPlayingHeight - metadataHeight
		if maxThumbHeight < 5 {
			maxThumbHeight = 5
		}

		// Calculate thumbnail width based on the pane width minus borders/padding
		thumbWidth := innerWidth
		thumb = clampLines(thumb, maxThumbHeight)
		thumb = lipgloss.Place(thumbWidth, maxThumbHeight, lipgloss.Center, lipgloss.Top, thumb)

		nowPlayingContent = lipgloss.JoinVertical(lipgloss.Center,
			ThumbnailStyle.Width(thumbWidth).MaxHeight(maxThumbHeight).Render(thumb),
			"\n",
			title+likeStatus+offlineStatus,
			artist,
			"\n",
			playbar,
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
	playbar := m.statusBar.PlaybarView(track, state, width)

	title := StyleTitle.Render(truncateText(track.Title, 28))
	artist := StyleMeta.Render(truncateText(track.Artist, 28))

	return lipgloss.JoinHorizontal(lipgloss.Center,
		StyleMeta.Render("♪ "),
		title,
		StyleMeta.Render(" • "),
		artist,
		StyleMeta.Render("  "),
		playbar,
	)
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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
