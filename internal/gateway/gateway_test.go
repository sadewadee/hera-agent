package gateway

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockAdapter implements PlatformAdapter for testing.
type mockAdapter struct {
	mu        sync.Mutex
	name      string
	connected bool
	handler   MessageHandler
	sendErr   error
	connectFn func(ctx context.Context) error
}

func newMockAdapter(name string) *mockAdapter {
	return &mockAdapter{name: name}
}

func (m *mockAdapter) Name() string { return m.name }

func (m *mockAdapter) Connect(ctx context.Context) error {
	if m.connectFn != nil {
		return m.connectFn(ctx)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockAdapter) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockAdapter) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockAdapter) Send(ctx context.Context, chatID string, msg OutgoingMessage) error {
	return m.sendErr
}

func (m *mockAdapter) OnMessage(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

func (m *mockAdapter) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
	return &ChatInfo{ID: chatID, Platform: m.name, Type: "private"}, nil
}

func (m *mockAdapter) SupportedMedia() []MediaType {
	return []MediaType{MediaPhoto, MediaFile}
}

// simulateMessage triggers the handler as if a message arrived.
func (m *mockAdapter) simulateMessage(ctx context.Context, msg IncomingMessage) {
	m.mu.Lock()
	h := m.handler
	m.mu.Unlock()
	if h != nil {
		h(ctx, msg)
	}
}

func TestNewGateway(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout: 5 * time.Minute,
	})
	if gw == nil {
		t.Fatal("NewGateway returned nil")
	}
}

func TestGateway_AddAdapter(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout: 5 * time.Minute,
	})
	adapter := newMockAdapter("test")
	gw.AddAdapter(adapter)

	adapters := gw.Adapters()
	if len(adapters) != 1 {
		t.Fatalf("Adapters() length = %d, want 1", len(adapters))
	}
	if adapters[0].Name() != "test" {
		t.Errorf("adapter name = %q, want %q", adapters[0].Name(), "test")
	}
}

func TestGateway_Start_ConnectsAdapters(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout: 5 * time.Minute,
	})
	adapter := newMockAdapter("test")
	gw.AddAdapter(adapter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer gw.Stop()

	// Give goroutines a moment to run
	time.Sleep(10 * time.Millisecond)

	if !adapter.IsConnected() {
		t.Error("adapter should be connected after Start")
	}
}

func TestGateway_Stop_DisconnectsAdapters(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout: 5 * time.Minute,
	})
	adapter := newMockAdapter("test")
	gw.AddAdapter(adapter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	gw.Stop()

	if adapter.IsConnected() {
		t.Error("adapter should be disconnected after Stop")
	}
}

func TestGateway_HandleMessage_RoutesToHandler(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout: 5 * time.Minute,
	})
	adapter := newMockAdapter("telegram")
	gw.AddAdapter(adapter)

	// Authorize the test user so the auth check passes.
	gw.Pairing().AuthorizeUser("telegram", "user1")

	var received IncomingMessage
	var receivedOnce sync.Once
	done := make(chan struct{})
	gw.OnMessage(func(ctx context.Context, sess *GatewaySession, msg IncomingMessage) {
		receivedOnce.Do(func() {
			received = msg
			close(done)
		})
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer gw.Stop()

	time.Sleep(10 * time.Millisecond)

	msg := IncomingMessage{
		Platform:  "telegram",
		ChatID:    "chat1",
		UserID:    "user1",
		Username:  "alice",
		Text:      "hello",
		Timestamp: time.Now(),
	}
	adapter.simulateMessage(ctx, msg)

	select {
	case <-done:
		if received.Text != "hello" {
			t.Errorf("received Text = %q, want %q", received.Text, "hello")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message handler")
	}
}

func TestGateway_HandleMessage_UnauthorizedIsDropped(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout: 5 * time.Minute,
	})
	adapter := newMockAdapter("telegram")
	gw.AddAdapter(adapter)

	// Do NOT authorize the user — message must be silently dropped.
	handlerCalled := make(chan struct{}, 1)
	gw.OnMessage(func(ctx context.Context, sess *GatewaySession, msg IncomingMessage) {
		handlerCalled <- struct{}{}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer gw.Stop()

	time.Sleep(10 * time.Millisecond)

	msg := IncomingMessage{
		Platform:  "telegram",
		ChatID:    "chat1",
		UserID:    "unauthorized-user",
		Text:      "should be dropped",
		Timestamp: time.Now(),
	}
	adapter.simulateMessage(ctx, msg)

	select {
	case <-handlerCalled:
		t.Fatal("handler was called for unauthorized user — auth bypass not fixed")
	case <-time.After(200 * time.Millisecond):
		// Correct: handler was not called.
	}
}

func TestGateway_ReconnectBackoff(t *testing.T) {
	bo := newBackoff(100*time.Millisecond, 1*time.Second, 3)

	d1 := bo.Next()
	if d1 != 100*time.Millisecond {
		t.Errorf("first backoff = %v, want %v", d1, 100*time.Millisecond)
	}

	d2 := bo.Next()
	if d2 != 200*time.Millisecond {
		t.Errorf("second backoff = %v, want %v", d2, 200*time.Millisecond)
	}

	d3 := bo.Next()
	if d3 != 400*time.Millisecond {
		t.Errorf("third backoff = %v, want %v", d3, 400*time.Millisecond)
	}

	// Exceed max retries
	if !bo.Exhausted() {
		t.Error("backoff should be exhausted after max retries")
	}

	bo.Reset()
	if bo.Exhausted() {
		t.Error("backoff should not be exhausted after Reset")
	}
}

func TestGateway_ReconnectBackoff_CapsAtMax(t *testing.T) {
	bo := newBackoff(1*time.Second, 3*time.Second, 10)

	// Advance a few steps
	bo.Next()      // 1s
	bo.Next()      // 2s
	d := bo.Next() // 4s -> capped at 3s

	if d != 3*time.Second {
		t.Errorf("capped backoff = %v, want %v", d, 3*time.Second)
	}
}

func TestGateway_Start_FailingAdapter(t *testing.T) {
	gw := NewGateway(GatewayOptions{
		SessionTimeout:    5 * time.Minute,
		ReconnectBaseWait: 10 * time.Millisecond,
		ReconnectMaxWait:  50 * time.Millisecond,
		MaxReconnects:     10,
	})

	var attempts atomic.Int32
	adapter := newMockAdapter("failing")
	adapter.connectFn = func(ctx context.Context) error {
		attempts.Add(1)
		return errors.New("connection refused")
	}
	gw.AddAdapter(adapter)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = gw.Start(ctx)

	// Give reconnection goroutines time to attempt several retries
	time.Sleep(300 * time.Millisecond)

	gw.Stop()

	if attempts.Load() < 2 {
		t.Errorf("expected at least 2 connection attempts, got %d", attempts.Load())
	}
}
