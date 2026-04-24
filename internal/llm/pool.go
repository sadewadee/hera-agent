package llm

import (
	"net/http"
	"sync"
	"time"
)

// Cooldown durations per failure class.
const (
	Cooldown401 = 1 * time.Hour   // stale/revoked credential — long timeout
	Cooldown429 = 1 * time.Minute // short rate-limit
	CooldownNet = 10 * time.Second
)

// CredentialPool provides round-robin API key rotation with rate limit
// tracking. A failed key is quarantined for a duration that depends on
// the HTTP status: 401/403 get a long cooldown (key likely revoked);
// 429 gets a short cooldown; network errors a very short one.
type CredentialPool struct {
	mu   sync.Mutex
	keys []string
	idx  int

	// rateLimited tracks when each key's rate limit expires.
	rateLimited map[string]time.Time
}

// NewCredentialPool creates a new credential pool with the given API keys.
// Empty strings are skipped so callers don't have to pre-clean.
func NewCredentialPool(keys []string) *CredentialPool {
	cleaned := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" {
			cleaned = append(cleaned, k)
		}
	}
	return &CredentialPool{
		keys:        cleaned,
		rateLimited: make(map[string]time.Time),
	}
}

// Next returns the next available API key using round-robin selection.
// Rate-limited keys are skipped. If all keys are rate-limited, the key
// whose rate limit expires soonest is returned.
func (p *CredentialPool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.keys) == 0 {
		return ""
	}

	now := time.Now()
	n := len(p.keys)

	// Try to find a non-rate-limited key starting from current index.
	for i := 0; i < n; i++ {
		idx := (p.idx + i) % n
		key := p.keys[idx]
		if expires, limited := p.rateLimited[key]; !limited || now.After(expires) {
			// Clean up expired entry.
			delete(p.rateLimited, key)
			p.idx = (idx + 1) % n
			return key
		}
	}

	// All keys are rate-limited. Return the one that expires soonest.
	var bestKey string
	var bestExpiry time.Time
	for _, key := range p.keys {
		expires := p.rateLimited[key]
		if bestKey == "" || expires.Before(bestExpiry) {
			bestKey = key
			bestExpiry = expires
		}
	}
	// Advance index past this key.
	for i, key := range p.keys {
		if key == bestKey {
			p.idx = (i + 1) % n
			break
		}
	}
	return bestKey
}

// MarkRateLimited marks a key as rate-limited for the given duration.
func (p *CredentialPool) MarkRateLimited(key string, backoff time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimited[key] = time.Now().Add(backoff)
}

// IsRateLimited returns whether the given key is currently rate-limited.
func (p *CredentialPool) IsRateLimited(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	expires, ok := p.rateLimited[key]
	if !ok {
		return false
	}
	if time.Now().After(expires) {
		delete(p.rateLimited, key)
		return false
	}
	return true
}

// MarkFailure quarantines the key based on the HTTP status. 0 means a
// network-level failure (no response). This is the preferred entry point
// for providers: they see the status code and hand it over without
// having to know the backoff duration themselves.
func (p *CredentialPool) MarkFailure(key string, status int) {
	if key == "" {
		return
	}
	var d time.Duration
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		d = Cooldown401
	case status == http.StatusTooManyRequests:
		d = Cooldown429
	default:
		d = CooldownNet
	}
	p.MarkRateLimited(key, d)
}

// MarkSuccess clears any cooldown on the key so it re-enters the active
// rotation immediately.
func (p *CredentialPool) MarkSuccess(key string) {
	if key == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.rateLimited, key)
}

// Size returns the number of configured keys. Safe on a nil receiver
// so callers (providers) don't have to nil-check before every question.
func (p *CredentialPool) Size() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.keys)
}

// BuildCredentialPool constructs a pool from a ProviderConfig. APIKeys
// (the new multi-key field) takes precedence; APIKey (legacy single key)
// is appended if not already present. Returns nil when no keys are set
// — providers that accept keyless use (ollama, local servers) handle
// the nil case.
func BuildCredentialPool(cfg ProviderConfig) *CredentialPool {
	keys := make([]string, 0, len(cfg.APIKeys)+1)
	seen := make(map[string]bool)
	for _, k := range cfg.APIKeys {
		if k != "" && !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	if cfg.APIKey != "" && !seen[cfg.APIKey] {
		keys = append(keys, cfg.APIKey)
	}
	if len(keys) == 0 {
		return nil
	}
	return NewCredentialPool(keys)
}
