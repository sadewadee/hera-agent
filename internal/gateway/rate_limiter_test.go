package gateway

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	// First 3 should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow("user1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th should be denied
	if rl.Allow("user1") {
		t.Error("4th request should be denied")
	}
}

func TestRateLimiter_DifferentUsers(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)

	rl.Allow("user1")
	rl.Allow("user1")

	// user1 is exhausted
	if rl.Allow("user1") {
		t.Error("user1 should be rate limited")
	}

	// user2 should still be allowed
	if !rl.Allow("user2") {
		t.Error("user2 should be allowed")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)

	rl.Allow("user1")
	rl.Allow("user1")

	if rl.Allow("user1") {
		t.Error("should be limited before refill")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("user1") {
		t.Error("should be allowed after refill window")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)

	if r := rl.Remaining("new_user"); r != 5 {
		t.Errorf("Remaining = %d, want 5", r)
	}

	rl.Allow("new_user")
	rl.Allow("new_user")

	if r := rl.Remaining("new_user"); r != 3 {
		t.Errorf("Remaining = %d, want 3", r)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)

	rl.Allow("user1")
	rl.Allow("user1")
	rl.Reset("user1")

	if !rl.Allow("user1") {
		t.Error("should be allowed after reset")
	}
}

func TestRateLimiter_CleanIdle(t *testing.T) {
	rl := NewRateLimiter(10, time.Second)

	rl.Allow("user1")
	rl.Allow("user2")

	time.Sleep(60 * time.Millisecond)
	removed := rl.CleanIdle(50 * time.Millisecond)

	if removed != 2 {
		t.Errorf("CleanIdle removed %d, want 2", removed)
	}
}
