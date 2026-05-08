package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	bolt "go.etcd.io/bbolt"
)

var (
	BucketTracks    = []byte("Tracks")
	BucketLikes     = []byte("Likes")
	BucketHistory   = []byte("History")
	BucketSearch    = []byte("SearchHistory")
	BucketPlaylists = []byte("Playlists")
	BucketState     = []byte("State")
)

type Playlist struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	TrackIDs  []string `json:"track_ids"`
	CreatedAt int64    `json:"created_at"`
}

type DB struct {
	conn *bolt.DB
}

func New(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	conn, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	err = conn.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{BucketTracks, BucketLikes, BucketHistory, BucketSearch, BucketPlaylists, BucketState}
		for _, b := range buckets {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Track operations
func (db *DB) SaveTrack(track provider.Track) error {
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketTracks)
		data, err := json.Marshal(track)
		if err != nil {
			return err
		}
		return b.Put([]byte(track.VideoID), data)
	})
}

func (db *DB) GetTrack(videoID string) (*provider.Track, error) {
	var track provider.Track
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketTracks)
		v := b.Get([]byte(videoID))
		if v == nil {
			return fmt.Errorf("track not found")
		}
		return json.Unmarshal(v, &track)
	})
	if err != nil {
		return nil, err
	}
	return &track, nil
}

// Like operations
func (db *DB) ToggleLike(videoID string) (bool, error) {
	var liked bool
	err := db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketLikes)
		if b.Get([]byte(videoID)) != nil {
			liked = false
			return b.Delete([]byte(videoID))
		}
		liked = true
		return b.Put([]byte(videoID), []byte(time.Now().Format(time.RFC3339)))
	})
	return liked, err
}

func (db *DB) IsLiked(videoID string) bool {
	var liked bool
	_ = db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketLikes)
		liked = b.Get([]byte(videoID)) != nil
		return nil
	})
	return liked
}

func (db *DB) GetLikedTracks() ([]provider.Track, error) {
	var tracks []provider.Track
	err := db.conn.View(func(tx *bolt.Tx) error {
		likesBucket := tx.Bucket(BucketLikes)
		tracksBucket := tx.Bucket(BucketTracks)
		
		return likesBucket.ForEach(func(k, v []byte) error {
			trackData := tracksBucket.Get(k)
			if trackData != nil {
				var t provider.Track
				if err := json.Unmarshal(trackData, &t); err == nil {
					tracks = append(tracks, t)
				}
			}
			return nil
		})
	})
	return tracks, err
}

// History operations
func (db *DB) AddToHistory(track provider.Track) error {
	_ = db.SaveTrack(track)
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketHistory)
		key := []byte(time.Now().Format(time.RFC3339Nano))
		return b.Put(key, []byte(track.VideoID))
	})
}

func (db *DB) AddSearch(query string) error {
	if query == "" {
		return nil
	}
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketSearch)
		// Use timestamp as key for ordering, query as value
		key := []byte(time.Now().Format(time.RFC3339Nano))
		return b.Put(key, []byte(query))
	})
}

func (db *DB) GetSearchHistory(limit int) ([]string, error) {
	var history []string
	seen := make(map[string]bool)
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketSearch)
		c := b.Cursor()
		for k, v := c.Last(); k != nil && len(history) < limit; k, v = c.Prev() {
			query := string(v)
			if !seen[query] {
				history = append(history, query)
				seen[query] = true
			}
		}
		return nil
	})
	return history, err
}

func (db *DB) GetHistory(limit int) ([]provider.Track, error) {
	var tracks []provider.Track
	err := db.conn.View(func(tx *bolt.Tx) error {
		historyBucket := tx.Bucket(BucketHistory)
		tracksBucket := tx.Bucket(BucketTracks)
		
		c := historyBucket.Cursor()
		count := 0
		for k, v := c.Last(); k != nil && count < limit; k, v = c.Prev() {
			trackData := tracksBucket.Get(v)
			if trackData != nil {
				var t provider.Track
				if err := json.Unmarshal(trackData, &t); err == nil {
					tracks = append(tracks, t)
					count++
				}
			}
		}
		return nil
	})
	return tracks, err
}

func (db *DB) GetDownloadedTracks() ([]provider.Track, error) {
	var tracks []provider.Track
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketTracks)
		return b.ForEach(func(k, v []byte) error {
			var t provider.Track
			if err := json.Unmarshal(v, &t); err == nil {
				if t.LocalPath != "" {
					// Verify file still exists
					if _, err := os.Stat(t.LocalPath); err == nil {
						tracks = append(tracks, t)
					}
				}
			}
			return nil
		})
	})
	return tracks, err
}

func (db *DB) GetTracksByLastPlayed() ([]provider.Track, error) {
	var tracks []provider.Track
	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketTracks)
		return b.ForEach(func(k, v []byte) error {
			var t provider.Track
			if err := json.Unmarshal(v, &t); err == nil {
				if t.LocalPath != "" {
					tracks = append(tracks, t)
				}
			}
			return nil
		})
	})
	return tracks, err
}

// Playlist operations
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

func (db *DB) SaveConfig(cfg interface{}) error {
	return db.SaveState("config", cfg)
}

// State operations
func (db *DB) SaveState(key string, value interface{}) error {
	return db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketState)
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), data)
	})
}

func (db *DB) GetState(key string, v interface{}) error {
	return db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketState)
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("state not found")
		}
		return json.Unmarshal(data, v)
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
