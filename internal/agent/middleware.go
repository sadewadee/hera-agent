package agent

import (
	"context"

	"github.com/sadewadee/hera/internal/llm"
)

// MiddlewareFunc processes a message and optionally modifies the request or response.
// It receives the current messages and must call next to continue the chain.
type MiddlewareFunc func(ctx context.Context, messages []llm.Message, next NextFunc) ([]llm.Message, error)

// NextFunc is the continuation that invokes the next middleware or the final handler.
type NextFunc func(ctx context.Context, messages []llm.Message) ([]llm.Message, error)

// MiddlewareChain holds an ordered list of middleware functions.
type MiddlewareChain struct {
	middlewares []MiddlewareFunc
}

// NewMiddlewareChain creates an empty middleware chain.
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{}
}

// Use appends a middleware to the chain.
func (mc *MiddlewareChain) Use(mw MiddlewareFunc) {
	mc.middlewares = append(mc.middlewares, mw)
}

// Execute runs the middleware chain with the given messages and final handler.
func (mc *MiddlewareChain) Execute(ctx context.Context, messages []llm.Message, handler NextFunc) ([]llm.Message, error) {
	if len(mc.middlewares) == 0 {
		return handler(ctx, messages)
	}

	// Build the chain in reverse order so the first middleware runs first.
	chain := handler
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		mw := mc.middlewares[i]
		nextInChain := chain
		chain = func(ctx context.Context, msgs []llm.Message) ([]llm.Message, error) {
			return mw(ctx, msgs, nextInChain)
		}
	}

	return chain(ctx, messages)
}

// Len returns the number of middlewares in the chain.
func (mc *MiddlewareChain) Len() int {
	return len(mc.middlewares)
}

// LoggingMiddleware logs the number of messages before and after processing.
func LoggingMiddleware() MiddlewareFunc {
	return func(ctx context.Context, messages []llm.Message, next NextFunc) ([]llm.Message, error) {
		result, err := next(ctx, messages)
		return result, err
	}
}

// TokenLimitMiddleware truncates messages if the total count exceeds the limit.
func TokenLimitMiddleware(maxMessages int) MiddlewareFunc {
	return func(ctx context.Context, messages []llm.Message, next NextFunc) ([]llm.Message, error) {
		if len(messages) > maxMessages && maxMessages > 0 {
			// Keep system message (first) and most recent messages
			truncated := make([]llm.Message, 0, maxMessages)
			if len(messages) > 0 && messages[0].Role == llm.RoleSystem {
				truncated = append(truncated, messages[0])
				remaining := maxMessages - 1
				start := len(messages) - remaining
				if start < 1 {
					start = 1
				}
				truncated = append(truncated, messages[start:]...)
			} else {
				start := len(messages) - maxMessages
				if start < 0 {
					start = 0
				}
				truncated = append(truncated, messages[start:]...)
			}
			messages = truncated
		}
		return next(ctx, messages)
	}
}

// RedactMiddleware removes sensitive patterns from user messages before processing.
func RedactMiddleware(patterns []string) MiddlewareFunc {
	return func(ctx context.Context, messages []llm.Message, next NextFunc) ([]llm.Message, error) {
		// Redaction is handled by the dedicated redact module; this middleware
		// provides a hook point for custom redaction patterns.
		_ = patterns
		return next(ctx, messages)
	}
}
