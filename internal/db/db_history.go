package db

import (
	"encoding/json"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	bolt "go.etcd.io/bbolt"
)

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
