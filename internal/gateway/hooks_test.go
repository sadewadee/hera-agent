package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testHook struct {
	name       string
	beforeErr  error
	afterErr   error
	beforeFn   func(*IncomingMessage) *IncomingMessage
	beforeCalls int
	afterCalls  int
}

func (h *testHook) Name() string { return h.name }
func (h *testHook) BeforeMessage(_ context.Context, msg *IncomingMessage) (*IncomingMessage, error) {
	h.beforeCalls++
	if h.beforeErr != nil {
		return nil, h.beforeErr
	}
	if h.beforeFn != nil {
		return h.beforeFn(msg), nil
	}
	return msg, nil
}
func (h *testHook) AfterMessage(_ context.Context, _ *IncomingMessage, _ string) error {
	h.afterCalls++
	return h.afterErr
}

func TestHookManager_NewHookManager(t *testing.T) {
	hm := NewHookManager()
	require.NotNil(t, hm)
	assert.Empty(t, hm.Hooks())
}

func TestHookManager_Register(t *testing.T) {
	hm := NewHookManager()
	hm.Register(&testHook{name: "test1"})
	hm.Register(&testHook{name: "test2"})
	assert.Equal(t, []string{"test1", "test2"}, hm.Hooks())
}

func TestHookManager_Unregister(t *testing.T) {
	hm := NewHookManager()
	hm.Register(&testHook{name: "test1"})
	hm.Register(&testHook{name: "test2"})

	removed := hm.Unregister("test1")
	assert.True(t, removed)
	assert.Equal(t, []string{"test2"}, hm.Hooks())
}

func TestHookManager_Unregister_NotFound(t *testing.T) {
	hm := NewHookManager()
	removed := hm.Unregister("nonexistent")
	assert.False(t, removed)
}

func TestHookManager_RunBefore_ExecutesInOrder(t *testing.T) {
	var order []string
	hm := NewHookManager()
	hm.Register(&testHook{name: "first", beforeFn: func(msg *IncomingMessage) *IncomingMessage {
		order = append(order, "first")
		return msg
	}})
	hm.Register(&testHook{name: "second", beforeFn: func(msg *IncomingMessage) *IncomingMessage {
		order = append(order, "second")
		return msg
	}})

	msg := &IncomingMessage{Text: "hello"}
	_, err := hm.RunBefore(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order)
}

func TestHookManager_RunBefore_ModifiesMessage(t *testing.T) {
	hm := NewHookManager()
	hm.Register(&testHook{name: "modifier", beforeFn: func(msg *IncomingMessage) *IncomingMessage {
		modified := *msg
		modified.Text = "modified: " + msg.Text
		return &modified
	}})

	msg := &IncomingMessage{Text: "hello"}
	result, err := hm.RunBefore(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "modified: hello", result.Text)
}

func TestHookManager_RunBefore_ErrorStopsPipeline(t *testing.T) {
	hm := NewHookManager()
	h1 := &testHook{name: "failing", beforeErr: errors.New("rejected")}
	h2 := &testHook{name: "second"}
	hm.Register(h1)
	hm.Register(h2)

	msg := &IncomingMessage{Text: "hello"}
	_, err := hm.RunBefore(context.Background(), msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failing")
	assert.Equal(t, 0, h2.beforeCalls)
}

func TestHookManager_RunAfter_ExecutesAll(t *testing.T) {
	hm := NewHookManager()
	h1 := &testHook{name: "first"}
	h2 := &testHook{name: "second"}
	hm.Register(h1)
	hm.Register(h2)

	msg := &IncomingMessage{Text: "hello"}
	err := hm.RunAfter(context.Background(), msg, "response")
	require.NoError(t, err)
	assert.Equal(t, 1, h1.afterCalls)
	assert.Equal(t, 1, h2.afterCalls)
}

func TestHookManager_RunAfter_ErrorStops(t *testing.T) {
	hm := NewHookManager()
	h1 := &testHook{name: "failing", afterErr: errors.New("after error")}
	h2 := &testHook{name: "second"}
	hm.Register(h1)
	hm.Register(h2)

	msg := &IncomingMessage{Text: "hello"}
	err := hm.RunAfter(context.Background(), msg, "response")
	require.Error(t, err)
	assert.Equal(t, 0, h2.afterCalls)
}

func TestLoggingHook_Name(t *testing.T) {
	h := NewLoggingHook()
	assert.Equal(t, "logging", h.Name())
}

func TestLoggingHook_BeforeMessage(t *testing.T) {
	h := NewLoggingHook()
	msg := &IncomingMessage{Platform: "test", UserID: "u1", Text: "hello"}
	result, err := h.BeforeMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, msg, result)
}

func TestLoggingHook_AfterMessage(t *testing.T) {
	h := NewLoggingHook()
	msg := &IncomingMessage{Platform: "test", UserID: "u1"}
	err := h.AfterMessage(context.Background(), msg, "response")
	require.NoError(t, err)
}

func TestRateLimitHook_Name(t *testing.T) {
	h := NewRateLimitHook(5, time.Minute)
	assert.Equal(t, "rate_limit", h.Name())
}

func TestRateLimitHook_DefaultValues(t *testing.T) {
	h := NewRateLimitHook(0, 0)
	assert.Equal(t, 20, h.maxMsgs)
	assert.Equal(t, time.Minute, h.window)
}

func TestRateLimitHook_AllowsUnderLimit(t *testing.T) {
	h := NewRateLimitHook(3, time.Minute)
	msg := &IncomingMessage{Platform: "test", UserID: "u1"}

	for i := 0; i < 3; i++ {
		_, err := h.BeforeMessage(context.Background(), msg)
		require.NoError(t, err)
	}
}

func TestRateLimitHook_BlocksOverLimit(t *testing.T) {
	h := NewRateLimitHook(2, time.Minute)
	msg := &IncomingMessage{Platform: "test", UserID: "u1"}

	h.BeforeMessage(context.Background(), msg)
	h.BeforeMessage(context.Background(), msg)

	_, err := h.BeforeMessage(context.Background(), msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestRateLimitHook_DifferentUsersIndependent(t *testing.T) {
	h := NewRateLimitHook(1, time.Minute)
	msg1 := &IncomingMessage{Platform: "test", UserID: "u1"}
	msg2 := &IncomingMessage{Platform: "test", UserID: "u2"}

	_, err := h.BeforeMessage(context.Background(), msg1)
	require.NoError(t, err)
	_, err = h.BeforeMessage(context.Background(), msg2)
	require.NoError(t, err)
}

func TestRateLimitHook_AfterMessage_NoOp(t *testing.T) {
	h := NewRateLimitHook(5, time.Minute)
	err := h.AfterMessage(context.Background(), nil, "")
	require.NoError(t, err)
}
