// Package acp provides the Agent Client Protocol implementation.
//
// events.go implements callback factories for bridging agent events to ACP
// session updates. Each factory returns a callable that the agent uses for
// its event callbacks. Internally, the callbacks push ACP session updates
// via the event bus.
package acp

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// ToolCallEvent represents a tool call start/complete event for ACP.
type ToolCallEvent struct {
	ID        string         `json:"id"`
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Result    string         `json:"result,omitempty"`
	Status    string         `json:"status"` // "started", "completed"
}

// ThinkingEvent represents an agent thinking/reasoning event.
type ThinkingEvent struct {
	Text string `json:"text"`
}

// MessageEvent represents an agent message stream event.
type MessageEvent struct {
	Text string `json:"text"`
}

// StepEvent represents an agent step completion event.
type StepEvent struct {
	APICallCount int              `json:"api_call_count"`
	PrevTools    []ToolStepResult `json:"prev_tools,omitempty"`
}

// ToolStepResult holds the result of a tool invocation in a step.
type ToolStepResult struct {
	Name   string `json:"name"`
	Result string `json:"result,omitempty"`
}

// ToolCallIDTracker tracks in-flight tool call IDs per tool name using
// a FIFO queue so parallel same-name calls complete against the correct
// ACP tool call.
type ToolCallIDTracker struct {
	mu    sync.Mutex
	calls map[string][]string // tool_name -> queue of IDs
}

// NewToolCallIDTracker creates a new tracker.
func NewToolCallIDTracker() *ToolCallIDTracker {
	return &ToolCallIDTracker{
		calls: make(map[string][]string),
	}
}

// Push adds a tool call ID for the given tool name.
func (t *ToolCallIDTracker) Push(toolName, id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls[toolName] = append(t.calls[toolName], id)
}

// Pop removes and returns the oldest tool call ID for the given tool name.
// Returns empty string if no IDs are queued.
func (t *ToolCallIDTracker) Pop(toolName string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	queue := t.calls[toolName]
	if len(queue) == 0 {
		return ""
	}
	id := queue[0]
	t.calls[toolName] = queue[1:]
	if len(t.calls[toolName]) == 0 {
		delete(t.calls, toolName)
	}
	return id
}

// MakeToolProgressCallback creates a tool progress callback that emits
// ToolCallStart events. Event types other than "tool.started" are ignored.
func MakeToolProgressCallback(
	sessionID string,
	tracker *ToolCallIDTracker,
	emitFn func(sessionID string, event []byte),
) func(eventType, name, preview string, args map[string]any) {
	return func(eventType, name, preview string, args map[string]any) {
		if eventType != "tool.started" {
			return
		}

		id := generateToolCallID()
		tracker.Push(name, id)

		event := ToolCallEvent{
			ID:        id,
			ToolName:  name,
			Arguments: args,
			Status:    "started",
		}

		data, err := json.Marshal(event)
		if err != nil {
			slog.Debug("failed to marshal tool start event", "error", err)
			return
		}
		emitFn(sessionID, data)
	}
}

// MakeThinkingCallback creates a callback that emits agent thinking text.
func MakeThinkingCallback(
	sessionID string,
	emitFn func(sessionID string, event []byte),
) func(text string) {
	return func(text string) {
		if text == "" {
			return
		}
		event := ThinkingEvent{Text: text}
		data, err := json.Marshal(event)
		if err != nil {
			slog.Debug("failed to marshal thinking event", "error", err)
			return
		}
		emitFn(sessionID, data)
	}
}

// MakeStepCallback creates a step callback that emits ToolCallComplete
// events for completed tools.
func MakeStepCallback(
	sessionID string,
	tracker *ToolCallIDTracker,
	emitFn func(sessionID string, event []byte),
) func(apiCallCount int, prevTools []ToolStepResult) {
	return func(apiCallCount int, prevTools []ToolStepResult) {
		for _, tool := range prevTools {
			id := tracker.Pop(tool.Name)
			if id == "" {
				continue
			}

			event := ToolCallEvent{
				ID:       id,
				ToolName: tool.Name,
				Result:   tool.Result,
				Status:   "completed",
			}
			data, err := json.Marshal(event)
			if err != nil {
				slog.Debug("failed to marshal tool complete event", "error", err)
				continue
			}
			emitFn(sessionID, data)
		}
	}
}

// MakeMessageCallback creates a callback that streams agent response text.
func MakeMessageCallback(
	sessionID string,
	emitFn func(sessionID string, event []byte),
) func(text string) {
	return func(text string) {
		if text == "" {
			return
		}
		event := MessageEvent{Text: text}
		data, err := json.Marshal(event)
		if err != nil {
			slog.Debug("failed to marshal message event", "error", err)
			return
		}
		emitFn(sessionID, data)
	}
}

// generateToolCallID creates a unique tool call ID.
func generateToolCallID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("tc-%x", b)
}
