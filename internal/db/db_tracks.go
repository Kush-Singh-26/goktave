package db

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	bolt "go.etcd.io/bbolt"
)

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
