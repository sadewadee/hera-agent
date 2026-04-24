package agent

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitTracker monitors rate limit responses from LLM providers
// and integrates with the CredentialPool to mark keys as rate-limited.
type RateLimitTracker struct {
	mu sync.Mutex
	// Per-provider tracking: "provider:key" -> RateLimitInfo
	tracking map[string]*RateLimitInfo
}

// RateLimitInfo holds rate limit state for a single provider/key pair.
type RateLimitInfo struct {
	Provider      string    `json:"provider"`
	Key           string    `json:"key"`
	Limited       bool      `json:"limited"`
	RetryAfter    time.Time `json:"retry_after"`
	RequestsLeft  int       `json:"requests_left"`
	TokensLeft    int       `json:"tokens_left"`
	LastCheckedAt time.Time `json:"last_checked_at"`
}

// NewRateLimitTracker creates a new rate limit tracker.
func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{
		tracking: make(map[string]*RateLimitInfo),
	}
}

// TrackResponse inspects an HTTP response for rate limit headers and 429 status.
// Returns true if the response indicates rate limiting.
func (t *RateLimitTracker) TrackResponse(provider, key string, resp *http.Response) bool {
	if resp == nil {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	trackKey := provider + ":" + key
	info, ok := t.tracking[trackKey]
	if !ok {
		info = &RateLimitInfo{
			Provider: provider,
			Key:      key,
		}
		t.tracking[trackKey] = info
	}
	info.LastCheckedAt = time.Now()

	// Parse rate limit headers (common across OpenAI, Anthropic, etc.).
	if v := resp.Header.Get("X-RateLimit-Remaining-Requests"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			info.RequestsLeft = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Remaining-Tokens"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			info.TokensLeft = n
		}
	}

	// Handle 429 Too Many Requests.
	if resp.StatusCode == http.StatusTooManyRequests {
		info.Limited = true
		backoff := parseRetryAfter(resp)
		info.RetryAfter = time.Now().Add(backoff)
		return true
	}

	info.Limited = false
	return false
}

// IsLimited reports whether a provider/key pair is currently rate-limited.
func (t *RateLimitTracker) IsLimited(provider, key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	trackKey := provider + ":" + key
	info, ok := t.tracking[trackKey]
	if !ok {
		return false
	}
	if !info.Limited {
		return false
	}
	if time.Now().After(info.RetryAfter) {
		info.Limited = false
		return false
	}
	return true
}

// GetBackoff returns the remaining backoff duration for a rate-limited key.
// Returns 0 if the key is not rate-limited.
func (t *RateLimitTracker) GetBackoff(provider, key string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	trackKey := provider + ":" + key
	info, ok := t.tracking[trackKey]
	if !ok || !info.Limited {
		return 0
	}
	remaining := time.Until(info.RetryAfter)
	if remaining < 0 {
		info.Limited = false
		return 0
	}
	return remaining
}

// GetInfo returns rate limit info for a provider/key pair.
func (t *RateLimitTracker) GetInfo(provider, key string) *RateLimitInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	trackKey := provider + ":" + key
	info, ok := t.tracking[trackKey]
	if !ok {
		return nil
	}
	// Return a copy.
	cp := *info
	return &cp
}

// parseRetryAfter extracts a backoff duration from the Retry-After header.
// Falls back to 60 seconds if the header is missing or unparseable.
func parseRetryAfter(resp *http.Response) time.Duration {
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return 60 * time.Second
	}

	// Try seconds first.
	val = strings.TrimSpace(val)
	if secs, err := strconv.Atoi(val); err == nil {
		return time.Duration(secs) * time.Second
	}

	// Try HTTP-date format.
	if t, err := time.Parse(time.RFC1123, val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 60 * time.Second
}
