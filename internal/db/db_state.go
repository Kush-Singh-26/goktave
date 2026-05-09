package db

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

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
