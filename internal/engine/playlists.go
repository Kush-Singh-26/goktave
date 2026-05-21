package engine

import (
	"github.com/Kush-Singh-26/goktave/internal/db"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

func (e *DefaultEngine) CreatePlaylist(name string) error {
	return e.db.CreatePlaylist(name)
}

func (e *DefaultEngine) DeletePlaylist(name string) error {
	return e.db.DeletePlaylist(name)
}

func (e *DefaultEngine) GetPlaylists() ([]db.Playlist, error) {
	return e.db.GetPlaylists()
}

func (e *DefaultEngine) AddTrackToPlaylist(playlistName string, videoID string) error {
	return e.db.AddTrackToPlaylist(playlistName, videoID)
}

func (e *DefaultEngine) RemoveTrackFromPlaylist(playlistName string, videoID string) error {
	return e.db.RemoveTrackFromPlaylist(playlistName, videoID)
}

func (e *DefaultEngine) GetPlaylistTracks(name string) ([]provider.Track, error) {
	return e.db.GetPlaylistTracks(name)
}

func (e *DefaultEngine) PlayPlaylist(name string) error {
	tracks, err := e.db.GetPlaylistTracks(name)
	if err != nil {
		return err
	}
	return e.PlayTracks(tracks)
}

func (e *DefaultEngine) PlayTracks(tracks []provider.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.queue = make([]provider.Track, len(tracks)-1)
	copy(e.queue, tracks[1:])
	e.saveQueue()
	return e.playLockedWithOffset(tracks[0], true, 0)
}
