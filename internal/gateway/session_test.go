package gateway

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}
}

func TestSessionManager_GetOrCreate_NewSession(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	sess := sm.GetOrCreate("telegram", "user123")

	if sess.Platform != "telegram" {
		t.Errorf("Platform = %q, want %q", sess.Platform, "telegram")
	}
	if sess.UserID != "user123" {
		t.Errorf("UserID = %q, want %q", sess.UserID, "user123")
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if sess.LastActivity.IsZero() {
		t.Error("LastActivity should not be zero")
	}
}

func TestSessionManager_GetOrCreate_ReturnsSameSession(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	sess1 := sm.GetOrCreate("telegram", "user123")
	sess2 := sm.GetOrCreate("telegram", "user123")

	if sess1.ID != sess2.ID {
		t.Errorf("expected same session, got IDs %q and %q", sess1.ID, sess2.ID)
	}
}

func TestSessionManager_GetOrCreate_DifferentPlatforms(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	sess1 := sm.GetOrCreate("telegram", "user123")
	sess2 := sm.GetOrCreate("discord", "user123")

	if sess1.ID == sess2.ID {
		t.Error("different platforms should produce different sessions")
	}
}

func TestSessionManager_GetOrCreate_UpdatesLastActivity(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	sess1 := sm.GetOrCreate("telegram", "user123")
	firstActivity := sess1.LastActivity

	// Small pause so time.Now() changes
	time.Sleep(1 * time.Millisecond)

	sess2 := sm.GetOrCreate("telegram", "user123")
	if !sess2.LastActivity.After(firstActivity) {
		t.Error("LastActivity should be updated on access")
	}
}

func TestSessionManager_Remove(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	sess := sm.GetOrCreate("telegram", "user123")
	sm.Remove(sess.ID)

	newSess := sm.GetOrCreate("telegram", "user123")
	if newSess.ID == sess.ID {
		t.Error("after Remove, GetOrCreate should create a new session")
	}
}

func TestSessionManager_ActiveCount(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	sm.GetOrCreate("telegram", "user1")
	sm.GetOrCreate("telegram", "user2")
	sm.GetOrCreate("discord", "user1")

	if got := sm.ActiveCount(); got != 3 {
		t.Errorf("ActiveCount() = %d, want 3", got)
	}
}

func TestSessionManager_CleanExpired(t *testing.T) {
	sm := NewSessionManager(1 * time.Millisecond)
	sm.GetOrCreate("telegram", "user1")

	time.Sleep(5 * time.Millisecond)

	removed := sm.CleanExpired()
	if removed != 1 {
		t.Errorf("CleanExpired() = %d, want 1", removed)
	}
	if sm.ActiveCount() != 0 {
		t.Errorf("ActiveCount() after clean = %d, want 0", sm.ActiveCount())
	}
}

func TestSessionManager_ExpiryWatcher(t *testing.T) {
	sm := NewSessionManager(5 * time.Millisecond)
	sm.GetOrCreate("telegram", "user1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm.StartExpiryWatcher(ctx, 10*time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	if sm.ActiveCount() != 0 {
		t.Errorf("expected sessions to be expired by watcher, got %d active", sm.ActiveCount())
	}

	cancel()
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			platform := "telegram"
			if n%2 == 0 {
				platform = "discord"
			}
			sess := sm.GetOrCreate(platform, "user")
			if sess.ID == "" {
				t.Errorf("concurrent GetOrCreate returned empty session ID")
			}
		}(i)
	}
	wg.Wait()
}

func TestSessionKey(t *testing.T) {
	key := sessionKey("telegram", "user123")
	if key != "telegram:user123" {
		t.Errorf("sessionKey = %q, want %q", key, "telegram:user123")
	}
}
