package agent

import (
	"context"
	"sync"

	"github.com/sadewadee/hera/internal/llm"
)

// EventType represents the type of agent lifecycle event.
type EventType string

const (
	EventBeforeRequest  EventType = "before_request"
	EventAfterResponse  EventType = "after_response"
	EventBeforeToolCall EventType = "before_tool_call"
	EventAfterToolCall  EventType = "after_tool_call"
	EventError          EventType = "error"
	EventSessionStart   EventType = "session_start"
	EventSessionEnd     EventType = "session_end"
	EventStreamToken    EventType = "stream_token"
)

// Event carries data about an agent lifecycle event.
type Event struct {
	Type      EventType      `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	Message   *llm.Message   `json:"message,omitempty"`
	ToolName  string         `json:"tool_name,omitempty"`
	Error     error          `json:"-"`
	Data      map[string]any `json:"data,omitempty"`
}

// CallbackFunc is a function invoked on agent events.
type CallbackFunc func(ctx context.Context, event Event)

// CallbackManager manages agent lifecycle callbacks.
type CallbackManager struct {
	mu        sync.RWMutex
	callbacks map[EventType][]CallbackFunc
}

// NewCallbackManager creates a new callback manager.
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		callbacks: make(map[EventType][]CallbackFunc),
	}
}

// On registers a callback for a specific event type.
func (cm *CallbackManager) On(eventType EventType, fn CallbackFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks[eventType] = append(cm.callbacks[eventType], fn)
}

// OnAny registers a callback for all event types.
func (cm *CallbackManager) OnAny(fn CallbackFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, et := range []EventType{
		EventBeforeRequest, EventAfterResponse, EventBeforeToolCall,
		EventAfterToolCall, EventError, EventSessionStart, EventSessionEnd,
		EventStreamToken,
	} {
		cm.callbacks[et] = append(cm.callbacks[et], fn)
	}
}

// Emit fires all callbacks registered for the given event type.
func (cm *CallbackManager) Emit(ctx context.Context, event Event) {
	cm.mu.RLock()
	handlers := cm.callbacks[event.Type]
	cm.mu.RUnlock()

	for _, fn := range handlers {
		fn(ctx, event)
	}
}

// Clear removes all registered callbacks.
func (cm *CallbackManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks = make(map[EventType][]CallbackFunc)
}

// Count returns the number of callbacks registered for an event type.
func (cm *CallbackManager) Count(eventType EventType) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.callbacks[eventType])
}
