package gateway

import (
	"sync"
	"time"
)

// RateLimiter provides per-user rate limiting for the gateway.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     int           // tokens per window
	window   time.Duration // refill window
	maxBurst int           // max tokens per bucket
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a rate limiter with the given rate and window.
// rate is the number of allowed requests per window duration.
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	if rate <= 0 {
		rate = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		window:   window,
		maxBurst: rate,
	}
}

// Allow checks if a request from the given key (user ID or IP) should be allowed.
// Returns true if allowed, false if rate limited.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &tokenBucket{
			tokens:     rl.maxBurst - 1,
			lastRefill: time.Now(),
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := time.Since(bucket.lastRefill)
	if elapsed >= rl.window {
		periods := int(elapsed / rl.window)
		bucket.tokens += periods * rl.rate
		if bucket.tokens > rl.maxBurst {
			bucket.tokens = rl.maxBurst
		}
		bucket.lastRefill = bucket.lastRefill.Add(time.Duration(periods) * rl.window)
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// Remaining returns the number of remaining tokens for the given key.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return rl.maxBurst
	}

	// Refill tokens
	elapsed := time.Since(bucket.lastRefill)
	tokens := bucket.tokens
	if elapsed >= rl.window {
		periods := int(elapsed / rl.window)
		tokens += periods * rl.rate
		if tokens > rl.maxBurst {
			tokens = rl.maxBurst
		}
	}

	return tokens
}

// Reset clears the rate limit state for a specific key.
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, key)
}

// CleanIdle removes buckets that have been idle for longer than the given duration.
func (rl *RateLimiter) CleanIdle(maxIdle time.Duration) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, bucket := range rl.buckets {
		if now.Sub(bucket.lastRefill) > maxIdle {
			delete(rl.buckets, key)
			removed++
		}
	}
	return removed
}
