package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOllamaProvider(t *testing.T) {
	cfg := ProviderConfig{Model: "llama3"}
	p, err := NewOllamaProvider(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewOllamaProvider() returned nil")
	}
}

func TestNewOllamaProvider_DefaultBaseURL(t *testing.T) {
	cfg := ProviderConfig{Model: "llama3"}
	p, _ := NewOllamaProvider(cfg)
	op := p.(*OllamaProvider)
	if op.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %q, want %q", op.baseURL, "http://localhost:11434")
	}
}

func TestNewOllamaProvider_CustomBaseURL(t *testing.T) {
	cfg := ProviderConfig{Model: "llama3", BaseURL: "http://myhost:1234"}
	p, _ := NewOllamaProvider(cfg)
	op := p.(*OllamaProvider)
	if op.baseURL != "http://myhost:1234" {
		t.Errorf("baseURL = %q, want %q", op.baseURL, "http://myhost:1234")
	}
}

func TestOllama_ModelInfo(t *testing.T) {
	cfg := ProviderConfig{Model: "llama3"}
	p, _ := NewOllamaProvider(cfg)
	info := p.ModelInfo()
	if info.ID != "llama3" {
		t.Errorf("ModelInfo().ID = %q, want %q", info.ID, "llama3")
	}
	if info.Provider != "ollama" {
		t.Errorf("ModelInfo().Provider = %q, want %q", info.Provider, "ollama")
	}
}

func TestOllama_CountTokens(t *testing.T) {
	cfg := ProviderConfig{Model: "llama3"}
	p, _ := NewOllamaProvider(cfg)
	msgs := []Message{{Role: RoleUser, Content: "Hello, world!"}}
	count, err := p.CountTokens(msgs)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}
	if count <= 0 {
		t.Errorf("CountTokens() = %d, want > 0", count)
	}
}

func TestOllama_CountTokens_WithToolCalls(t *testing.T) {
	cfg := ProviderConfig{Model: "llama3"}
	p, _ := NewOllamaProvider(cfg)

	msgs := []Message{
		{
			Role:    RoleAssistant,
			Content: "Let me check that.",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Name: "get_weather",
					Args: json.RawMessage(`{"city":"NYC","units":"metric"}`),
				},
			},
		},
	}

	count, err := p.CountTokens(msgs)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}

	// Calculate expected:
	// per-message overhead = 4
	// "Let me check that." = 19 chars / 4 = 4
	// ToolCall name "get_weather" = 11 chars / 4 = 2
	// ToolCall args `{"city":"NYC","units":"metric"}` = 30 chars / 4 = 7
	// total = 4 + 4 + 2 + 7 = 17
	expected := 4 + len("Let me check that.")/4 + len("get_weather")/4 + len(`{"city":"NYC","units":"metric"}`)/4
	if count != expected {
		t.Errorf("CountTokens() = %d, want %d (ToolCalls should contribute to count)", count, expected)
	}

	// Also verify it is strictly greater than count without tool calls.
	msgsNoTools := []Message{
		{
			Role:    RoleAssistant,
			Content: "Let me check that.",
		},
	}
	countNoTools, _ := p.CountTokens(msgsNoTools)
	if count <= countNoTools {
		t.Errorf("CountTokens with ToolCalls (%d) should be > without (%d)", count, countNoTools)
	}
}

func TestOllama_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/chat") {
			t.Errorf("path = %s, want /api/chat suffix", r.URL.Path)
		}

		resp := map[string]any{
			"model": "llama3",
			"message": map[string]any{
				"role":    "assistant",
				"content": "Hello from Ollama!",
			},
			"done":              true,
			"eval_count":        5,
			"prompt_eval_count": 10,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{Model: "llama3", BaseURL: server.URL}
	p, _ := NewOllamaProvider(cfg)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "llama3",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Message.Content != "Hello from Ollama!" {
		t.Errorf("content = %q, want %q", resp.Message.Content, "Hello from Ollama!")
	}
}

func TestOllama_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, _ := w.(http.Flusher)

		chunks := []map[string]any{
			{"model": "llama3", "message": map[string]any{"role": "assistant", "content": "Hel"}, "done": false},
			{"model": "llama3", "message": map[string]any{"role": "assistant", "content": "lo!"}, "done": false},
			{"model": "llama3", "message": map[string]any{"role": "assistant", "content": ""}, "done": true, "eval_count": 2, "prompt_eval_count": 5},
		}
		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "%s\n", data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	cfg := ProviderConfig{Model: "llama3", BaseURL: server.URL}
	p, _ := NewOllamaProvider(cfg)
	ch, err := p.ChatStream(context.Background(), ChatRequest{
		Model:    "llama3",
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

func TestOllama_Chat_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("model not found"))
	}))
	defer server.Close()

	cfg := ProviderConfig{Model: "llama3", BaseURL: server.URL}
	p, _ := NewOllamaProvider(cfg)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "llama3",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestOllama_RegisterFactory(t *testing.T) {
	reg := NewRegistry()
	RegisterOllama(reg)

	names := reg.List()
	found := false
	for _, n := range names {
		if n == "ollama" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ollama factory not registered")
	}
}
