package engine

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"github.com/Kush-Singh-26/goktave/internal/logger"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

func (e *DefaultEngine) saveQueue() {
	// Assumes lock is held or not needed
	data, err := json.Marshal(e.queue)
	if err != nil {
		logger.L.Error("failed to marshal queue", "err", err)
		return
	}
	// Atomic write: a crash mid-write must not corrupt the persisted queue.
	tmp := e.cfg.QueuePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		logger.L.Error("failed to save queue", "err", err)
		return
	}
	if err := os.Rename(tmp, e.cfg.QueuePath); err != nil {
		logger.L.Error("failed to finalize queue file", "err", err)
	}
}

func (e *DefaultEngine) loadQueue() {
	data, err := os.ReadFile(e.cfg.QueuePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.L.Error("failed to read queue file", "err", err)
		}
		return
	}
	var q []provider.Track
	if err := json.Unmarshal(data, &q); err != nil {
		logger.L.Error("failed to unmarshal queue", "err", err)
		return
	}
	e.queue = q
}

func (e *DefaultEngine) Queue(track provider.Track) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.queueLocked(track)
}

func (e *DefaultEngine) queueLocked(track provider.Track) {
	logger.L.Info("Adding to queue", "title", track.Title)
	e.queue = append(e.queue, track)
	e.saveQueue()
}

func (e *DefaultEngine) Next() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.nextLocked()
}

func (e *DefaultEngine) nextLocked() error {
	if len(e.queue) == 0 {
		track := e.currentTrack
		if track != nil {
			logger.L.Info("Queue empty, fetching radio for", "title", track.Title)
			// Unlock briefly for network call if we want, but provider calls are usually fine
			// Actually, GetUpNext is a network call, we should NOT hold the lock.
			e.mu.Unlock()
			results, _, err := e.provider.GetUpNext(track.VideoID)
			e.mu.Lock()

			if err == nil && len(results) > 1 {
				for i := 1; i < 21 && i < len(results); i++ {
					e.queue = append(e.queue, results[i])
				}
				e.saveQueue()
				return e.nextLocked()
			}
		}
		return nil
	}

	next := e.queue[0]
	e.queue = e.queue[1:]
	e.saveQueue()

	return e.playLocked(next, true)
}

func (e *DefaultEngine) GetQueue() []provider.Track {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Return a copy: the UI iterates this lock-free while engine
	// goroutines may append/re-slice the underlying slice.
	q := make([]provider.Track, len(e.queue))
	copy(q, e.queue)
	return q
}

func (e *DefaultEngine) RemoveFromQueue(index int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if index < 0 || index >= len(e.queue) {
		return
	}
	e.queue = append(e.queue[:index], e.queue[index+1:]...)
	e.saveQueue()
}

func (e *DefaultEngine) ClearQueue() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.queue = nil
	e.saveQueue()
}

func (e *DefaultEngine) PlayFromQueue(index int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if index < 0 || index >= len(e.queue) {
		return nil
	}
	track := e.queue[index]
	e.queue = append(e.queue[:index], e.queue[index+1:]...)
	e.saveQueue()

	return e.playLocked(track, true)
}

func (e *DefaultEngine) MoveInQueue(from, to int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if from < 0 || from >= len(e.queue) || to < 0 || to >= len(e.queue) {
		return
	}
	track := e.queue[from]
	e.queue = append(e.queue[:from], e.queue[from+1:]...)

	newQueue := make([]provider.Track, 0, len(e.queue)+1)
	newQueue = append(newQueue, e.queue[:to]...)
	newQueue = append(newQueue, track)
	newQueue = append(newQueue, e.queue[to:]...)
	e.queue = newQueue
	e.saveQueue()
}

func (e *DefaultEngine) Shuffle() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.queue) <= 1 {
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(e.queue), func(i, j int) {
		e.queue[i], e.queue[j] = e.queue[j], e.queue[i]
	})
	e.saveQueue()
}
