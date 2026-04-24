package agent

import (
	"context"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareChain_NewChain(t *testing.T) {
	mc := NewMiddlewareChain()
	require.NotNil(t, mc)
	assert.Equal(t, 0, mc.Len())
}

func TestMiddlewareChain_Use(t *testing.T) {
	mc := NewMiddlewareChain()
	mc.Use(LoggingMiddleware())
	assert.Equal(t, 1, mc.Len())
}

func TestMiddlewareChain_Execute_NoMiddleware(t *testing.T) {
	mc := NewMiddlewareChain()
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	handler := func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	}
	result, err := mc.Execute(context.Background(), msgs, handler)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "hello", result[0].Content)
}

func TestMiddlewareChain_Execute_SingleMiddleware(t *testing.T) {
	mc := NewMiddlewareChain()
	mc.Use(func(_ context.Context, msgs []llm.Message, next NextFunc) ([]llm.Message, error) {
		// Add a prefix to the first message
		msgs[0].Content = "modified: " + msgs[0].Content
		return next(context.Background(), msgs)
	})

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	handler := func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	}

	result, err := mc.Execute(context.Background(), msgs, handler)
	require.NoError(t, err)
	assert.Equal(t, "modified: hello", result[0].Content)
}

func TestMiddlewareChain_Execute_OrderPreserved(t *testing.T) {
	mc := NewMiddlewareChain()
	var order []string

	mc.Use(func(_ context.Context, msgs []llm.Message, next NextFunc) ([]llm.Message, error) {
		order = append(order, "first")
		return next(context.Background(), msgs)
	})
	mc.Use(func(_ context.Context, msgs []llm.Message, next NextFunc) ([]llm.Message, error) {
		order = append(order, "second")
		return next(context.Background(), msgs)
	})

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	handler := func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		order = append(order, "handler")
		return m, nil
	}

	_, err := mc.Execute(context.Background(), msgs, handler)
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second", "handler"}, order)
}

func TestTokenLimitMiddleware_UnderLimit(t *testing.T) {
	mw := TokenLimitMiddleware(10)
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
		{Role: llm.RoleUser, Content: "hello"},
	}

	result, err := mw(context.Background(), msgs, func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	})
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTokenLimitMiddleware_OverLimit_PreservesSystemMessage(t *testing.T) {
	mw := TokenLimitMiddleware(3)
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system prompt"},
		{Role: llm.RoleUser, Content: "msg1"},
		{Role: llm.RoleAssistant, Content: "msg2"},
		{Role: llm.RoleUser, Content: "msg3"},
		{Role: llm.RoleAssistant, Content: "msg4"},
	}

	result, err := mw(context.Background(), msgs, func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	})
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, llm.RoleSystem, result[0].Role) // system preserved
}

func TestTokenLimitMiddleware_OverLimit_NoSystemMessage(t *testing.T) {
	mw := TokenLimitMiddleware(2)
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "msg1"},
		{Role: llm.RoleAssistant, Content: "msg2"},
		{Role: llm.RoleUser, Content: "msg3"},
	}

	result, err := mw(context.Background(), msgs, func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	})
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTokenLimitMiddleware_ZeroLimit(t *testing.T) {
	mw := TokenLimitMiddleware(0)
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "msg1"},
		{Role: llm.RoleAssistant, Content: "msg2"},
	}

	result, err := mw(context.Background(), msgs, func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	})
	require.NoError(t, err)
	assert.Len(t, result, 2) // no truncation when limit is 0
}

func TestLoggingMiddleware(t *testing.T) {
	mw := LoggingMiddleware()
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	result, err := mw(context.Background(), msgs, func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	})
	require.NoError(t, err)
	assert.Equal(t, msgs, result)
}

func TestRedactMiddleware(t *testing.T) {
	mw := RedactMiddleware([]string{"password"})
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	result, err := mw(context.Background(), msgs, func(_ context.Context, m []llm.Message) ([]llm.Message, error) {
		return m, nil
	})
	require.NoError(t, err)
	assert.Equal(t, msgs, result)
}
