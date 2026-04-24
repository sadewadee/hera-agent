package agent

import (
	"strings"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	t.Run("creates session with correct ID format", func(t *testing.T) {
		s := NewSession("telegram", "user123")
		prefix := "telegram:user123:"
		if !strings.HasPrefix(s.ID, prefix) {
			t.Errorf("session ID %q does not have prefix %q", s.ID, prefix)
		}
		// UUID part should be present after the prefix
		uuidPart := s.ID[len(prefix):]
		if len(uuidPart) == 0 {
			t.Error("session ID missing UUID suffix")
		}
	})

	t.Run("sets platform and user ID", func(t *testing.T) {
		s := NewSession("discord", "user456")
		if s.Platform != "discord" {
			t.Errorf("platform = %q, want %q", s.Platform, "discord")
		}
		if s.UserID != "user456" {
			t.Errorf("user_id = %q, want %q", s.UserID, "user456")
		}
	})

	t.Run("initializes timestamps", func(t *testing.T) {
		before := time.Now()
		s := NewSession("cli", "user1")
		after := time.Now()

		if s.CreatedAt.Before(before) || s.CreatedAt.After(after) {
			t.Error("CreatedAt not set to current time")
		}
		if s.UpdatedAt.Before(before) || s.UpdatedAt.After(after) {
			t.Error("UpdatedAt not set to current time")
		}
	})

	t.Run("starts with zero turn count and empty messages", func(t *testing.T) {
		s := NewSession("cli", "user1")
		if s.TurnCount != 0 {
			t.Errorf("turn_count = %d, want 0", s.TurnCount)
		}
		if len(s.Messages) != 0 {
			t.Errorf("messages length = %d, want 0", len(s.Messages))
		}
	})
}

func TestSessionManager_CreateAndGet(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	t.Run("create and retrieve session", func(t *testing.T) {
		s := sm.Create("telegram", "user1")
		if s == nil {
			t.Fatal("Create returned nil")
		}

		got, ok := sm.Get(s.ID)
		if !ok {
			t.Fatal("Get returned false for existing session")
		}
		if got.ID != s.ID {
			t.Errorf("ID = %q, want %q", got.ID, s.ID)
		}
	})

	t.Run("get returns false for nonexistent", func(t *testing.T) {
		_, ok := sm.Get("nonexistent-session-id")
		if ok {
			t.Error("Get returned true for nonexistent session")
		}
	})
}

func TestSessionManager_GetOrCreate(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	t.Run("creates new session when none exists", func(t *testing.T) {
		s := sm.GetOrCreate("telegram", "user1")
		if s == nil {
			t.Fatal("GetOrCreate returned nil")
		}
		if s.Platform != "telegram" || s.UserID != "user1" {
			t.Errorf("session = {%s, %s}, want {telegram, user1}", s.Platform, s.UserID)
		}
	})

	t.Run("returns existing session for same platform+user", func(t *testing.T) {
		s1 := sm.GetOrCreate("discord", "user2")
		s2 := sm.GetOrCreate("discord", "user2")
		if s1.ID != s2.ID {
			t.Errorf("expected same session ID, got %q and %q", s1.ID, s2.ID)
		}
	})

	t.Run("creates different sessions for different users", func(t *testing.T) {
		s1 := sm.GetOrCreate("telegram", "userA")
		s2 := sm.GetOrCreate("telegram", "userB")
		if s1.ID == s2.ID {
			t.Error("expected different session IDs for different users")
		}
	})
}

func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)
	sm.Create("telegram", "user1")
	sm.Create("discord", "user2")

	sessions := sm.List()
	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)
	s := sm.Create("telegram", "user1")

	sm.Delete(s.ID)

	_, ok := sm.Get(s.ID)
	if ok {
		t.Error("session still exists after Delete")
	}
}

func TestSessionManager_Expiry(t *testing.T) {
	// Use a very short timeout for testing
	sm := NewSessionManager(1 * time.Millisecond)
	s := sm.Create("telegram", "user1")

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	sm.CleanExpired()

	_, ok := sm.Get(s.ID)
	if ok {
		t.Error("expired session was not cleaned up")
	}
}
