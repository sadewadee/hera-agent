package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCredentialPool_Next_SingleKey(t *testing.T) {
	pool := NewCredentialPool([]string{"key1"})
	got := pool.Next()
	if got != "key1" {
		t.Errorf("Next() = %q, want %q", got, "key1")
	}
}

func TestCredentialPool_Next_RoundRobin(t *testing.T) {
	pool := NewCredentialPool([]string{"a", "b", "c"})

	results := make([]string, 6)
	for i := 0; i < 6; i++ {
		results[i] = pool.Next()
	}

	want := []string{"a", "b", "c", "a", "b", "c"}
	for i, got := range results {
		if got != want[i] {
			t.Errorf("Next() call %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestCredentialPool_MarkRateLimited(t *testing.T) {
	pool := NewCredentialPool([]string{"a", "b"})

	pool.MarkRateLimited("a", 100*time.Millisecond)

	// a is rate-limited, so Next should skip to b
	got := pool.Next()
	if got != "b" {
		t.Errorf("Next() after marking 'a' rate-limited = %q, want %q", got, "b")
	}

	// After backoff expires, a should be available again
	time.Sleep(150 * time.Millisecond)
	// Advance past b so we wrap around to a
	_ = pool.Next() // gets b (or a if index wrapped)
	// Check that a is now available
	found := false
	for i := 0; i < 3; i++ {
		if pool.Next() == "a" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'a' to be available after backoff expired")
	}
}

func TestCredentialPool_AllRateLimited_ReturnsSomething(t *testing.T) {
	pool := NewCredentialPool([]string{"a", "b"})
	pool.MarkRateLimited("a", 1*time.Hour)
	pool.MarkRateLimited("b", 1*time.Hour)

	// When all keys are rate-limited, still return something (the one closest to expiry)
	got := pool.Next()
	if got == "" {
		t.Error("Next() returned empty string when all keys are rate-limited")
	}
}

func TestCredentialPool_Next_ConcurrentSafe(t *testing.T) {
	pool := NewCredentialPool([]string{"a", "b", "c"})

	var wg sync.WaitGroup
	results := make([]string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = pool.Next()
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		if r == "" {
			t.Errorf("concurrent Next() call %d returned empty string", i)
		}
	}
}

func TestCredentialPool_Empty(t *testing.T) {
	pool := NewCredentialPool([]string{})
	got := pool.Next()
	if got != "" {
		t.Errorf("Next() on empty pool = %q, want empty string", got)
	}
}

func TestCredentialPool_IsRateLimited(t *testing.T) {
	pool := NewCredentialPool([]string{"a", "b"})
	pool.MarkRateLimited("a", 100*time.Millisecond)

	if !pool.IsRateLimited("a") {
		t.Error("IsRateLimited('a') = false, want true")
	}
	if pool.IsRateLimited("b") {
		t.Error("IsRateLimited('b') = true, want false")
	}

	time.Sleep(150 * time.Millisecond)
	if pool.IsRateLimited("a") {
		t.Error("IsRateLimited('a') = true after backoff, want false")
	}
}

func TestCredentialPool_MarkFailure_StatusMapping(t *testing.T) {
	pool := NewCredentialPool([]string{"k1", "k2", "k3"})

	pool.MarkFailure("k1", http.StatusUnauthorized)
	pool.MarkFailure("k2", http.StatusTooManyRequests)
	pool.MarkFailure("k3", 0) // network

	// All three should be rate-limited immediately after marking.
	if !pool.IsRateLimited("k1") {
		t.Error("401 key should be quarantined")
	}
	if !pool.IsRateLimited("k2") {
		t.Error("429 key should be quarantined")
	}
	if !pool.IsRateLimited("k3") {
		t.Error("network-fail key should be quarantined")
	}

	// MarkSuccess clears.
	pool.MarkSuccess("k1")
	if pool.IsRateLimited("k1") {
		t.Error("MarkSuccess should clear cooldown")
	}
}

func TestBuildCredentialPool_Precedence(t *testing.T) {
	cfg := ProviderConfig{
		APIKey:  "legacy",
		APIKeys: []string{"pool1", "pool2"},
	}
	p := BuildCredentialPool(cfg)
	if p == nil {
		t.Fatal("pool should be non-nil")
	}
	if p.Size() != 3 {
		t.Errorf("Size = %d, want 3 (pool1, pool2, legacy)", p.Size())
	}
}

func TestBuildCredentialPool_DedupesLegacyAgainstPool(t *testing.T) {
	cfg := ProviderConfig{
		APIKey:  "dup",
		APIKeys: []string{"dup", "other"},
	}
	p := BuildCredentialPool(cfg)
	if p.Size() != 2 {
		t.Errorf("Size = %d, want 2 (dup deduped)", p.Size())
	}
}

func TestBuildCredentialPool_NoKeys(t *testing.T) {
	if p := BuildCredentialPool(ProviderConfig{}); p != nil {
		t.Errorf("empty config should yield nil pool, got size %d", p.Size())
	}
}

// TestCompatibleProvider_PoolRotatesOn401 is an end-to-end check: two
// keys in the pool, the server rejects the first with 401 once, then
// accepts the second. The provider's built-in retry must recover.
func TestCompatibleProvider_PoolRotatesOn401(t *testing.T) {
	var seenAuths []string
	var mu sync.Mutex
	var bad atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seenAuths = append(seenAuths, r.Header.Get("Authorization"))
		mu.Unlock()

		// Reject "Bearer stale" once.
		if r.Header.Get("Authorization") == "Bearer stale" && !bad.Load() {
			bad.Store(true)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"model":   "test",
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "ok"}, "finish_reason": "stop"}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer srv.Close()

	p, err := NewCompatibleProvider(ProviderConfig{
		BaseURL: srv.URL,
		Model:   "test",
		APIKeys: []string{"stale", "fresh"},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	resp, err := p.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat failed despite pool having a healthy key: %v", err)
	}
	if resp.Message.Content != "ok" {
		t.Errorf("unexpected content %q", resp.Message.Content)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seenAuths) < 2 {
		t.Fatalf("expected >=2 attempts, got %d: %v", len(seenAuths), seenAuths)
	}
	// First attempt used stale, second used fresh.
	if seenAuths[0] != "Bearer stale" {
		t.Errorf("first attempt = %q, want %q", seenAuths[0], "Bearer stale")
	}
	if seenAuths[1] != "Bearer fresh" {
		t.Errorf("second attempt = %q, want %q", seenAuths[1], "Bearer fresh")
	}
}
