package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProvider is a minimal Provider used to exercise FallbackProvider
// routing logic without spinning up HTTP mocks.
type stubProvider struct {
	name string
	// Per-call queue of responses/errors; exhausted entries return EOF.
	chatErr   error
	chatResp  *ChatResponse
	streamErr error
	streamCh  <-chan StreamEvent
	called    int
}

func (s *stubProvider) Chat(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	s.called++
	return s.chatResp, s.chatErr
}
func (s *stubProvider) ChatStream(_ context.Context, _ ChatRequest) (<-chan StreamEvent, error) {
	s.called++
	return s.streamCh, s.streamErr
}
func (s *stubProvider) CountTokens(_ []Message) (int, error) {
	return 0, nil
}
func (s *stubProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{Provider: s.name, ContextWindow: 4096}
}

func TestFallback_PrimarySucceeds_FallbackNotCalled(t *testing.T) {
	primary := &stubProvider{name: "primary", chatResp: &ChatResponse{Message: Message{Content: "ok"}}}
	secondary := &stubProvider{name: "secondary"}

	f := NewFallbackProvider(primary, "primary", []Provider{secondary}, []string{"secondary"})
	resp, err := f.Chat(context.Background(), ChatRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message.Content)
	assert.Equal(t, 1, primary.called)
	assert.Equal(t, 0, secondary.called, "fallback should not be called when primary succeeds")
}

func TestFallback_OnServerError_SwitchesToFallback(t *testing.T) {
	primary := &stubProvider{name: "primary", chatErr: errors.New("service unavailable (503)")}
	secondary := &stubProvider{name: "secondary", chatResp: &ChatResponse{Message: Message{Content: "from-secondary"}}}

	f := NewFallbackProvider(primary, "primary", []Provider{secondary}, []string{"secondary"})
	resp, err := f.Chat(context.Background(), ChatRequest{})
	require.NoError(t, err)
	assert.Equal(t, "from-secondary", resp.Message.Content)
	assert.Equal(t, 1, primary.called)
	assert.Equal(t, 1, secondary.called)
}

func TestFallback_OnNetworkError_SwitchesToFallback(t *testing.T) {
	primary := &stubProvider{name: "primary", chatErr: errors.New("connection refused")}
	secondary := &stubProvider{name: "secondary", chatResp: &ChatResponse{Message: Message{Content: "ok"}}}

	f := NewFallbackProvider(primary, "primary", []Provider{secondary}, []string{"secondary"})
	_, err := f.Chat(context.Background(), ChatRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, secondary.called)
}

func TestFallback_OnAuthError_DoesNotSwitch(t *testing.T) {
	// Auth errors are the credential pool's job, not fallback's —
	// a different provider would need its own key anyway.
	primary := &stubProvider{name: "primary", chatErr: errors.New("401 unauthorized")}
	secondary := &stubProvider{name: "secondary"}

	f := NewFallbackProvider(primary, "primary", []Provider{secondary}, []string{"secondary"})
	_, err := f.Chat(context.Background(), ChatRequest{})
	require.Error(t, err)
	assert.Equal(t, 0, secondary.called, "auth error must not trigger fallback")
}

func TestFallback_OnContextOverflow_DoesNotSwitch(t *testing.T) {
	primary := &stubProvider{name: "primary", chatErr: errors.New("context length exceeded")}
	secondary := &stubProvider{name: "secondary"}

	f := NewFallbackProvider(primary, "primary", []Provider{secondary}, []string{"secondary"})
	_, err := f.Chat(context.Background(), ChatRequest{})
	require.Error(t, err)
	assert.Equal(t, 0, secondary.called)
}

func TestFallback_AllProvidersFail(t *testing.T) {
	p1 := &stubProvider{name: "p1", chatErr: errors.New("503 service unavailable")}
	p2 := &stubProvider{name: "p2", chatErr: errors.New("504 gateway timeout")}
	p3 := &stubProvider{name: "p3", chatErr: errors.New("network timeout")}

	f := NewFallbackProvider(p1, "p1", []Provider{p2, p3}, []string{"p2", "p3"})
	_, err := f.Chat(context.Background(), ChatRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all providers failed")
	assert.Equal(t, 1, p1.called)
	assert.Equal(t, 1, p2.called)
	assert.Equal(t, 1, p3.called)
}

func TestFallback_EmptyFallbackList_PassThrough(t *testing.T) {
	primary := &stubProvider{name: "primary", chatResp: &ChatResponse{Message: Message{Content: "ok"}}}
	f := NewFallbackProvider(primary, "primary", nil, nil)
	resp, err := f.Chat(context.Background(), ChatRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message.Content)
}

func TestShouldFallback_Classifications(t *testing.T) {
	cases := []struct {
		err  string
		want bool
	}{
		{"500 internal server error", true},
		{"503 service unavailable", true},
		{"504 gateway timeout", true},
		{"connection reset", true},
		{"network timeout", true},
		{"no such host", true},
		{"401 unauthorized", false},
		{"403 forbidden", false},
		{"429 too many requests", false},
		{"context length exceeded", false},
		{"400 bad request", false},
		{"some unknown thing", false},
	}
	for _, tc := range cases {
		got := shouldFallback(errors.New(tc.err))
		if got != tc.want {
			t.Errorf("shouldFallback(%q) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
