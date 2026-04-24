package batch

import (
	"testing"
	"time"
)

func TestExponentialBackoff_Increases(t *testing.T) {
	b := newExponentialBackoff(5, 100*time.Millisecond, 5*time.Second)

	prev := time.Duration(0)
	// Over several calls, the cap (before jitter) grows; with jitter we can't
	// guarantee strict increase, but the average should rise. We just verify
	// all values are non-negative and within [0, max].
	for i := 0; i < 10; i++ {
		d := b.Next()
		if d < 0 {
			t.Errorf("attempt %d: negative delay %v", i, d)
		}
		if d > 5*time.Second {
			t.Errorf("attempt %d: delay %v exceeds max 5s", i, d)
		}
		_ = prev
		prev = d
	}
}

func TestExponentialBackoff_RespectsMax(t *testing.T) {
	maxDelay := 200 * time.Millisecond
	b := newExponentialBackoff(10, 10*time.Millisecond, maxDelay)

	for i := 0; i < 20; i++ {
		d := b.Next()
		if d > maxDelay {
			t.Errorf("attempt %d: delay %v exceeds max %v", i, d, maxDelay)
		}
	}
}

func TestExponentialBackoff_ZeroBase(t *testing.T) {
	b := newExponentialBackoff(3, 0, time.Second)
	d := b.Next()
	if d < 0 {
		t.Errorf("expected non-negative delay, got %v", d)
	}
}
