package mcp

import (
	"testing"
	"time"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	eb := NewEventBus()
	ch := eb.Subscribe()
	defer eb.Unsubscribe(ch)

	want := Event{Type: "message", Data: map[string]any{"text": "hello"}}
	eb.Publish(want)

	select {
	case got := <-ch:
		if got.Type != want.Type {
			t.Errorf("event type = %q, want %q", got.Type, want.Type)
		}
		if got.Data["text"] != "hello" {
			t.Errorf("event data[text] = %v, want %q", got.Data["text"], "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventBus_MultipleListeners(t *testing.T) {
	eb := NewEventBus()
	ch1 := eb.Subscribe()
	ch2 := eb.Subscribe()
	defer eb.Unsubscribe(ch1)
	defer eb.Unsubscribe(ch2)

	want := Event{Type: "session_created", Data: map[string]any{"id": "s1"}}
	eb.Publish(want)

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != want.Type {
				t.Errorf("listener %d: event type = %q, want %q", i, got.Type, want.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("listener %d: timed out waiting for event", i)
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus()
	ch := eb.Subscribe()
	eb.Unsubscribe(ch)

	eb.Publish(Event{Type: "session_expired"})

	select {
	case evt, ok := <-ch:
		if ok {
			t.Errorf("received event after unsubscribe: %+v", evt)
		}
	case <-time.After(100 * time.Millisecond):
		// Expected: no event received after unsubscribe.
	}
}

func TestEventBus_NonBlocking(t *testing.T) {
	eb := NewEventBus()

	// Publish with no subscribers should not block.
	done := make(chan struct{})
	go func() {
		eb.Publish(Event{Type: "message"})
		close(done)
	}()

	select {
	case <-done:
		// Expected: Publish returned without blocking.
	case <-time.After(time.Second):
		t.Fatal("Publish blocked with no subscribers")
	}
}
