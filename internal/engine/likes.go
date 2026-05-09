package engine

import "github.com/Kush-Singh-26/goktave/internal/provider"

func (e *DefaultEngine) ToggleLike(videoID string) (bool, error) {
	return e.db.ToggleLike(videoID)
}

func (e *DefaultEngine) IsLiked(videoID string) bool {
	return e.db.IsLiked(videoID)
}

func (e *DefaultEngine) GetLikedTracks() ([]provider.Track, error) {
	return e.db.GetLikedTracks()
}
