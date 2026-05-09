package db

import (
	"encoding/json"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/provider"
	bolt "go.etcd.io/bbolt"
)

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
