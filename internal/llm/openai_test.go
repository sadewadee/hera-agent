package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	cfg := ProviderConfig{
		APIKey: "test-key",
		Model:  "gpt-4o",
	}
	p, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewOpenAIProvider() returned nil")
	}
}

func TestNewOpenAIProvider_MissingAPIKey(t *testing.T) {
	cfg := ProviderConfig{
		Model: "gpt-4o",
	}
	_, err := NewOpenAIProvider(cfg)
	if err == nil {
		t.Fatal("NewOpenAIProvider() expected error for missing API key")
	}
}

func TestOpenAI_ModelInfo(t *testing.T) {
	cfg := ProviderConfig{APIKey: "k", Model: "gpt-4o"}
	p, _ := NewOpenAIProvider(cfg)
	info := p.ModelInfo()
	if info.ID != "gpt-4o" {
		t.Errorf("ModelInfo().ID = %q, want %q", info.ID, "gpt-4o")
	}
	if info.Provider != "openai" {
		t.Errorf("ModelInfo().Provider = %q, want %q", info.Provider, "openai")
	}
}

func TestOpenAI_CountTokens(t *testing.T) {
	cfg := ProviderConfig{APIKey: "k", Model: "gpt-4o"}
	p, _ := NewOpenAIProvider(cfg)

	msgs := []Message{
		{Role: RoleUser, Content: "Hello, world!"},
	}
	count, err := p.CountTokens(msgs)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}
	if count <= 0 {
		t.Errorf("CountTokens() = %d, want > 0", count)
	}
}

func TestOpenAI_Chat(t *testing.T) {
	// Mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("path = %s, want /chat/completions suffix", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-key")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		if reqBody["stream"] != false {
			t.Error("expected stream=false for non-streaming Chat")
		}

		resp := map[string]any{
			"id":    "chatcmpl-123",
			"model": "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello from OpenAI!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4o",
	}
	p, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Message.Content != "Hello from OpenAI!" {
		t.Errorf("Chat().Message.Content = %q, want %q", resp.Message.Content, "Hello from OpenAI!")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Chat().Usage.TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("Chat().FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

func TestOpenAI_Chat_WithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "chatcmpl-456",
			"model": "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_abc",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"city":"NYC"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     20,
				"completion_tokens": 10,
				"total_tokens":      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "gpt-4o"}
	p, _ := NewOpenAIProvider(cfg)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "weather?"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
}

func TestOpenAI_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		if reqBody["stream"] != true {
			t.Error("expected stream=true for streaming ChatStream")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"role":"assistant","content":"Hel"},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"lo!"},"index":0}]}`,
			`data: {"id":"chatcmpl-1","choices":[{"delta":{},"index":0,"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "gpt-4o"}
	p, _ := NewOpenAIProvider(cfg)

	ch, err := p.ChatStream(context.Background(), ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	var deltas []string
	var gotDone bool
	for evt := range ch {
		if evt.Error != nil {
			t.Fatalf("stream error: %v", evt.Error)
		}
		switch evt.Type {
		case "delta":
			deltas = append(deltas, evt.Delta)
		case "done":
			gotDone = true
		}
	}

	combined := strings.Join(deltas, "")
	if combined != "Hello!" {
		t.Errorf("combined deltas = %q, want %q", combined, "Hello!")
	}
	if !gotDone {
		t.Error("did not receive 'done' event")
	}
}

func TestOpenAI_Chat_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "gpt-4o"}
	p, _ := NewOpenAIProvider(cfg)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("Chat() expected error for 429 response")
	}
}

func TestOpenAI_BaseURLOverride(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		resp := map[string]any{
			"id": "chatcmpl-1", "model": "gpt-4o",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "ok"},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "gpt-4o"}
	p, _ := NewOpenAIProvider(cfg)
	p.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if !called {
		t.Error("custom base URL server was not called")
	}
}

func TestOpenAI_RegisterFactory(t *testing.T) {
	reg := NewRegistry()
	RegisterOpenAI(reg)

	names := reg.List()
	found := false
	for _, n := range names {
		if n == "openai" {
			found = true
			break
		}
	}
	if !found {
		t.Error("openai factory not registered")
	}
}
