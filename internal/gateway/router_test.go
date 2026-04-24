package gateway

import (
	"context"
	"testing"
	"time"
)

func TestNewRouter(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	r := NewRouter(sm)
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
}

func TestRouter_Route_CreatesSession(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	r := NewRouter(sm)

	var received *GatewaySession
	r.SetHandler(func(ctx context.Context, sess *GatewaySession, msg IncomingMessage) {
		received = sess
	})

	msg := IncomingMessage{
		Platform:  "telegram",
		ChatID:    "chat1",
		UserID:    "user1",
		Username:  "alice",
		Text:      "hello",
		Timestamp: time.Now(),
	}

	r.Route(context.Background(), msg)

	if received == nil {
		t.Fatal("handler was not called")
	}
	if received.Platform != "telegram" {
		t.Errorf("session Platform = %q, want %q", received.Platform, "telegram")
	}
	if received.UserID != "user1" {
		t.Errorf("session UserID = %q, want %q", received.UserID, "user1")
	}
}

func TestRouter_Route_SameSessionForSameUser(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	r := NewRouter(sm)

	var sessions []*GatewaySession
	r.SetHandler(func(ctx context.Context, sess *GatewaySession, msg IncomingMessage) {
		sessions = append(sessions, sess)
	})

	msg := IncomingMessage{
		Platform:  "telegram",
		ChatID:    "chat1",
		UserID:    "user1",
		Text:      "hello",
		Timestamp: time.Now(),
	}

	r.Route(context.Background(), msg)
	r.Route(context.Background(), msg)

	if len(sessions) != 2 {
		t.Fatalf("expected 2 handler calls, got %d", len(sessions))
	}
	if sessions[0].ID != sessions[1].ID {
		t.Error("expected same session for same platform+user")
	}
}

func TestRouter_Route_NoHandler(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	r := NewRouter(sm)

	// Should not panic when no handler is set
	msg := IncomingMessage{
		Platform:  "telegram",
		ChatID:    "chat1",
		UserID:    "user1",
		Text:      "hello",
		Timestamp: time.Now(),
	}

	r.Route(context.Background(), msg)
}
