package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompatibleProvider_RequiresBaseURL(t *testing.T) {
	_, err := NewCompatibleProvider(ProviderConfig{
		APIKey: "test",
		Model:  "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is required")
}

func TestNewCompatibleProvider_Success(t *testing.T) {
	p, err := NewCompatibleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost:8080/v1",
		Model:   "test-model",
	})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestCompatibleProvider_ModelInfo(t *testing.T) {
	p, _ := NewCompatibleProvider(ProviderConfig{
		BaseURL: "http://localhost:8080/v1",
		Model:   "my-model",
	})
	info := p.ModelInfo()
	assert.Equal(t, "my-model", info.ID)
	assert.Equal(t, "compatible", info.Provider)
	assert.True(t, info.SupportsTools)
}

func TestCompatibleProvider_CountTokens(t *testing.T) {
	p, _ := NewCompatibleProvider(ProviderConfig{
		BaseURL: "http://localhost:8080/v1",
		Model:   "test",
	})
	msgs := []Message{
		{Role: RoleUser, Content: "Hello world, this is a test message"},
	}
	count, err := p.CountTokens(msgs)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestCompatibleProvider_CountTokens_Empty(t *testing.T) {
	p, _ := NewCompatibleProvider(ProviderConfig{
		BaseURL: "http://localhost:8080/v1",
		Model:   "test",
	})
	count, err := p.CountTokens([]Message{{Role: RoleUser, Content: ""}})
	require.NoError(t, err)
	assert.Equal(t, 4, count) // minimum tokens per message
}

func TestCompatibleProvider_Chat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/chat/completions")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]interface{}{"role": "assistant", "content": "Hello!"},
					"finish_reason": "stop",
				},
			},
			"model": "test-model",
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, err := NewCompatibleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Model:   "test-model",
	})
	require.NoError(t, err)

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
		MaxTokens: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Message.Content)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
}

func TestCompatibleProvider_Chat_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer srv.Close()

	p, _ := NewCompatibleProvider(ProviderConfig{
		BaseURL: srv.URL,
		Model:   "test",
	})

	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCompatibleProvider_Chat_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{},
			"model":   "test",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewCompatibleProvider(ProviderConfig{
		BaseURL: srv.URL,
		Model:   "test",
	})

	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty choices")
}

func TestCompatibleProvider_TrimsTrailingSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"role": "assistant", "content": "ok"}, "finish_reason": "stop"},
			},
			"model": "test",
			"usage": map[string]interface{}{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewCompatibleProvider(ProviderConfig{
		BaseURL: srv.URL + "/",
		Model:   "test",
	})

	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message.Content)
}

// TestCompatibleProvider_BuildRequestBody_ToolChoice verifies the fix
// for v0.9.4 bug A: when Tools is non-empty, the request body must
// include a "tool_choice" key so the LLM reliably emits tool calls
// instead of preferring text completion in persona-heavy contexts.
func TestCompatibleProvider_BuildRequestBody_ToolChoice(t *testing.T) {
	p := &CompatibleProvider{model: "claude-sonnet-4", label: "compatible"}

	t.Run("defaults to auto when tools present and ToolChoice empty", func(t *testing.T) {
		body := p.buildRequestBody(ChatRequest{
			Messages: []Message{{Role: RoleUser, Content: "hi"}},
			Tools: []ToolDef{{
				Name:        "memory_note_save",
				Description: "...",
				Parameters:  []byte(`{"type":"object"}`),
			}},
		}, false)
		assert.Equal(t, "auto", body["tool_choice"])
	})

	t.Run("respects explicit ToolChoice", func(t *testing.T) {
		body := p.buildRequestBody(ChatRequest{
			Messages: []Message{{Role: RoleUser, Content: "hi"}},
			Tools: []ToolDef{{
				Name:        "memory_note_save",
				Description: "...",
				Parameters:  []byte(`{"type":"object"}`),
			}},
			ToolChoice: "required",
		}, false)
		assert.Equal(t, "required", body["tool_choice"])
	})

	t.Run("absent when no tools", func(t *testing.T) {
		body := p.buildRequestBody(ChatRequest{
			Messages: []Message{{Role: RoleUser, Content: "hi"}},
		}, false)
		_, present := body["tool_choice"]
		assert.False(t, present, "tool_choice must not appear when tools list is empty")
	})
}
