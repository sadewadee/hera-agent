package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/llm"
)

// setupTestDB creates a temporary SQLite database for testing.
func setupTestDB(t *testing.T) *SQLiteProvider {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	p, err := NewSQLiteProvider(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteProvider(%q): %v", dbPath, err)
	}
	t.Cleanup(func() {
		p.Close()
	})
	return p
}

func TestNewSQLiteProvider(t *testing.T) {
	t.Run("creates database file", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "test.db")
		p, err := NewSQLiteProvider(dbPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer p.Close()

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "sub", "dir", "test.db")
		p, err := NewSQLiteProvider(dbPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer p.Close()

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created in nested directory")
		}
	})
}

func TestSQLiteProvider_SaveAndGetFacts(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	t.Run("save and retrieve a fact", func(t *testing.T) {
		err := p.SaveFact(ctx, "user1", "name", "Alice")
		if err != nil {
			t.Fatalf("SaveFact: %v", err)
		}

		facts, err := p.GetFacts(ctx, "user1")
		if err != nil {
			t.Fatalf("GetFacts: %v", err)
		}
		if len(facts) != 1 {
			t.Fatalf("expected 1 fact, got %d", len(facts))
		}
		if facts[0].Key != "name" || facts[0].Value != "Alice" {
			t.Errorf("fact = {%s: %s}, want {name: Alice}", facts[0].Key, facts[0].Value)
		}
		if facts[0].UserID != "user1" {
			t.Errorf("user_id = %q, want %q", facts[0].UserID, "user1")
		}
	})

	t.Run("upserts on same key", func(t *testing.T) {
		err := p.SaveFact(ctx, "user2", "city", "Bangkok")
		if err != nil {
			t.Fatalf("SaveFact: %v", err)
		}
		err = p.SaveFact(ctx, "user2", "city", "Chiang Mai")
		if err != nil {
			t.Fatalf("SaveFact (upsert): %v", err)
		}

		facts, err := p.GetFacts(ctx, "user2")
		if err != nil {
			t.Fatalf("GetFacts: %v", err)
		}
		if len(facts) != 1 {
			t.Fatalf("expected 1 fact after upsert, got %d", len(facts))
		}
		if facts[0].Value != "Chiang Mai" {
			t.Errorf("value = %q, want %q", facts[0].Value, "Chiang Mai")
		}
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		facts, err := p.GetFacts(ctx, "unknown-user")
		if err != nil {
			t.Fatalf("GetFacts: %v", err)
		}
		if len(facts) != 0 {
			t.Errorf("expected 0 facts, got %d", len(facts))
		}
	})
}

func TestSQLiteProvider_SaveAndGetConversation(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "Hi there!", Timestamp: time.Now()},
	}

	t.Run("save and retrieve conversation", func(t *testing.T) {
		err := p.SaveConversation(ctx, "session-1", messages)
		if err != nil {
			t.Fatalf("SaveConversation: %v", err)
		}

		got, err := p.GetConversation(ctx, "session-1")
		if err != nil {
			t.Fatalf("GetConversation: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(got))
		}
		if got[0].Role != llm.RoleUser || got[0].Content != "Hello" {
			t.Errorf("message[0] = {%s: %q}, want {user: \"Hello\"}", got[0].Role, got[0].Content)
		}
		if got[1].Role != llm.RoleAssistant || got[1].Content != "Hi there!" {
			t.Errorf("message[1] = {%s: %q}, want {assistant: \"Hi there!\"}", got[1].Role, got[1].Content)
		}
	})

	t.Run("returns empty for unknown session", func(t *testing.T) {
		got, err := p.GetConversation(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("GetConversation: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected 0 messages, got %d", len(got))
		}
	})
}

func TestSQLiteProvider_Search(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	// Seed data
	if err := p.SaveFact(ctx, "user1", "hobby", "playing guitar and singing"); err != nil {
		t.Fatalf("SaveFact: %v", err)
	}
	if err := p.SaveFact(ctx, "user1", "food", "loves Thai food especially green curry"); err != nil {
		t.Fatalf("SaveFact: %v", err)
	}
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "I play guitar every weekend", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "That sounds fun!", Timestamp: time.Now()},
	}
	if err := p.SaveConversation(ctx, "session-search", msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	t.Run("finds facts by keyword", func(t *testing.T) {
		results, err := p.Search(ctx, "guitar", SearchOpts{Limit: 10})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least 1 result, got 0")
		}
		foundFact := false
		for _, r := range results {
			if r.Source == "fact" {
				foundFact = true
			}
		}
		if !foundFact {
			t.Error("expected a fact result in search results")
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		results, err := p.Search(ctx, "guitar", SearchOpts{Limit: 1})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) > 1 {
			t.Errorf("expected at most 1 result, got %d", len(results))
		}
	})

	t.Run("filters by user_id", func(t *testing.T) {
		results, err := p.Search(ctx, "guitar", SearchOpts{Limit: 10, UserID: "nonexistent"})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		for _, r := range results {
			if r.Source == "fact" {
				t.Error("should not return facts for nonexistent user")
			}
		}
	})
}

func TestSQLiteProvider_SessionSearch(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "Let me tell you about my trip to Japan", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "I would love to hear about it!", Timestamp: time.Now()},
	}
	if err := p.SaveConversation(ctx, "session-japan", msgs); err != nil {
		t.Fatalf("SaveConversation: %v", err)
	}

	t.Run("finds sessions by keyword", func(t *testing.T) {
		results, err := p.SessionSearch(ctx, "Japan")
		if err != nil {
			t.Fatalf("SessionSearch: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least 1 session result")
		}
		if results[0].SessionID != "session-japan" {
			t.Errorf("session_id = %q, want %q", results[0].SessionID, "session-japan")
		}
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		results, err := p.SessionSearch(ctx, "xyznonexistentkeyword")
		if err != nil {
			t.Fatalf("SessionSearch: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestExtractUserID(t *testing.T) {
	cases := []struct {
		name      string
		sessionID string
		want      string
	}{
		{"standard format", "telegram:u42:abc-123", "u42"},
		{"two parts", "cli:u7", "u7"},
		{"empty userID field", "telegram::abc", ""},
		{"single part", "local", ""},
		{"empty string", "", ""},
		{"extra colons preserved in uuid", "discord:user1:a:b:c", "user1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractUserID(tc.sessionID)
			if got != tc.want {
				t.Errorf("extractUserID(%q) = %q, want %q", tc.sessionID, got, tc.want)
			}
		})
	}
}

func TestSQLiteProvider_Search_CrossSession(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	sessionA := "telegram:alice:aaa"
	sessionB := "telegram:alice:bbb"
	sessionC := "telegram:bob:ccc"

	save := func(sid, content string) {
		t.Helper()
		msgs := []llm.Message{{Role: llm.RoleUser, Content: content, Timestamp: time.Now()}}
		if err := p.SaveConversation(ctx, sid, msgs); err != nil {
			t.Fatalf("SaveConversation(%s): %v", sid, err)
		}
	}
	save(sessionA, "discussing kubernetes deployments")
	save(sessionB, "more kubernetes cluster talk")
	save(sessionC, "kubernetes notes from bob")

	t.Run("alice finds her own sessions only", func(t *testing.T) {
		results, err := p.Search(ctx, "kubernetes", SearchOpts{Limit: 10, UserID: "alice"})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results for alice, got %d", len(results))
		}
		for _, r := range results {
			if !strings.HasPrefix(r.SourceID, sessionA) && !strings.HasPrefix(r.SourceID, sessionB) {
				t.Errorf("unexpected source_id %q for alice", r.SourceID)
			}
		}
	})

	t.Run("no filter returns all users", func(t *testing.T) {
		results, err := p.Search(ctx, "kubernetes", SearchOpts{Limit: 10})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results without filter, got %d", len(results))
		}
	})
}

func TestSQLiteProvider_Notes(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	t.Run("save and get a note", func(t *testing.T) {
		note := Note{
			UserID:      "u1",
			Type:        NoteTypeUser,
			Name:        "user_role",
			Description: "senior Go engineer",
			Content:     "10 yrs Go, mostly backend + CLI tools; new to Rust",
		}
		if err := p.SaveNote(ctx, note); err != nil {
			t.Fatalf("SaveNote: %v", err)
		}

		got, err := p.GetNote(ctx, "u1", "user_role")
		if err != nil {
			t.Fatalf("GetNote: %v", err)
		}
		if got == nil {
			t.Fatal("expected note, got nil")
		}
		if got.Description != note.Description || got.Content != note.Content {
			t.Errorf("round-trip mismatch: %+v", got)
		}
		if got.Type != NoteTypeUser {
			t.Errorf("type = %q, want user", got.Type)
		}
	})

	t.Run("upsert replaces existing", func(t *testing.T) {
		err := p.SaveNote(ctx, Note{
			UserID: "u1", Type: NoteTypeUser, Name: "user_role",
			Description: "updated desc", Content: "updated content",
		})
		if err != nil {
			t.Fatalf("SaveNote: %v", err)
		}
		got, _ := p.GetNote(ctx, "u1", "user_role")
		if got.Description != "updated desc" {
			t.Errorf("description not updated: %q", got.Description)
		}
	})

	t.Run("update preserves untouched fields", func(t *testing.T) {
		if err := p.UpdateNote(ctx, "u1", "user_role", "", "body only"); err != nil {
			t.Fatalf("UpdateNote: %v", err)
		}
		got, _ := p.GetNote(ctx, "u1", "user_role")
		if got.Description != "updated desc" {
			t.Errorf("description should remain %q, got %q", "updated desc", got.Description)
		}
		if got.Content != "body only" {
			t.Errorf("content not updated: %q", got.Content)
		}
	})

	t.Run("list by type filters", func(t *testing.T) {
		_ = p.SaveNote(ctx, Note{UserID: "u1", Type: NoteTypeFeedback, Name: "prefer_brief", Description: "short replies", Content: "why: told me 2x"})
		_ = p.SaveNote(ctx, Note{UserID: "u1", Type: NoteTypeProject, Name: "ship_friday", Description: "release gate", Content: "freeze after Thu"})

		users, _ := p.ListNotes(ctx, "u1", NoteTypeUser)
		if len(users) != 1 {
			t.Errorf("user notes = %d, want 1", len(users))
		}

		all, _ := p.ListNotes(ctx, "u1", "")
		if len(all) != 3 {
			t.Errorf("all notes = %d, want 3", len(all))
		}
	})

	t.Run("delete removes note and fts entry", func(t *testing.T) {
		if err := p.DeleteNote(ctx, "u1", "prefer_brief"); err != nil {
			t.Fatalf("DeleteNote: %v", err)
		}
		got, _ := p.GetNote(ctx, "u1", "prefer_brief")
		if got != nil {
			t.Error("note should be gone")
		}
		res, _ := p.Search(ctx, "prefer_brief", SearchOpts{Limit: 10, UserID: "u1"})
		for _, r := range res {
			if r.SourceID == "prefer_brief" {
				t.Error("fts entry should be deleted")
			}
		}
	})

	t.Run("search finds note content by keyword", func(t *testing.T) {
		results, err := p.Search(ctx, "release", SearchOpts{Limit: 10, UserID: "u1"})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		var found bool
		for _, r := range results {
			if r.Source == "note:project" && r.SourceID == "ship_friday" {
				found = true
			}
		}
		if !found {
			t.Errorf("note not found in search results: %+v", results)
		}
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		err := p.SaveNote(ctx, Note{UserID: "u1", Type: NoteType("bogus"), Name: "x", Content: "y"})
		if err == nil {
			t.Error("expected error for invalid type")
		}
	})

	t.Run("update missing note errors", func(t *testing.T) {
		err := p.UpdateNote(ctx, "u1", "nonexistent", "", "x")
		if err == nil {
			t.Error("expected error updating nonexistent note")
		}
	})
}

func TestSQLiteProvider_ListUserSessions(t *testing.T) {
	p := setupTestDB(t)
	ctx := context.Background()

	// SaveConversation is replace-per-session, so pass full message
	// slices at once.
	require.NoError(t, p.SaveConversation(ctx, "telegram:alice:s1", []llm.Message{
		{Role: llm.RoleUser, Content: "hello from alice session 1", Timestamp: time.Now()},
		{Role: llm.RoleAssistant, Content: "hi back", Timestamp: time.Now()},
	}))
	require.NoError(t, p.SaveConversation(ctx, "telegram:alice:s2", []llm.Message{
		{Role: llm.RoleUser, Content: "second conversation", Timestamp: time.Now().Add(time.Minute)},
	}))
	require.NoError(t, p.SaveConversation(ctx, "telegram:bob:sB", []llm.Message{
		{Role: llm.RoleUser, Content: "bob's session", Timestamp: time.Now()},
	}))
	require.NoError(t, p.SaveConversation(ctx, "cli:someprefix:different", []llm.Message{
		{Role: llm.RoleUser, Content: "unrelated", Timestamp: time.Now()},
	}))

	t.Run("scoped to alice", func(t *testing.T) {
		sessions, err := p.ListUserSessions(ctx, "alice", 10)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
		// Should NOT include bob or other users.
		for _, s := range sessions {
			assert.Contains(t, s.SessionID, ":alice:")
		}
	})

	t.Run("preview uses first user message", func(t *testing.T) {
		sessions, _ := p.ListUserSessions(ctx, "alice", 10)
		previews := map[string]string{}
		for _, s := range sessions {
			previews[s.SessionID] = s.Preview
		}
		assert.Equal(t, "hello from alice session 1", previews["telegram:alice:s1"])
	})

	t.Run("message count aggregates correctly", func(t *testing.T) {
		sessions, _ := p.ListUserSessions(ctx, "alice", 10)
		counts := map[string]int{}
		for _, s := range sessions {
			counts[s.SessionID] = s.MessageCount
		}
		assert.Equal(t, 2, counts["telegram:alice:s1"])
		assert.Equal(t, 1, counts["telegram:alice:s2"])
	})

	t.Run("limit respected", func(t *testing.T) {
		sessions, _ := p.ListUserSessions(ctx, "alice", 1)
		assert.Len(t, sessions, 1)
	})

	t.Run("no sessions for unknown user", func(t *testing.T) {
		sessions, _ := p.ListUserSessions(ctx, "ghost", 10)
		assert.Empty(t, sessions)
	})

	t.Run("empty userID errors", func(t *testing.T) {
		_, err := p.ListUserSessions(ctx, "", 10)
		assert.Error(t, err)
	})
}

func TestSQLiteProvider_Close(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	p, err := NewSQLiteProvider(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteProvider: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
