package agent

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// JitteredBackoff implements exponential backoff with jitter for retrying operations.
type JitteredBackoff struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	MaxRetries int
	attempt    int
}

// NewJitteredBackoff creates a new backoff with sensible defaults.
func NewJitteredBackoff() *JitteredBackoff {
	return &JitteredBackoff{
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		MaxRetries: 5,
	}
}

// Next returns the next backoff duration with jitter and advances the attempt counter.
func (b *JitteredBackoff) Next() time.Duration {
	if b.attempt >= b.MaxRetries { return b.MaxDelay }
	delay := float64(b.BaseDelay) * math.Pow(2, float64(b.attempt))
	if delay > float64(b.MaxDelay) { delay = float64(b.MaxDelay) }
	jitter := delay * 0.5 * rand.Float64()
	b.attempt++
	return time.Duration(delay + jitter)
}

// Reset resets the attempt counter.
func (b *JitteredBackoff) Reset() { b.attempt = 0 }

// Exhausted returns true when max retries have been consumed.
func (b *JitteredBackoff) Exhausted() bool { return b.attempt >= b.MaxRetries }

// RetryWithBackoff executes fn with jittered backoff on error.
func RetryWithBackoff(ctx context.Context, maxRetries int, fn func() error) error {
	bo := &JitteredBackoff{BaseDelay: 500 * time.Millisecond, MaxDelay: 30 * time.Second, MaxRetries: maxRetries}
	var lastErr error
	for !bo.Exhausted() {
		if err := fn(); err != nil {
			lastErr = err
			select {
			case <-ctx.Done(): return ctx.Err()
			case <-time.After(bo.Next()):
			}
			continue
		}
		return nil
	}
	return lastErr
}
