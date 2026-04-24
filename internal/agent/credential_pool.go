// Package agent provides the core agent runtime.
//
// credential_pool.go implements a multi-key credential pool with rotation
// strategies for same-provider failover. This extends the basic pool in
// internal/llm/pool.go with per-credential metadata, priority ordering,
// and persistent status tracking.
package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Credential status constants.
const (
	StatusOK        = "ok"
	StatusExhausted = "exhausted"
)

// Auth type constants.
const (
	AuthTypeOAuth  = "oauth"
	AuthTypeAPIKey = "api_key"
)

// Source constants.
const (
	SourceManual = "manual"
)

// Pool rotation strategy constants.
const (
	StrategyFillFirst  = "fill_first"
	StrategyRoundRobin = "round_robin"
	StrategyRandom     = "random"
	StrategyLeastUsed  = "least_used"
)

// Cooldown before retrying an exhausted credential.
const (
	ExhaustedTTL429Seconds     = 3600 // 1 hour for rate-limited
	ExhaustedTTLDefaultSeconds = 3600 // 1 hour default
)

// CustomPoolPrefix is the key prefix for custom OpenAI-compatible endpoints.
const CustomPoolPrefix = "custom:"

// SupportedPoolStrategies lists all valid rotation strategies.
var SupportedPoolStrategies = map[string]bool{
	StrategyFillFirst:  true,
	StrategyRoundRobin: true,
	StrategyRandom:     true,
	StrategyLeastUsed:  true,
}

// PooledCredential represents a single credential in the pool with metadata.
type PooledCredential struct {
	Provider         string            `json:"provider"`
	ID               string            `json:"id"`
	Label            string            `json:"label"`
	AuthType         string            `json:"auth_type"`
	Priority         int               `json:"priority"`
	Source           string            `json:"source"`
	AccessToken      string            `json:"access_token"`
	RefreshToken     string            `json:"refresh_token,omitempty"`
	LastStatus       string            `json:"last_status,omitempty"`
	LastStatusAt     float64           `json:"last_status_at,omitempty"`
	LastErrorCode    int               `json:"last_error_code,omitempty"`
	LastErrorReason  string            `json:"last_error_reason,omitempty"`
	LastErrorMessage string            `json:"last_error_message,omitempty"`
	LastErrorResetAt float64           `json:"last_error_reset_at,omitempty"`
	BaseURL          string            `json:"base_url,omitempty"`
	ExpiresAt        string            `json:"expires_at,omitempty"`
	ExpiresAtMs      int64             `json:"expires_at_ms,omitempty"`
	LastRefresh      string            `json:"last_refresh,omitempty"`
	InferenceBaseURL string            `json:"inference_base_url,omitempty"`
	AgentKey         string            `json:"agent_key,omitempty"`
	AgentKeyExpires  string            `json:"agent_key_expires_at,omitempty"`
	RequestCount     int64             `json:"request_count"`
	Extra            map[string]string `json:"extra,omitempty"`
}

// RuntimeAPIKey returns the API key to use at runtime.
// For "nous" provider, agent_key takes precedence over access_token.
func (c *PooledCredential) RuntimeAPIKey() string {
	if c.Provider == "nous" {
		if c.AgentKey != "" {
			return c.AgentKey
		}
		return c.AccessToken
	}
	return c.AccessToken
}

// RuntimeBaseURL returns the base URL to use at runtime.
// For "nous" provider, inference_base_url takes precedence.
func (c *PooledCredential) RuntimeBaseURL() string {
	if c.Provider == "nous" {
		if c.InferenceBaseURL != "" {
			return c.InferenceBaseURL
		}
	}
	return c.BaseURL
}

// NewPooledCredential creates a PooledCredential from a map of values,
// filling in defaults where needed.
func NewPooledCredential(provider string, payload map[string]string) *PooledCredential {
	c := &PooledCredential{
		Provider:    provider,
		ID:          payload["id"],
		Label:       payload["label"],
		AuthType:    payload["auth_type"],
		Source:      payload["source"],
		AccessToken: payload["access_token"],
		Extra:       make(map[string]string),
	}
	if c.ID == "" {
		c.ID = randomHex(6)
	}
	if c.Label == "" {
		if s, ok := payload["source"]; ok {
			c.Label = s
		} else {
			c.Label = provider
		}
	}
	if c.AuthType == "" {
		c.AuthType = AuthTypeAPIKey
	}
	if c.Source == "" {
		c.Source = SourceManual
	}
	if p, ok := payload["priority"]; ok {
		c.Priority, _ = strconv.Atoi(p)
	}
	return c
}

// LabelFromToken extracts a human-readable label from a JWT token's claims.
func LabelFromToken(token, fallback string) string {
	// Simplified JWT claims extraction (no full decode in Go without deps).
	// Return fallback for now; full JWT decoding is in internal/acp/auth.go.
	if token == "" {
		return fallback
	}
	return fallback
}

// NextPriority returns the next priority value for a list of credentials.
func NextPriority(entries []*PooledCredential) int {
	maxP := -1
	for _, e := range entries {
		if e.Priority > maxP {
			maxP = e.Priority
		}
	}
	return maxP + 1
}

// IsManualSource returns true if source is "manual" or "manual:*".
func IsManualSource(source string) bool {
	normalized := strings.TrimSpace(strings.ToLower(source))
	return normalized == SourceManual || strings.HasPrefix(normalized, SourceManual+":")
}

// ExhaustedTTL returns cooldown seconds based on the HTTP status that caused
// exhaustion.
func ExhaustedTTL(errorCode int) int {
	if errorCode == 429 {
		return ExhaustedTTL429Seconds
	}
	return ExhaustedTTLDefaultSeconds
}

// ParseAbsoluteTimestamp parses a provider reset timestamp.
// Accepts epoch seconds, epoch milliseconds, and ISO-8601 strings.
func ParseAbsoluteTimestamp(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		if f <= 0 {
			return 0, false
		}
		if f > 1_000_000_000_000 {
			return f / 1000.0, true
		}
		return f, true
	}
	// Attempt ISO-8601 parse.
	value = strings.Replace(value, "Z", "+00:00", 1)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, value); err == nil {
			return float64(t.Unix()), true
		}
	}
	return 0, false
}

var reQuotaResetDelay = regexp.MustCompile(`(?i)quotaResetDelay[:\s"]+(\d+(?:\.\d+)?)(ms|s)`)
var reRetryAfter = regexp.MustCompile(`(?i)retry\s+(?:after\s+)?(\d+(?:\.\d+)?)\s*(?:sec|secs|seconds|s\b)`)

// ExtractRetryDelaySeconds attempts to extract a retry delay from an error
// message. Returns the delay in seconds and true if found.
func ExtractRetryDelaySeconds(message string) (float64, bool) {
	if message == "" {
		return 0, false
	}
	if m := reQuotaResetDelay.FindStringSubmatch(message); m != nil {
		val, _ := strconv.ParseFloat(m[1], 64)
		if strings.ToLower(m[2]) == "ms" {
			return val / 1000.0, true
		}
		return val, true
	}
	if m := reRetryAfter.FindStringSubmatch(message); m != nil {
		val, _ := strconv.ParseFloat(m[1], 64)
		return val, true
	}
	return 0, false
}

// NormalizeErrorContext cleans up an error context map, extracting
// standard fields (reason, message, reset_at, retry_after).
func NormalizeErrorContext(ctx map[string]string) map[string]string {
	if len(ctx) == 0 {
		return map[string]string{}
	}
	normalized := make(map[string]string)
	if reason, ok := ctx["reason"]; ok && strings.TrimSpace(reason) != "" {
		normalized["reason"] = strings.TrimSpace(reason)
	}
	if msg, ok := ctx["message"]; ok && strings.TrimSpace(msg) != "" {
		normalized["message"] = strings.TrimSpace(msg)
	}
	if resetAt, ok := ctx["reset_at"]; ok && resetAt != "" {
		normalized["reset_at"] = resetAt
	}
	if retryAfter, ok := ctx["retry_after"]; ok && retryAfter != "" {
		normalized["retry_after"] = retryAfter
	}
	return normalized
}

// CredentialPool manages multiple credentials for a single provider
// with rotation strategies.
type CredentialPool struct {
	mu          sync.Mutex
	provider    string
	entries     []*PooledCredential
	strategy    string
	idx         int
	rateLimited map[string]time.Time
}

// NewCredentialPool creates a credential pool for a provider.
func NewCredentialPool(provider string, strategy string) *CredentialPool {
	if !SupportedPoolStrategies[strategy] {
		strategy = StrategyRoundRobin
	}
	return &CredentialPool{
		provider:    provider,
		strategy:    strategy,
		rateLimited: make(map[string]time.Time),
	}
}

// Add adds a credential to the pool.
func (p *CredentialPool) Add(cred *PooledCredential) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = append(p.entries, cred)
}

// Next returns the next available credential using the configured strategy.
func (p *CredentialPool) Next() *PooledCredential {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.entries) == 0 {
		return nil
	}

	now := time.Now()
	n := len(p.entries)

	switch p.strategy {
	case StrategyRoundRobin:
		for i := 0; i < n; i++ {
			idx := (p.idx + i) % n
			cred := p.entries[idx]
			if expires, limited := p.rateLimited[cred.ID]; !limited || now.After(expires) {
				delete(p.rateLimited, cred.ID)
				p.idx = (idx + 1) % n
				return cred
			}
		}
	case StrategyLeastUsed:
		var best *PooledCredential
		for _, cred := range p.entries {
			if expires, limited := p.rateLimited[cred.ID]; limited && !now.After(expires) {
				continue
			}
			delete(p.rateLimited, cred.ID)
			if best == nil || cred.RequestCount < best.RequestCount {
				best = cred
			}
		}
		if best != nil {
			return best
		}
	case StrategyFillFirst:
		for _, cred := range p.entries {
			if expires, limited := p.rateLimited[cred.ID]; limited && !now.After(expires) {
				continue
			}
			delete(p.rateLimited, cred.ID)
			return cred
		}
	}

	// All rate-limited: return the one expiring soonest.
	var bestCred *PooledCredential
	var bestExpiry time.Time
	for _, cred := range p.entries {
		expires := p.rateLimited[cred.ID]
		if bestCred == nil || expires.Before(bestExpiry) {
			bestCred = cred
			bestExpiry = expires
		}
	}
	return bestCred
}

// MarkExhausted marks a credential as rate-limited for the given duration.
func (p *CredentialPool) MarkExhausted(credID string, backoff time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimited[credID] = time.Now().Add(backoff)
	slog.Debug("credential marked exhausted",
		"provider", p.provider,
		"credential_id", credID,
		"backoff", backoff,
	)
}

// Size returns the number of credentials in the pool.
func (p *CredentialPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// GetCustomProviderPoolKey returns the pool key for a custom provider.
func GetCustomProviderPoolKey(name string) string {
	normalized := strings.TrimSpace(strings.ToLower(name))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return fmt.Sprintf("%s%s", CustomPoolPrefix, normalized)
}
