package agent

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestCallbackManager_OnAndEmit(t *testing.T) {
	cm := NewCallbackManager()
	var called atomic.Int32

	cm.On(EventBeforeRequest, func(ctx context.Context, event Event) {
		called.Add(1)
	})

	cm.Emit(context.Background(), Event{Type: EventBeforeRequest})
	if called.Load() != 1 {
		t.Errorf("callback called %d times, want 1", called.Load())
	}
}

func TestCallbackManager_MultipleCallbacks(t *testing.T) {
	cm := NewCallbackManager()
	var count atomic.Int32

	cm.On(EventAfterResponse, func(ctx context.Context, event Event) { count.Add(1) })
	cm.On(EventAfterResponse, func(ctx context.Context, event Event) { count.Add(1) })

	cm.Emit(context.Background(), Event{Type: EventAfterResponse})
	if count.Load() != 2 {
		t.Errorf("callbacks called %d times, want 2", count.Load())
	}
}

func TestCallbackManager_OnAny(t *testing.T) {
	cm := NewCallbackManager()
	var count atomic.Int32

	cm.OnAny(func(ctx context.Context, event Event) { count.Add(1) })

	cm.Emit(context.Background(), Event{Type: EventBeforeRequest})
	cm.Emit(context.Background(), Event{Type: EventAfterResponse})
	cm.Emit(context.Background(), Event{Type: EventError})

	if count.Load() != 3 {
		t.Errorf("callbacks called %d times, want 3", count.Load())
	}
}

func TestCallbackManager_DifferentTypes(t *testing.T) {
	cm := NewCallbackManager()
	var beforeCount, afterCount atomic.Int32

	cm.On(EventBeforeRequest, func(ctx context.Context, event Event) { beforeCount.Add(1) })
	cm.On(EventAfterResponse, func(ctx context.Context, event Event) { afterCount.Add(1) })

	cm.Emit(context.Background(), Event{Type: EventBeforeRequest})
	cm.Emit(context.Background(), Event{Type: EventBeforeRequest})
	cm.Emit(context.Background(), Event{Type: EventAfterResponse})

	if beforeCount.Load() != 2 {
		t.Errorf("before callbacks: %d, want 2", beforeCount.Load())
	}
	if afterCount.Load() != 1 {
		t.Errorf("after callbacks: %d, want 1", afterCount.Load())
	}
}

func TestCallbackManager_Clear(t *testing.T) {
	cm := NewCallbackManager()
	var called atomic.Int32

	cm.On(EventError, func(ctx context.Context, event Event) { called.Add(1) })
	cm.Clear()
	cm.Emit(context.Background(), Event{Type: EventError})

	if called.Load() != 0 {
		t.Error("callbacks should not fire after Clear()")
	}
}

func TestCallbackManager_Count(t *testing.T) {
	cm := NewCallbackManager()
	cm.On(EventStreamToken, func(ctx context.Context, event Event) {})
	cm.On(EventStreamToken, func(ctx context.Context, event Event) {})

	if cm.Count(EventStreamToken) != 2 {
		t.Errorf("Count = %d, want 2", cm.Count(EventStreamToken))
	}
	if cm.Count(EventError) != 0 {
		t.Errorf("Count for unregistered type = %d, want 0", cm.Count(EventError))
	}
}
