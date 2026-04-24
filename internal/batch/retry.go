package batch

import (
	"math"
	"math/rand/v2"
	"time"
)

// exponentialBackoff computes per-attempt wait durations with full jitter.
// See: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
type exponentialBackoff struct {
	maxRetries int
	base       time.Duration
	max        time.Duration
	attempt    int
}

// newExponentialBackoff creates a backoff calculator.
// base is the minimum delay; max caps the delay regardless of attempt count.
// maxRetries is stored for reference but not enforced here — the caller decides
// when to stop.
func newExponentialBackoff(maxRetries int, base, max time.Duration) *exponentialBackoff {
	return &exponentialBackoff{
		maxRetries: maxRetries,
		base:       base,
		max:        max,
	}
}

// Next returns the next backoff delay and advances the internal counter.
// It uses full jitter: delay = random(0, min(max, base * 2^attempt)).
func (b *exponentialBackoff) Next() time.Duration {
	exp := math.Pow(2, float64(b.attempt))
	cap := float64(b.base) * exp
	if cap > float64(b.max) {
		cap = float64(b.max)
	}
	// Full jitter: uniform [0, cap)
	jitter := rand.Float64() * cap //nolint:gosec // jitter does not need crypto-safe randomness
	b.attempt++
	return time.Duration(jitter)
}
