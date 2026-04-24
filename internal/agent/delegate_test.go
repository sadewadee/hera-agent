package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAgent is a test-only AgentRunner that returns a fixed response.
type fakeAgent struct {
	reply string
	err   error
	// captures the last call for assertion
	lastPlatform string
	lastChatID   string
	lastUserID   string
	lastText     string
}

func (f *fakeAgent) HandleMessage(_ context.Context, platform, chatID, userID, text string) (string, error) {
	f.lastPlatform = platform
	f.lastChatID = chatID
	f.lastUserID = userID
	f.lastText = text
	return f.reply, f.err
}

func TestAgentRegistry_RegisterAndGet(t *testing.T) {
	reg := NewAgentRegistry()
	fa := &fakeAgent{reply: "hello"}
	reg.Register("worker", fa)

	got, ok := reg.Get("worker")
	require.True(t, ok, "Get() should find registered agent")
	assert.Equal(t, fa, got)
}

func TestAgentRegistry_GetMissing(t *testing.T) {
	reg := NewAgentRegistry()
	_, ok := reg.Get("missing")
	assert.False(t, ok)
}

func TestAgentRegistry_ReplaceRegistration(t *testing.T) {
	reg := NewAgentRegistry()
	first := &fakeAgent{reply: "first"}
	second := &fakeAgent{reply: "second"}
	reg.Register("worker", first)
	reg.Register("worker", second) // replace
	got, _ := reg.Get("worker")
	assert.Equal(t, second, got, "second registration should replace first")
}

func TestAgentRegistry_Names(t *testing.T) {
	reg := NewAgentRegistry()
	reg.Register("alpha", &fakeAgent{})
	reg.Register("beta", &fakeAgent{})
	names := reg.Names()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
}

func TestAgentRegistry_DelegateTo_Success(t *testing.T) {
	reg := NewAgentRegistry()
	fa := &fakeAgent{reply: "task done"}
	reg.Register("coder", fa)

	resp, err := reg.DelegateTo(context.Background(), "coder", "write a function")
	require.NoError(t, err)
	assert.Equal(t, "task done", resp)
	assert.Equal(t, "delegate", fa.lastPlatform)
	assert.Equal(t, "write a function", fa.lastText)
}

func TestAgentRegistry_DelegateTo_UnknownAgent(t *testing.T) {
	reg := NewAgentRegistry()
	_, err := reg.DelegateTo(context.Background(), "nonexistent", "do something")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agent registered as")
}

func TestAgentRegistry_DelegateTo_AgentError(t *testing.T) {
	reg := NewAgentRegistry()
	fa := &fakeAgent{err: errors.New("LLM unavailable")}
	reg.Register("coder", fa)

	_, err := reg.DelegateTo(context.Background(), "coder", "write something")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM unavailable")
}

func TestAgentRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewAgentRegistry()
	// Concurrent Register + Get — race detector will catch issues.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			reg.Register("concurrent", &fakeAgent{reply: "ok"})
		}
	}()
	for i := 0; i < 50; i++ {
		reg.Get("concurrent")
	}
	<-done
}
