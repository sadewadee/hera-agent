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

func TestNewGeminiProvider(t *testing.T) {
	cfg := ProviderConfig{APIKey: "test-key", Model: "gemini-2.0-flash"}
	p, err := NewGeminiProvider(cfg)
	if err != nil {
		t.Fatalf("NewGeminiProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewGeminiProvider() returned nil")
	}
}

func TestNewGeminiProvider_MissingAPIKey(t *testing.T) {
	cfg := ProviderConfig{Model: "gemini-2.0-flash"}
	_, err := NewGeminiProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestGemini_ModelInfo(t *testing.T) {
	cfg := ProviderConfig{APIKey: "k", Model: "gemini-2.0-flash"}
	p, _ := NewGeminiProvider(cfg)
	info := p.ModelInfo()
	if info.ID != "gemini-2.0-flash" {
		t.Errorf("ModelInfo().ID = %q, want %q", info.ID, "gemini-2.0-flash")
	}
	if info.Provider != "gemini" {
		t.Errorf("ModelInfo().Provider = %q, want %q", info.Provider, "gemini")
	}
}

func TestGemini_CountTokens(t *testing.T) {
	cfg := ProviderConfig{APIKey: "k", Model: "gemini-2.0-flash"}
	p, _ := NewGeminiProvider(cfg)
	msgs := []Message{{Role: RoleUser, Content: "Hello, world!"}}
	count, err := p.CountTokens(msgs)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}
	if count <= 0 {
		t.Errorf("CountTokens() = %d, want > 0", count)
	}
}

func TestGemini_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		// Should use generateContent endpoint
		if !strings.Contains(r.URL.Path, "generateContent") {
			t.Errorf("path = %s, expected generateContent", r.URL.Path)
		}
		// API key should be in header, not URL query param (security best practice)
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Errorf("x-goog-api-key header = %q, want %q", r.Header.Get("x-goog-api-key"), "test-key")
		}
		// Verify key is NOT in URL (security: prevents key leakage in logs/proxies)
		if r.URL.Query().Get("key") != "" {
			t.Errorf("API key should not be in URL query params, got key=%q", r.URL.Query().Get("key"))
		}

		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": "Hello from Gemini!"},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     10,
				"candidatesTokenCount": 5,
				"totalTokenCount":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "test-key", BaseURL: server.URL, Model: "gemini-2.0-flash"}
	p, _ := NewGeminiProvider(cfg)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.0-flash",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Message.Content != "Hello from Gemini!" {
		t.Errorf("content = %q, want %q", resp.Message.Content, "Hello from Gemini!")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestGemini_Chat_WithFunctionCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{
								"functionCall": map[string]any{
									"name": "get_weather",
									"args": map[string]any{"city": "NYC"},
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     20,
				"candidatesTokenCount": 10,
				"totalTokenCount":      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "gemini-2.0-flash"}
	p, _ := NewGeminiProvider(cfg)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.0-flash",
		Messages: []Message{{Role: RoleUser, Content: "weather?"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
}

func TestGemini_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should use streamGenerateContent endpoint
		if !strings.Contains(r.URL.Path, "streamGenerateContent") {
			t.Errorf("path = %s, expected streamGenerateContent", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		flusher, _ := w.(http.Flusher)

		// Gemini streaming returns a JSON array
		chunks := []map[string]any{
			{
				"candidates": []map[string]any{
					{"content": map[string]any{"parts": []map[string]any{{"text": "Hel"}}, "role": "model"}},
				},
			},
			{
				"candidates": []map[string]any{
					{"content": map[string]any{"parts": []map[string]any{{"text": "lo!"}}, "role": "model"}},
				},
			},
			{
				"candidates": []map[string]any{
					{
						"content":      map[string]any{"parts": []map[string]any{{"text": ""}}, "role": "model"},
						"finishReason": "STOP",
					},
				},
				"usageMetadata": map[string]any{
					"promptTokenCount":     5,
					"candidatesTokenCount": 2,
					"totalTokenCount":      7,
				},
			},
		}
		// Write as newline-delimited JSON
		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "%s\n", data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "gemini-2.0-flash"}
	p, _ := NewGeminiProvider(cfg)
	ch, err := p.ChatStream(context.Background(), ChatRequest{
		Model:    "gemini-2.0-flash",
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

func TestGemini_Chat_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    403,
				"message": "API key not valid",
				"status":  "PERMISSION_DENIED",
			},
		})
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "bad", BaseURL: server.URL, Model: "gemini-2.0-flash"}
	p, _ := NewGeminiProvider(cfg)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "gemini-2.0-flash",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGemini_RegisterFactory(t *testing.T) {
	reg := NewRegistry()
	RegisterGemini(reg)

	names := reg.List()
	found := false
	for _, n := range names {
		if n == "gemini" {
			found = true
			break
		}
	}
	if !found {
		t.Error("gemini factory not registered")
	}
}
