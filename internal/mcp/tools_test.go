package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/llm"
)

// newTestSessionManager creates a SessionManager with test sessions pre-populated.
func newTestSessionManager(t *testing.T) *agent.SessionManager {
	t.Helper()
	sm := agent.NewSessionManager(30 * time.Minute)
	return sm
}

func TestConversationsList_Empty(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	sm := newTestSessionManager(t)
	deps := Deps{Sessions: sm}
	registerConversationsList(srv, deps)

	callParams, _ := json.Marshal(map[string]any{
		"name":      "conversations_list",
		"arguments": map[string]any{},
	})
	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  callParams,
	})

	if resp.Error != nil {
		t.Fatalf("conversations_list returned error: %+v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content array, got %T", result["content"])
	}

	// Parse the text content to verify the response structure.
	textEntry, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not a map: %T", content[0])
	}
	textStr, ok := textEntry["text"].(string)
	if !ok {
		t.Fatalf("text is not a string: %T", textEntry["text"])
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(textStr), &parsed); err != nil {
		t.Fatalf("unmarshal text content: %v", err)
	}

	count, ok := parsed["count"].(float64)
	if !ok {
		t.Fatalf("count is not a number: %T", parsed["count"])
	}
	if count != 0 {
		t.Errorf("count = %v, want 0", count)
	}
}

func TestConversationsList_WithSessions(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	sm := newTestSessionManager(t)
	sm.Create("telegram", "user1")
	sm.Create("discord", "user2")
	deps := Deps{Sessions: sm}
	registerConversationsList(srv, deps)

	callParams, _ := json.Marshal(map[string]any{
		"name":      "conversations_list",
		"arguments": map[string]any{},
	})
	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/call",
		Params:  callParams,
	})

	if resp.Error != nil {
		t.Fatalf("conversations_list returned error: %+v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	content := result["content"].([]any)
	textEntry := content[0].(map[string]any)
	textStr := textEntry["text"].(string)

	var parsed map[string]any
	json.Unmarshal([]byte(textStr), &parsed)

	count := parsed["count"].(float64)
	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestMessagesSend(t *testing.T) {
	// We cannot easily create a real Agent without full deps, so we test
	// that the handler returns an error when Agent is nil (expected behavior).
	srv := NewServer("test", "1.0.0")
	sm := newTestSessionManager(t)
	deps := Deps{Sessions: sm, Agent: nil}
	registerMessagesSend(srv, deps)

	callParams, _ := json.Marshal(map[string]any{
		"name":      "messages_send",
		"arguments": map[string]string{"text": "hello"},
	})
	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params:  callParams,
	})

	if resp.Error != nil {
		t.Fatalf("tools/call returned JSON-RPC error: %+v", resp.Error)
	}

	// The tool handler should return an isError result since agent is nil.
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("expected isError=true when agent is nil")
	}
}

func TestChannelsList(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	sm := newTestSessionManager(t)

	// Create sessions on different platforms.
	s1 := sm.Create("telegram", "user1")
	s1.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "hi"})
	sm.Create("telegram", "user2")
	sm.Create("discord", "user3")

	deps := Deps{Sessions: sm}
	registerChannelsList(srv, deps)

	callParams, _ := json.Marshal(map[string]any{
		"name":      "channels_list",
		"arguments": map[string]any{},
	})
	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "tools/call",
		Params:  callParams,
	})

	if resp.Error != nil {
		t.Fatalf("channels_list returned error: %+v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	content := result["content"].([]any)
	textEntry := content[0].(map[string]any)
	textStr := textEntry["text"].(string)

	var parsed map[string]any
	json.Unmarshal([]byte(textStr), &parsed)

	channels, ok := parsed["channels"].([]any)
	if !ok {
		t.Fatalf("channels is not a slice: %T", parsed["channels"])
	}
	if len(channels) < 2 {
		t.Errorf("expected at least 2 channels (telegram, discord), got %d", len(channels))
	}
}

func TestEventsWait_ReceivesEvent(t *testing.T) {
	eb := NewEventBus()
	sm := newTestSessionManager(t)
	deps := Deps{Sessions: sm, EventBus: eb}

	// Call the handler directly (not through the server stdin pipe) so we
	// can control timing.  Extract the handler via a small wrapper server.
	srv := NewServer("test", "1.0.0")
	registerEventsWait(srv, deps)

	srv.mu.RLock()
	handler := srv.tools["events_wait"]
	srv.mu.RUnlock()

	if handler == nil {
		t.Fatal("events_wait handler not registered")
	}

	params, _ := json.Marshal(map[string]any{"timeout_ms": 5000})

	// Publish an event shortly after the handler starts waiting.
	go func() {
		time.Sleep(50 * time.Millisecond)
		eb.Publish(Event{Type: "message", Data: map[string]any{"text": "hi"}})
	}()

	ctx := context.Background()
	result, err := handler(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}
	if m["event"] != "message" {
		t.Errorf("event = %v, want %q", m["event"], "message")
	}
	data, ok := m["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is not a map: %T", m["data"])
	}
	if data["text"] != "hi" {
		t.Errorf("data[text] = %v, want %q", data["text"], "hi")
	}
}

func TestEventsWait_Timeout(t *testing.T) {
	eb := NewEventBus()
	sm := newTestSessionManager(t)
	deps := Deps{Sessions: sm, EventBus: eb}

	srv := NewServer("test", "1.0.0")
	registerEventsWait(srv, deps)

	srv.mu.RLock()
	handler := srv.tools["events_wait"]
	srv.mu.RUnlock()

	// Use a very short timeout so the test completes fast.
	params, _ := json.Marshal(map[string]any{"timeout_ms": 50})

	ctx := context.Background()
	start := time.Now()
	result, err := handler(ctx, params)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}
	if m["event"] != "timeout" {
		t.Errorf("event = %v, want %q", m["event"], "timeout")
	}

	// Verify it actually waited (at least 40ms to account for scheduling jitter).
	if elapsed < 40*time.Millisecond {
		t.Errorf("handler returned too fast: %v (expected ~50ms)", elapsed)
	}
}

func TestEventsWait_NilEventBus(t *testing.T) {
	sm := newTestSessionManager(t)
	deps := Deps{Sessions: sm, EventBus: nil}

	srv := NewServer("test", "1.0.0")
	registerEventsWait(srv, deps)

	srv.mu.RLock()
	handler := srv.tools["events_wait"]
	srv.mu.RUnlock()

	params, _ := json.Marshal(map[string]any{"timeout_ms": 1000})

	ctx := context.Background()
	result, err := handler(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}
	if m["event"] != "timeout" {
		t.Errorf("event = %v, want %q", m["event"], "timeout")
	}
}

func TestEventsWait_ContextCancel(t *testing.T) {
	eb := NewEventBus()
	sm := newTestSessionManager(t)
	deps := Deps{Sessions: sm, EventBus: eb}

	srv := NewServer("test", "1.0.0")
	registerEventsWait(srv, deps)

	srv.mu.RLock()
	handler := srv.tools["events_wait"]
	srv.mu.RUnlock()

	params, _ := json.Marshal(map[string]any{"timeout_ms": 30000})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context shortly after the handler starts.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := handler(ctx, params)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
