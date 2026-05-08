package extractor

import (
	"context"
	"sync"
	"time"
)

type cacheEntry struct {
	info      *StreamInfo
	expiresAt time.Time
}

type CachedExtractor struct {
	underlying Extractor
	cache      map[string]cacheEntry
	mu         sync.RWMutex
	ttl        time.Duration
}

func NewCachedExtractor(underlying Extractor, ttl time.Duration) *CachedExtractor {
	return &CachedExtractor{
		underlying: underlying,
		cache:      make(map[string]cacheEntry),
		ttl:        ttl,
	}
}

func (e *CachedExtractor) Extract(ctx context.Context, videoID string) (*StreamInfo, error) {
	e.mu.RLock()
	entry, ok := e.cache[videoID]
	e.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.info, nil
	}

	info, err := e.underlying.Extract(ctx, videoID)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[videoID] = cacheEntry{
		info:      info,
		expiresAt: time.Now().Add(e.ttl),
	}
	e.mu.Unlock()

	return info, nil
}
