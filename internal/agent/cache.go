package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// CacheEntry holds a cached response with metadata.
type CacheEntry struct {
	Response  string    `json:"response"`
	CreatedAt time.Time `json:"created_at"`
	HitCount  int       `json:"hit_count"`
}

// ResponseCache provides a simple in-memory LRU cache for repeated queries.
type ResponseCache struct {
	mu      sync.Mutex
	entries map[string]*CacheEntry
	order   []string // oldest first for eviction
	maxSize int
	ttl     time.Duration
}

// NewResponseCache creates a response cache with the given capacity and TTL.
func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	if maxSize <= 0 {
		maxSize = 100
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &ResponseCache{
		entries: make(map[string]*CacheEntry),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a cached response for the given query. Returns empty string and
// false if not found or expired.
func (rc *ResponseCache) Get(query string) (string, bool) {
	key := rc.hash(query)

	rc.mu.Lock()
	defer rc.mu.Unlock()

	entry, ok := rc.entries[key]
	if !ok {
		return "", false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > rc.ttl {
		delete(rc.entries, key)
		rc.removeFromOrder(key)
		return "", false
	}

	entry.HitCount++
	return entry.Response, true
}

// Set stores a response in the cache, evicting the oldest entry if full.
func (rc *ResponseCache) Set(query, response string) {
	key := rc.hash(query)

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Update existing entry
	if _, exists := rc.entries[key]; exists {
		rc.entries[key] = &CacheEntry{
			Response:  response,
			CreatedAt: time.Now(),
		}
		return
	}

	// Evict if at capacity
	if len(rc.entries) >= rc.maxSize {
		rc.evictOldest()
	}

	rc.entries[key] = &CacheEntry{
		Response:  response,
		CreatedAt: time.Now(),
	}
	rc.order = append(rc.order, key)
}

// Invalidate removes a specific entry from the cache.
func (rc *ResponseCache) Invalidate(query string) {
	key := rc.hash(query)

	rc.mu.Lock()
	defer rc.mu.Unlock()

	delete(rc.entries, key)
	rc.removeFromOrder(key)
}

// Clear removes all entries from the cache.
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.entries = make(map[string]*CacheEntry)
	rc.order = rc.order[:0]
}

// Size returns the number of entries in the cache.
func (rc *ResponseCache) Size() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.entries)
}

// Stats returns cache statistics.
func (rc *ResponseCache) Stats() (size int, totalHits int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for _, entry := range rc.entries {
		totalHits += entry.HitCount
	}
	return len(rc.entries), totalHits
}

func (rc *ResponseCache) hash(query string) string {
	h := sha256.Sum256([]byte(query))
	return hex.EncodeToString(h[:16])
}

func (rc *ResponseCache) evictOldest() {
	if len(rc.order) == 0 {
		return
	}
	oldest := rc.order[0]
	rc.order = rc.order[1:]
	delete(rc.entries, oldest)
}

func (rc *ResponseCache) removeFromOrder(key string) {
	for i, k := range rc.order {
		if k == key {
			rc.order = append(rc.order[:i], rc.order[i+1:]...)
			return
		}
	}
}
