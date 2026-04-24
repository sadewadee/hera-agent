package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Hook defines pre/post-message processing hooks for the gateway.
type Hook interface {
	// Name returns a unique identifier for the hook.
	Name() string

	// BeforeMessage is called before a message is processed by the agent.
	// It may modify the message or return an error to reject it.
	BeforeMessage(ctx context.Context, msg *IncomingMessage) (*IncomingMessage, error)

	// AfterMessage is called after the agent has produced a response.
	AfterMessage(ctx context.Context, msg *IncomingMessage, response string) error
}

// HookManager manages an ordered list of hooks and executes them in sequence.
type HookManager struct {
	mu    sync.RWMutex
	hooks []Hook
}

// NewHookManager creates a new HookManager.
func NewHookManager() *HookManager {
	return &HookManager{
		hooks: make([]Hook, 0),
	}
}

// Register adds a hook to the end of the pipeline.
func (hm *HookManager) Register(hook Hook) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.hooks = append(hm.hooks, hook)
}

// Unregister removes a hook by name.
func (hm *HookManager) Unregister(name string) bool {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	for i, h := range hm.hooks {
		if h.Name() == name {
			hm.hooks = append(hm.hooks[:i], hm.hooks[i+1:]...)
			return true
		}
	}
	return false
}

// RunBefore executes all BeforeMessage hooks in order.
// Returns the (possibly modified) message or an error if any hook rejects it.
func (hm *HookManager) RunBefore(ctx context.Context, msg *IncomingMessage) (*IncomingMessage, error) {
	hm.mu.RLock()
	hooks := make([]Hook, len(hm.hooks))
	copy(hooks, hm.hooks)
	hm.mu.RUnlock()

	current := msg
	for _, h := range hooks {
		modified, err := h.BeforeMessage(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("hook %s before: %w", h.Name(), err)
		}
		if modified != nil {
			current = modified
		}
	}
	return current, nil
}

// RunAfter executes all AfterMessage hooks in order.
func (hm *HookManager) RunAfter(ctx context.Context, msg *IncomingMessage, response string) error {
	hm.mu.RLock()
	hooks := make([]Hook, len(hm.hooks))
	copy(hooks, hm.hooks)
	hm.mu.RUnlock()

	for _, h := range hooks {
		if err := h.AfterMessage(ctx, msg, response); err != nil {
			return fmt.Errorf("hook %s after: %w", h.Name(), err)
		}
	}
	return nil
}

// Hooks returns the names of all registered hooks.
func (hm *HookManager) Hooks() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	names := make([]string, len(hm.hooks))
	for i, h := range hm.hooks {
		names[i] = h.Name()
	}
	return names
}

// --- Built-in Hooks ---

// LoggingHook logs all incoming and outgoing messages.
type LoggingHook struct {
	logger *slog.Logger
}

// NewLoggingHook creates a hook that logs messages.
func NewLoggingHook() *LoggingHook {
	return &LoggingHook{
		logger: slog.Default().With("component", "hook.logging"),
	}
}

func (h *LoggingHook) Name() string { return "logging" }

func (h *LoggingHook) BeforeMessage(_ context.Context, msg *IncomingMessage) (*IncomingMessage, error) {
	h.logger.Info("incoming message",
		"platform", msg.Platform,
		"user", msg.UserID,
		"chat", msg.ChatID,
		"length", len(msg.Text),
	)
	return msg, nil
}

func (h *LoggingHook) AfterMessage(_ context.Context, msg *IncomingMessage, response string) error {
	h.logger.Info("outgoing response",
		"platform", msg.Platform,
		"user", msg.UserID,
		"chat", msg.ChatID,
		"response_length", len(response),
	)
	return nil
}

// RateLimitHook enforces per-user message rate limits.
type RateLimitHook struct {
	mu      sync.Mutex
	limits  map[string][]time.Time // userKey -> timestamps
	maxMsgs int                    // max messages in window
	window  time.Duration          // time window
}

// NewRateLimitHook creates a rate limiting hook.
// maxMsgs is the maximum messages allowed within the time window.
func NewRateLimitHook(maxMsgs int, window time.Duration) *RateLimitHook {
	if maxMsgs <= 0 {
		maxMsgs = 20
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimitHook{
		limits:  make(map[string][]time.Time),
		maxMsgs: maxMsgs,
		window:  window,
	}
}

func (h *RateLimitHook) Name() string { return "rate_limit" }

func (h *RateLimitHook) BeforeMessage(_ context.Context, msg *IncomingMessage) (*IncomingMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := msg.Platform + ":" + msg.UserID
	now := time.Now()

	// Prune old timestamps.
	cutoff := now.Add(-h.window)
	timestamps := h.limits[key]
	valid := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	if len(valid) >= h.maxMsgs {
		return nil, fmt.Errorf("rate limited: max %d messages per %s", h.maxMsgs, h.window)
	}

	h.limits[key] = append(valid, now)
	return msg, nil
}

func (h *RateLimitHook) AfterMessage(_ context.Context, _ *IncomingMessage, _ string) error {
	return nil
}
