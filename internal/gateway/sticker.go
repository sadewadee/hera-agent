package gateway

import (
	"sync"
	"time"
)

// stickerEntry holds a cached sticker with metadata.
type stickerEntry struct {
	Data      []byte
	CachedAt  time.Time
	Platform  string
	StickerID string
}

// StickerCache provides an in-memory cache for sticker data fetched from
// messaging platforms (Telegram, Discord, etc.).
type StickerCache struct {
	mu      sync.RWMutex
	entries map[string]*stickerEntry // key: "platform:stickerID"
	maxSize int                      // max entries before eviction
	ttl     time.Duration            // time-to-live per entry
}

// NewStickerCache creates a new sticker cache with the given capacity and TTL.
func NewStickerCache(maxSize int, ttl time.Duration) *StickerCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &StickerCache{
		entries: make(map[string]*stickerEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// stickerKey builds a cache key from platform and sticker ID.
func stickerKey(platform, stickerID string) string {
	return platform + ":" + stickerID
}

// Get retrieves sticker data from the cache. Returns the data and true if
// found and not expired, or nil and false otherwise.
func (sc *StickerCache) Get(platform, stickerID string) ([]byte, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	key := stickerKey(platform, stickerID)
	entry, ok := sc.entries[key]
	if !ok {
		return nil, false
	}

	// Check TTL.
	if time.Since(entry.CachedAt) > sc.ttl {
		return nil, false
	}

	return entry.Data, true
}

// Put stores sticker data in the cache. Evicts the oldest entry if the cache
// is full.
func (sc *StickerCache) Put(platform, stickerID string, data []byte) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	key := stickerKey(platform, stickerID)

	// Evict expired entries first.
	sc.evictExpiredLocked()

	// If still at capacity, evict the oldest entry.
	if len(sc.entries) >= sc.maxSize {
		sc.evictOldestLocked()
	}

	sc.entries[key] = &stickerEntry{
		Data:      data,
		CachedAt:  time.Now(),
		Platform:  platform,
		StickerID: stickerID,
	}
}

// Size returns the number of entries in the cache.
func (sc *StickerCache) Size() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.entries)
}

// Clear removes all entries from the cache.
func (sc *StickerCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries = make(map[string]*stickerEntry)
}

// evictExpiredLocked removes entries that have exceeded their TTL.
// Caller must hold the write lock.
func (sc *StickerCache) evictExpiredLocked() {
	now := time.Now()
	for key, entry := range sc.entries {
		if now.Sub(entry.CachedAt) > sc.ttl {
			delete(sc.entries, key)
		}
	}
}

// evictOldestLocked removes the oldest entry. Caller must hold the write lock.
func (sc *StickerCache) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range sc.entries {
		if oldestKey == "" || entry.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CachedAt
		}
	}

	if oldestKey != "" {
		delete(sc.entries, oldestKey)
	}
}
