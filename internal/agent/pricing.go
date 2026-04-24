package agent

import (
	"fmt"
	"sync"

	"github.com/sadewadee/hera/internal/llm"
)

// UsageTracker accumulates token usage and cost per session.
type UsageTracker struct {
	mu       sync.Mutex
	sessions map[string]*SessionCost // sessionID -> cost
}

// SessionCost tracks cost for a single session.
type SessionCost struct {
	SessionID        string  `json:"session_id"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd"`
	Model            string  `json:"model"`
}

// NewUsageTracker creates a new usage tracker.
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		sessions: make(map[string]*SessionCost),
	}
}

// Track records token usage for a session using model metadata pricing.
func (ut *UsageTracker) Track(sessionID string, usage llm.Usage, modelID string) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	sc, ok := ut.sessions[sessionID]
	if !ok {
		sc = &SessionCost{SessionID: sessionID, Model: modelID}
		ut.sessions[sessionID] = sc
	}

	sc.PromptTokens += usage.PromptTokens
	sc.CompletionTokens += usage.CompletionTokens
	sc.TotalTokens += usage.TotalTokens

	// Look up pricing from model metadata database.
	meta, found := llm.LookupModel(modelID)
	if found {
		inCost := float64(usage.PromptTokens) / 1000.0 * meta.CostPer1kIn
		outCost := float64(usage.CompletionTokens) / 1000.0 * meta.CostPer1kOut
		sc.CostUSD += inCost + outCost
	}
}

// GetSessionCost returns cost info for a session.
func (ut *UsageTracker) GetSessionCost(sessionID string) *SessionCost {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	sc, ok := ut.sessions[sessionID]
	if !ok {
		return nil
	}
	cp := *sc
	return &cp
}

// TotalCost returns the aggregate cost across all sessions.
func (ut *UsageTracker) TotalCost() float64 {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	var total float64
	for _, sc := range ut.sessions {
		total += sc.CostUSD
	}
	return total
}

// TotalTokens returns the aggregate tokens across all sessions.
func (ut *UsageTracker) TotalTokens() (prompt, completion int) {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	for _, sc := range ut.sessions {
		prompt += sc.PromptTokens
		completion += sc.CompletionTokens
	}
	return
}

// Format returns a human-readable cost summary.
func (ut *UsageTracker) Format() string {
	prompt, completion := ut.TotalTokens()
	cost := ut.TotalCost()
	return fmt.Sprintf("Tokens: %d prompt + %d completion = %d total | Cost: $%.6f",
		prompt, completion, prompt+completion, cost)
}

// Reset clears all tracked sessions.
func (ut *UsageTracker) Reset() {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	ut.sessions = make(map[string]*SessionCost)
}
