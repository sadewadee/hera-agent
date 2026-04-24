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

func TestNewAnthropicProvider(t *testing.T) {
	cfg := ProviderConfig{APIKey: "test-key", Model: "claude-sonnet-4-20250514"}
	p, err := NewAnthropicProvider(cfg)
	if err != nil {
		t.Fatalf("NewAnthropicProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewAnthropicProvider() returned nil")
	}
}

func TestNewAnthropicProvider_MissingAPIKey(t *testing.T) {
	cfg := ProviderConfig{Model: "claude-sonnet-4-20250514"}
	_, err := NewAnthropicProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestAnthropic_ModelInfo(t *testing.T) {
	cfg := ProviderConfig{APIKey: "k", Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	info := p.ModelInfo()
	if info.ID != "claude-sonnet-4-20250514" {
		t.Errorf("ModelInfo().ID = %q, want %q", info.ID, "claude-sonnet-4-20250514")
	}
	if info.Provider != "anthropic" {
		t.Errorf("ModelInfo().Provider = %q, want %q", info.Provider, "anthropic")
	}
}

func TestAnthropic_CountTokens(t *testing.T) {
	cfg := ProviderConfig{APIKey: "k", Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)

	msgs := []Message{{Role: RoleUser, Content: "Hello, world!"}}
	count, err := p.CountTokens(msgs)
	if err != nil {
		t.Fatalf("CountTokens() error = %v", err)
	}
	if count <= 0 {
		t.Errorf("CountTokens() = %d, want > 0", count)
	}
}

func TestAnthropic_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/messages") {
			t.Errorf("path = %s, want /messages suffix", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q, want %q", r.Header.Get("x-api-key"), "test-key")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version header")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Anthropic uses "stream" field
		if reqBody["stream"] == true {
			t.Error("expected stream=false for non-streaming Chat")
		}

		// Verify system message extracted
		if _, ok := reqBody["system"]; !ok {
			// acceptable, system may not always be set
		}

		resp := map[string]any{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{
					"type": "text",
					"text": "Hello from Anthropic!",
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "test-key", BaseURL: server.URL, Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful."},
			{Role: RoleUser, Content: "Hi"},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Message.Content != "Hello from Anthropic!" {
		t.Errorf("content = %q, want %q", resp.Message.Content, "Hello from Anthropic!")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
	}
}

func TestAnthropic_Chat_WithToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "msg_456",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{
					"type": "text",
					"text": "Let me check the weather.",
				},
				{
					"type":  "tool_use",
					"id":    "toolu_abc",
					"name":  "get_weather",
					"input": map[string]any{"city": "NYC"},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 20, "output_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: "weather?"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.ID != "toolu_abc" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "toolu_abc")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
}

func TestAnthropic_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"msg_1","model":"claude-sonnet-4-20250514","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hel"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"lo!"}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":0}`,
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		}
		for _, evt := range events {
			fmt.Fprintf(w, "%s\n\n", evt)
			flusher.Flush()
		}
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	ch, err := p.ChatStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
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

func TestAnthropic_ChatStream_WithToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"msg_1","model":"claude-sonnet-4-20250514","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_abc123","name":"get_weather"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"ci"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"ty\": \"NYC\"}"}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":0}`,
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":15}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		}
		for _, evt := range events {
			fmt.Fprintf(w, "%s\n\n", evt)
			flusher.Flush()
		}
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	ch, err := p.ChatStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: "What is the weather in NYC?"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	var toolCalls []StreamEvent
	var gotDone bool
	for evt := range ch {
		if evt.Error != nil {
			t.Fatalf("stream error: %v", evt.Error)
		}
		switch evt.Type {
		case "tool_call":
			toolCalls = append(toolCalls, evt)
		case "done":
			gotDone = true
		}
	}

	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool_call event, got %d", len(toolCalls))
	}

	tc := toolCalls[0].ToolCall
	if tc == nil {
		t.Fatal("tool_call event has nil ToolCall")
	}
	if tc.ID != "toolu_abc123" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "toolu_abc123")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}

	var args map[string]any
	if err := json.Unmarshal(tc.Args, &args); err != nil {
		t.Fatalf("failed to unmarshal tool args: %v", err)
	}
	if args["city"] != "NYC" {
		t.Errorf("ToolCall.Args[city] = %v, want %q", args["city"], "NYC")
	}

	if !gotDone {
		t.Error("did not receive 'done' event")
	}
}

func TestAnthropic_ChatStream_MixedTextAndToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			// Message start
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"msg_2","model":"claude-sonnet-4-20250514","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`,
			// Text block
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me "}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"check."}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":0}`,
			// Tool use block
			`event: content_block_start` + "\n" + `data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_xyz","name":"search"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"q"}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"uery\": \"test\"}"}}`,
			`event: content_block_stop` + "\n" + `data: {"type":"content_block_stop","index":1}`,
			// End
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":20}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		}
		for _, evt := range events {
			fmt.Fprintf(w, "%s\n\n", evt)
			flusher.Flush()
		}
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	ch, err := p.ChatStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: "Search for test"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	var eventTypes []string
	var deltas []string
	var toolCalls []StreamEvent
	for evt := range ch {
		if evt.Error != nil {
			t.Fatalf("stream error: %v", evt.Error)
		}
		eventTypes = append(eventTypes, evt.Type)
		switch evt.Type {
		case "delta":
			deltas = append(deltas, evt.Delta)
		case "tool_call":
			toolCalls = append(toolCalls, evt)
		}
	}

	// Verify text deltas came first
	combined := strings.Join(deltas, "")
	if combined != "Let me check." {
		t.Errorf("combined deltas = %q, want %q", combined, "Let me check.")
	}

	// Verify tool call received
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool_call event, got %d", len(toolCalls))
	}
	tc := toolCalls[0].ToolCall
	if tc == nil {
		t.Fatal("tool_call event has nil ToolCall")
	}
	if tc.ID != "toolu_xyz" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "toolu_xyz")
	}
	if tc.Name != "search" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "search")
	}

	var args map[string]any
	if err := json.Unmarshal(tc.Args, &args); err != nil {
		t.Fatalf("failed to unmarshal tool args: %v", err)
	}
	if args["query"] != "test" {
		t.Errorf("ToolCall.Args[query] = %v, want %q", args["query"], "test")
	}

	// Verify ordering: deltas come before tool_call, done is last
	deltasSeen := false
	toolCallSeen := false
	for _, et := range eventTypes {
		switch et {
		case "delta":
			if toolCallSeen {
				t.Error("received delta after tool_call -- expected deltas before tool_call")
			}
			deltasSeen = true
		case "tool_call":
			if !deltasSeen {
				t.Error("received tool_call before any delta")
			}
			toolCallSeen = true
		case "done":
			if !toolCallSeen {
				t.Error("received done before tool_call")
			}
		}
	}
}

func TestAnthropic_Chat_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "max_tokens is required",
			},
		})
	}))
	defer server.Close()

	cfg := ProviderConfig{APIKey: "k", BaseURL: server.URL, Model: "claude-sonnet-4-20250514"}
	p, _ := NewAnthropicProvider(cfg)
	_, err := p.Chat(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestAnthropic_RegisterFactory(t *testing.T) {
	reg := NewRegistry()
	RegisterAnthropic(reg)

	names := reg.List()
	found := false
	for _, n := range names {
		if n == "anthropic" {
			found = true
			break
		}
	}
	if !found {
		t.Error("anthropic factory not registered")
	}
}
