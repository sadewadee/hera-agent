package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/llm"
)

func testCtx() context.Context { return context.Background() }

func TestSearchMessages(t *testing.T) {
	db := newTestDB(t)

	// Store a conversation with known content.
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "Tell me about elephants in Africa", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "African elephants are the largest land animals on Earth", Timestamp: time.Now()},
	}
	sessionID := "cli:testuser:abc-session"
	if err := db.SaveConversation(testCtx(), sessionID, msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	// Search for a keyword in the conversation.
	results, err := db.SearchMessages("elephants", nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one match for 'elephants', got 0")
	}

	// Verify the session ID is extracted correctly.
	found := false
	for _, r := range results {
		if r.SessionID == sessionID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected session %q in results, got %+v", sessionID, results)
	}
}

func TestSearchMessages_ExcludeSource(t *testing.T) {
	db := newTestDB(t)

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "elephants are grey", Timestamp: time.Now()},
	}
	if err := db.SaveConversation(testCtx(), "tool:bot:xyz", msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}
	if err := db.SaveConversation(testCtx(), "cli:user:xyz2", msgs); err != nil {
		t.Fatalf("SaveConversation cli: %v", err)
	}

	// Exclude "tool" source — should only see cli session.
	results, err := db.SearchMessages("elephants", nil, []string{"tool"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	for _, r := range results {
		if r.Source == "tool" {
			t.Errorf("excluded source 'tool' appeared in results: %+v", r)
		}
	}
}

func TestGetSessionMeta(t *testing.T) {
	db := newTestDB(t)

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "hello", Timestamp: time.Now()},
	}
	sessionID := "telegram:alice:sess1"
	if err := db.SaveConversation(testCtx(), sessionID, msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	meta, err := db.GetSessionMeta(sessionID)
	if err != nil {
		t.Fatalf("GetSessionMeta: %v", err)
	}
	if meta.ID != sessionID {
		t.Errorf("got ID %q, want %q", meta.ID, sessionID)
	}
	if meta.Source != "telegram" {
		t.Errorf("got Source %q, want 'telegram'", meta.Source)
	}
}

func TestGetSessionMeta_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetSessionMeta("nonexistent:u:id")
	if err == nil {
		t.Error("expected error for missing session, got nil")
	}
}

func TestGetSessionMessages(t *testing.T) {
	db := newTestDB(t)

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "ping", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "pong", Timestamp: time.Now().Add(time.Second)},
	}
	sessionID := "cli:u:s1"
	if err := db.SaveConversation(testCtx(), sessionID, msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	out, err := db.GetSessionMessages(sessionID)
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
	if out[0].Role != "user" || out[0].Content != "ping" {
		t.Errorf("unexpected first message: %+v", out[0])
	}
	if out[1].Role != "assistant" || out[1].Content != "pong" {
		t.Errorf("unexpected second message: %+v", out[1])
	}
}

func TestListSessionsRich(t *testing.T) {
	db := newTestDB(t)

	for i, sessID := range []string{"cli:u1:s1", "telegram:u2:s2", "tool:bot:s3"} {
		msgs := []llm.Message{
			{Role: llm.RoleUser, Content: "message for session", Timestamp: time.Now().Add(time.Duration(i) * time.Second)},
		}
		if err := db.SaveConversation(testCtx(), sessID, msgs); err != nil {
			t.Fatalf("SaveConversation %s: %v", sessID, err)
		}
	}

	// List with no exclusions — should see all 3.
	all, err := db.ListSessionsRich(10, nil)
	if err != nil {
		t.Fatalf("ListSessionsRich: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(all))
	}

	// Exclude "tool" source.
	filtered, err := db.ListSessionsRich(10, []string{"tool"})
	if err != nil {
		t.Fatalf("ListSessionsRich filtered: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 sessions after excluding 'tool', got %d", len(filtered))
	}
	for _, s := range filtered {
		if s.Source == "tool" {
			t.Errorf("excluded source 'tool' appeared in listing: %+v", s)
		}
	}
}

// newTestDB creates a temporary SQLite provider for testing.
func newTestDB(t *testing.T) *SQLiteProvider {
	t.Helper()
	db, err := NewSQLiteProvider(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteProvider: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
