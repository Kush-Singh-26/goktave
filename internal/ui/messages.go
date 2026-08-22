package ui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Kush-Singh-26/goktave/internal/provider"
)

type SearchResultsMsg struct {
	Results []provider.Track
	Err     error
}

type SuggestionsMsg struct {
	Suggestions []string
	Err         error
}

type SearchHistoryMsg struct {
	History []string
}

type ThumbnailMsg struct {
	VideoID string
	Art     string
	Cols    int
	Err     error
}

// SeekFlushMsg fires after seek nudging goes quiet, applying the accumulated
// offset in a single engine.Seek instead of one per keypress.
type SeekFlushMsg struct {
	Deadline time.Time
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func SetTitleCmd(title string) tea.Cmd {
	return func() tea.Msg {
		fmt.Printf("\033]2;%s\007", title)
		return nil
	}
}
