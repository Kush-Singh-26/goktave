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

	branding := "█▀▀▀█  █▀▀▀█  █  ▄▀  ▀▀█▀▀  █▀▀▀█  █   █  █▀▀▀█\n" +
		"█ ▀▀█  █   █  █▀▀▄     █    █▀▀▀█  █   █  █▀▀▀ \n" +
		"▀▀▀▀▀  ▀▀▀▀▀  ▀  ▀     ▀    ▀   ▀   ▀▀▀   ▀▀▀▀▀"

	brandingWidth := 51
	searchWidth := 40
	suggestWidth := 50
	compact := m.terminalHeight <= 33
	headerBoxHeight := 5
	if compact {
		headerBoxHeight = 3
	}

	// Adaptive scaling for smaller terminals
	if m.terminalWidth < brandingWidth+searchWidth+suggestWidth+4 {
		brandingWidth = int(float64(m.terminalWidth-2) * 0.30)
		remaining := (m.terminalWidth - 2) - brandingWidth
		searchWidth = int(float64(remaining) * 0.45)
		suggestWidth = int(float64(remaining) * 0.40)
	}
	if brandingWidth < 10 {
		brandingWidth = 10
	}
	if searchWidth < 10 {
		searchWidth = 10
	}
	if suggestWidth < 10 {
		suggestWidth = 10
	}

	brandingBox := HeaderStyle.Copy().
		Width(brandingWidth).
		Height(headerBoxHeight).
		Foreground(Terracotta).
		Padding(0, 1).
		Render(branding)

	searchBoxView := StyleMeta.Render("Search:") + "\n" + m.search.View()
	searchBoxStyle := HeaderStyle.Copy().
		Width(searchWidth).
		Height(headerBoxHeight).
		Padding(0, 1).
		Align(lipgloss.Left, lipgloss.Center)

	if m.focusArea == AreaSearch {
		searchBoxStyle = searchBoxStyle.BorderForeground(Terracotta)
	}
	searchBox := searchBoxStyle.Render(searchBoxView)

	var header string
	if compact {
		gap := "  "
		header = lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, gap, searchBox)
	} else {
		suggestLines := m.headerSuggestLines(suggestWidth)
		suggestBoxView := strings.Join(suggestLines, "\n")
		suggestBox := HeaderStyle.Copy().
			Width(suggestWidth).
			Height(headerBoxHeight).
			Padding(0, 1).
			Align(lipgloss.Left, lipgloss.Center).
			Render(suggestBoxView)

		// Add a gap between branding and search box to "shift" it right
		gap := "  "
		header = lipgloss.JoinHorizontal(lipgloss.Top, brandingBox, gap, searchBox, suggestBox)
	}

	// Help line
	helpHeight := 1
	if !compact && m.help.ShowAll {
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
	vizOuterHeight := 4 + paneFrameV
	topHeightOuter := visibleHeight - vizOuterHeight
	if topHeightOuter < 1 {
		topHeightOuter = visibleHeight
		vizOuterHeight = 0
	}
	topInnerHeight := topHeightOuter - paneFrameV
	if topInnerHeight < 1 {
		topInnerHeight = 1
	}
	if compact {
		vizOuterHeight = 0
		topHeightOuter = visibleHeight
		topInnerHeight = visibleHeight - paneFrameV
		if topInnerHeight < 1 {
			topInnerHeight = 1
		}
	}

	contentWidth := m.terminalWidth - 4
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

	// Queue Pane
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

	// Main Content Pane
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

	// Now Playing Pane
	nowPlayingContent := m.renderNowPlaying(topInnerHeight, nowPlayingInnerWidth)

	nowPlayingPane := PaneStyle.Copy().
		Width(nowPlayingWidth).
		Height(topHeightOuter).
		Render(nowPlayingContent)

	vizHeight := 4
	visualizerWidth := queueWidth + mainWidth
	visualizerInnerWidth := visualizerWidth - PaneStyle.GetHorizontalFrameSize()
	if visualizerInnerWidth < 1 {
		visualizerInnerWidth = 1
	}
	visualizerView := m.statusBar.VisualizerView(m.engine.GetVisualizerBars(visualizerInnerWidth), visualizerInnerWidth, vizHeight)
	leftTopRow := lipgloss.JoinHorizontal(lipgloss.Top, queuePane, contentPane)
	leftColumn := leftTopRow
	if vizOuterHeight > 0 {
		visualizerPane := PaneStyle.Copy().
			Width(visualizerWidth).
			Height(vizHeight + PaneStyle.GetVerticalFrameSize()).
			Render(visualizerView)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftTopRow, visualizerPane)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, nowPlayingPane)

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

			bar := lipgloss.NewStyle().Foreground(Terracotta).Render(strings.Repeat("█", filled)) +
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
		style := lipgloss.NewStyle().Padding(0, 1)
		if int(m.activeTab) == i {
			style = style.Foreground(Terracotta).Bold(true).Underline(true)
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

func (m Model) renderOverlays(fullView string) string {
	if m.showPlaylistPrompt {
		promptView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Terracotta).
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
			BorderForeground(Terracotta).
			Padding(1, 2).
			Render(strings.Join(lines, "\n"))

		fullView = m.placeOverlay(fullView, selectorView)
	}

	return fullView
}
