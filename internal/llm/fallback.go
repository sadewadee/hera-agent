package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// FallbackProvider wraps a primary Provider and an ordered list of
// fallback providers. A call is routed to the primary first; on certain
// error classes (server errors, network-level transients) the call is
// retried against each fallback in order. Auth, rate-limit, invalid
// request, and context-overflow errors are NOT fallback-triggered —
// credential rotation is the credential pool's job, and a fallback
// provider can't fix a malformed request or too-long context.
type FallbackProvider struct {
	primary  Provider
	fallback []Provider
	labels   []string // human-readable provider name per slot, for logs
}

// NewFallbackProvider builds a wrapper. If fallbacks is empty, the
// wrapper is effectively a pass-through around primary — safe to use
// unconditionally.
func NewFallbackProvider(primary Provider, primaryLabel string, fallbacks []Provider, fallbackLabels []string) *FallbackProvider {
	if primary == nil {
		return nil
	}
	labels := append([]string{primaryLabel}, fallbackLabels...)
	return &FallbackProvider{
		primary:  primary,
		fallback: fallbacks,
		labels:   labels,
	}
}

// Chat tries primary then fallbacks until one succeeds or all fail.
func (f *FallbackProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	resp, err := f.primary.Chat(ctx, req)
	if err == nil {
		return resp, nil
	}
	if !shouldFallback(err) {
		return nil, err
	}

	for i, next := range f.fallback {
		slog.Warn("llm: falling back",
			"from", f.labels[0],
			"to", f.labels[i+1],
			"reason", firstLine(err),
		)
		resp, err = next.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !shouldFallback(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("all providers failed; last error: %w", err)
}

// ChatStream tries primary then fallbacks. Returns as soon as one opens
// a stream (the stream itself may still error mid-flight; we don't
// re-route partial streams).
func (f *FallbackProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	ch, err := f.primary.ChatStream(ctx, req)
	if err == nil {
		return ch, nil
	}
	if !shouldFallback(err) {
		return nil, err
	}

	for i, next := range f.fallback {
		slog.Warn("llm: falling back (stream)",
			"from", f.labels[0],
			"to", f.labels[i+1],
			"reason", firstLine(err),
		)
		ch, err = next.ChatStream(ctx, req)
		if err == nil {
			return ch, nil
		}
		if !shouldFallback(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("all providers failed to open stream; last error: %w", err)
}

// CountTokens delegates to primary. Fallback tokenizers may disagree
// but our compression threshold is an estimate anyway, so primary's
// count is good enough.
func (f *FallbackProvider) CountTokens(messages []Message) (int, error) {
	return f.primary.CountTokens(messages)
}

// ModelInfo returns primary's model info (used for context-window
// calculations in the agent; fallbacks are expected to be comparable).
func (f *FallbackProvider) ModelInfo() ModelMetadata {
	return f.primary.ModelInfo()
}

// shouldFallback returns true when the error is provider-side transient
// (outage, network) where a different provider has a chance of succeeding.
// Auth, rate-limit, context-overflow, and invalid-request errors are NOT
// fallback candidates.
func shouldFallback(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	// Keep this synced with internal/agent/errors.go ClassifyError but
	// reimplemented locally to avoid importing the agent package.
	transient := []string{
		"timeout", "timed out",
		"connection reset", "connection refused",
		"eof", "broken pipe",
		"no such host", "temporary", "network",
	}
	server := []string{
		"500", "502", "503", "504",
		"internal server error", "bad gateway",
		"service unavailable", "gateway timeout",
		"server error",
	}
	for _, needle := range transient {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	for _, needle := range server {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// firstLine trims an error message to its first line for clean logs.
func firstLine(err error) string {
	s := err.Error()
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}
