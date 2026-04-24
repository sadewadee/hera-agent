package agent

import (
	"testing"
	"time"
)

func TestResponseCache_SetAndGet(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)

	cache.Set("hello", "world")
	val, ok := cache.Get("hello")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if val != "world" {
		t.Errorf("got %q, want %q", val, "world")
	}
}

func TestResponseCache_Miss(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)

	_, ok := cache.Get("missing")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestResponseCache_Eviction(t *testing.T) {
	cache := NewResponseCache(3, time.Minute)

	cache.Set("a", "1")
	cache.Set("b", "2")
	cache.Set("c", "3")
	cache.Set("d", "4") // Should evict "a"

	_, ok := cache.Get("a")
	if ok {
		t.Error("expected 'a' to be evicted")
	}

	val, ok := cache.Get("d")
	if !ok || val != "4" {
		t.Error("expected 'd' to be present")
	}
}

func TestResponseCache_Invalidate(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)

	cache.Set("key", "value")
	cache.Invalidate("key")

	_, ok := cache.Get("key")
	if ok {
		t.Error("expected cache miss after invalidation")
	}
}

func TestResponseCache_Clear(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)

	cache.Set("a", "1")
	cache.Set("b", "2")
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Size() = %d, want 0", cache.Size())
	}
}

func TestResponseCache_TTL(t *testing.T) {
	cache := NewResponseCache(10, 50*time.Millisecond)

	cache.Set("key", "value")

	// Should hit immediately
	_, ok := cache.Get("key")
	if !ok {
		t.Fatal("expected cache hit before TTL")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	_, ok = cache.Get("key")
	if ok {
		t.Error("expected cache miss after TTL")
	}
}

func TestResponseCache_Stats(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)

	cache.Set("a", "1")
	cache.Set("b", "2")
	cache.Get("a")
	cache.Get("a")
	cache.Get("b")

	size, hits := cache.Stats()
	if size != 2 {
		t.Errorf("size = %d, want 2", size)
	}
	if hits != 3 {
		t.Errorf("hits = %d, want 3", hits)
	}
}
