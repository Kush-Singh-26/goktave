package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
