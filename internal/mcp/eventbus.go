package mcp

import "sync"

// Event represents a notification dispatched through the EventBus.
type Event struct {
	Type string         `json:"type"` // "message", "session_created", "session_expired"
	Data map[string]any `json:"data,omitempty"`
}

// EventBus is a simple pub/sub mechanism for delivering events to waiting
// MCP tool handlers (e.g. events_wait).  It is safe for concurrent use.
type EventBus struct {
	mu        sync.Mutex
	listeners []chan Event
}

// NewEventBus creates a new EventBus with no listeners.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Subscribe returns a buffered channel that will receive future events.
// The caller must call Unsubscribe when done to avoid resource leaks.
func (eb *EventBus) Subscribe() <-chan Event {
	ch := make(chan Event, 16)
	eb.mu.Lock()
	eb.listeners = append(eb.listeners, ch)
	eb.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the listener list and closes it.
func (eb *EventBus) Unsubscribe(ch <-chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for i, l := range eb.listeners {
		if l == ch {
			eb.listeners = append(eb.listeners[:i], eb.listeners[i+1:]...)
			close(l)
			return
		}
	}
}

// Publish sends an event to all current listeners.  If a listener's buffer
// is full the event is dropped for that listener (non-blocking send).
func (eb *EventBus) Publish(event Event) {
	eb.mu.Lock()
	// Snapshot to avoid holding the lock during sends.
	snapshot := make([]chan Event, len(eb.listeners))
	copy(snapshot, eb.listeners)
	eb.mu.Unlock()

	for _, ch := range snapshot {
		select {
		case ch <- event:
		default:
			// Listener buffer full; drop event for this listener.
		}
	}
}
