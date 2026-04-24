package builtin

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
)

func TestSQLiteSessionDB_SearchMessages(t *testing.T) {
	p := newAdapterTestDB(t)
	adapter := NewSQLiteSessionDB(p)

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "how do penguins survive", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "penguins huddle together for warmth", Timestamp: time.Now()},
	}
	if err := p.SaveConversation(context.Background(), "cli:alice:s1", msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	results, err := adapter.SearchMessages("penguins", nil, nil, 10, 0)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'penguins', got none")
	}
	if results[0].SessionID != "cli:alice:s1" {
		t.Errorf("unexpected SessionID %q", results[0].SessionID)
	}
}

func TestSQLiteSessionDB_GetSession(t *testing.T) {
	p := newAdapterTestDB(t)
	adapter := NewSQLiteSessionDB(p)

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hi", Timestamp: time.Now()}}
	if err := p.SaveConversation(context.Background(), "telegram:bob:s2", msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	meta, err := adapter.GetSession("telegram:bob:s2")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if meta.ID != "telegram:bob:s2" {
		t.Errorf("got ID %q, want %q", meta.ID, "telegram:bob:s2")
	}
	if meta.Source != "telegram" {
		t.Errorf("got Source %q, want 'telegram'", meta.Source)
	}
}

func TestSQLiteSessionDB_GetSessionMessages(t *testing.T) {
	p := newAdapterTestDB(t)
	adapter := NewSQLiteSessionDB(p)

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "question", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "answer", Timestamp: time.Now().Add(time.Second)},
	}
	if err := p.SaveConversation(context.Background(), "discord:charlie:s3", msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	out, err := adapter.GetSessionMessages("discord:charlie:s3")
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
}

func TestSQLiteSessionDB_ListSessionsRich(t *testing.T) {
	p := newAdapterTestDB(t)
	adapter := NewSQLiteSessionDB(p)

	for _, sessID := range []string{"cli:u1:sa", "telegram:u2:sb"} {
		msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello", Timestamp: time.Now()}}
		if err := p.SaveConversation(context.Background(), sessID, msgs); err != nil {
			t.Fatalf("SaveConversation %s: %v", sessID, err)
		}
	}

	entries, err := adapter.ListSessionsRich(5, nil)
	if err != nil {
		t.Fatalf("ListSessionsRich: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(entries))
	}
}

func TestSessionDBFromManager_NilManager(t *testing.T) {
	db := SessionDBFromManager(nil)
	if db != nil {
		t.Error("expected nil SessionDB for nil Manager")
	}
}

func TestSessionDBFromManager_WithSQLite(t *testing.T) {
	p := newAdapterTestDB(t)
	mgr := memory.NewManager(p, nil)
	db := SessionDBFromManager(mgr)
	if db == nil {
		t.Error("expected non-nil SessionDB for SQLite-backed Manager")
	}
}

func newAdapterTestDB(t *testing.T) *memory.SQLiteProvider {
	t.Helper()
	p, err := memory.NewSQLiteProvider(filepath.Join(t.TempDir(), "adapter_test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteProvider: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}
