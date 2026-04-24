package agent

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// BudgetConfig configures token and cost limits for an agent.
// Zero values mean "unlimited".
type BudgetConfig struct {
	// MaxTokens is the maximum cumulative prompt+completion tokens before
	// HandleMessage starts rejecting requests with ErrBudgetExceeded.
	// 0 means unlimited.
	MaxTokens int `json:"max_tokens" yaml:"max_tokens" mapstructure:"max_tokens"`

	// MaxUSD is the maximum cumulative cost in USD. 0 means unlimited.
	MaxUSD float64 `json:"max_usd" yaml:"max_usd" mapstructure:"max_usd"`

	// Window is the time window over which limits are applied. If zero,
	// limits are applied over the entire process lifetime (no reset).
	Window time.Duration `json:"window" yaml:"window" mapstructure:"window"`
}

// ErrBudgetExceeded is returned by HandleMessage when a budget limit is hit.
var ErrBudgetExceeded = fmt.Errorf("agent budget exceeded")

// Budget tracks per-agent token and USD spend and enforces configured limits.
// Thread-safe.
type Budget struct {
	mu          sync.Mutex
	cfg         BudgetConfig
	tokensUsed  int
	costUSD     float64
	windowStart time.Time
}

// NewBudget creates a Budget from the given config. If cfg has no limits, all
// calls to Check() will return nil.
func NewBudget(cfg BudgetConfig) *Budget {
	return &Budget{
		cfg:         cfg,
		windowStart: time.Now(),
	}
}

// Check returns ErrBudgetExceeded if any limit has been reached.
// If a time Window is configured, it resets counters when the window expires
// before checking. Returns nil when no limits are configured.
func (b *Budget) Check() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset window if applicable.
	if b.cfg.Window > 0 && time.Since(b.windowStart) >= b.cfg.Window {
		b.tokensUsed = 0
		b.costUSD = 0
		b.windowStart = time.Now()
	}

	if b.cfg.MaxTokens > 0 && b.tokensUsed >= b.cfg.MaxTokens {
		slog.Warn("agent budget: token limit exceeded",
			"tokens_used", b.tokensUsed,
			"max_tokens", b.cfg.MaxTokens,
		)
		return fmt.Errorf("%w: %d/%d tokens used", ErrBudgetExceeded, b.tokensUsed, b.cfg.MaxTokens)
	}
	if b.cfg.MaxUSD > 0 && b.costUSD >= b.cfg.MaxUSD {
		slog.Warn("agent budget: USD limit exceeded",
			"cost_usd", b.costUSD,
			"max_usd", b.cfg.MaxUSD,
		)
		return fmt.Errorf("%w: $%.4f/$%.4f spent", ErrBudgetExceeded, b.costUSD, b.cfg.MaxUSD)
	}
	return nil
}

// Record adds tokenCount tokens and costUSD dollars to the running totals.
// It also logs a warning if limits are now exceeded (for observability).
func (b *Budget) Record(tokenCount int, costDollars float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokensUsed += tokenCount
	b.costUSD += costDollars
}

// Stats returns a snapshot of current usage.
func (b *Budget) Stats() (tokensUsed int, costUSD float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tokensUsed, b.costUSD
}

// costForTokens computes an approximate USD cost for tokenCount tokens using a
// simple default pricing (GPT-4 class: $0.03/1k in, treated as combined here).
// Callers that know exact provider pricing should use provider.ModelInfo()
// and pass the real cost into Record() directly.
func costForTokens(tokenCount int, costPer1kTokens float64) float64 {
	return float64(tokenCount) / 1000.0 * costPer1kTokens
}
