package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	bolt "go.etcd.io/bbolt"
)

func (db *DB) CreatePlaylist(name string) error {
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketPlaylists)
		p := Playlist{
			ID:        name,
			Name:      name,
			TrackIDs:  []string{},
			CreatedAt: time.Now().Unix(),
		}
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		return b.Put([]byte(name), data)
	})
}

func (db *DB) DeletePlaylist(name string) error {
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketPlaylists)
		return b.Delete([]byte(name))
	})
}

func (db *DB) GetPlaylists() ([]Playlist, error) {
	var playlists []Playlist
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketPlaylists)
		return b.ForEach(func(k, v []byte) error {
			var p Playlist
			if err := json.Unmarshal(v, &p); err == nil {
				playlists = append(playlists, p)
			}
			return nil
		})
	})
	return playlists, err
}

func (db *DB) AddTrackToPlaylist(playlistName string, videoID string) error {
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketPlaylists)
		v := b.Get([]byte(playlistName))
		if v == nil {
			return fmt.Errorf("playlist not found")
		}
		var p Playlist
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}
		// Check if already exists
		for _, id := range p.TrackIDs {
			if id == videoID {
				return nil
			}
		}
		p.TrackIDs = append(p.TrackIDs, videoID)
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		return b.Put([]byte(playlistName), data)
	})
}

func (db *DB) RemoveTrackFromPlaylist(playlistName string, videoID string) error {
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketPlaylists)
		v := b.Get([]byte(playlistName))
		if v == nil {
			return fmt.Errorf("playlist not found")
		}
		var p Playlist
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}
		newIDs := []string{}
		for _, id := range p.TrackIDs {
			if id != videoID {
				newIDs = append(newIDs, id)
			}
		}
		p.TrackIDs = newIDs
		data, err := json.Marshal(p)
		if err != nil {
			return err
		}
		return b.Put([]byte(playlistName), data)
	})
}

func (db *DB) GetPlaylistTracks(name string) ([]provider.Track, error) {
	var tracks []provider.Track
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketPlaylists)
		v := b.Get([]byte(name))
		if v == nil {
			return fmt.Errorf("playlist not found")
		}
		var p Playlist
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}

		tracksBucket := tx.Bucket(BucketTracks)
		for _, id := range p.TrackIDs {
			trackData := tracksBucket.Get([]byte(id))
			if trackData != nil {
				var t provider.Track
				if err := json.Unmarshal(trackData, &t); err == nil {
					tracks = append(tracks, t)
				}
			}
		}
		return nil
	})
	return tracks, err
}
