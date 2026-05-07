package ui

import "fmt"

func formatDuration(totalSeconds int) string {
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
