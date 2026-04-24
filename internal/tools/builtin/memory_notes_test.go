package builtin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// newNotesManager spins up a real SQLite-backed manager so the note tools
// exercise the provider end to end — no mock indirection.
func newNotesManager(t *testing.T) *memory.Manager {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "notes.db")
	prov, err := memory.NewSQLiteProvider(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { prov.Close() })
	return memory.NewManager(prov, &mockSummarizer{})
}

func TestMemoryNoteSave_AndGet(t *testing.T) {
	mgr := newNotesManager(t)
	save := &MemoryNoteSaveTool{manager: mgr}
	get := &MemoryNoteGetTool{manager: mgr}

	raw, _ := json.Marshal(map[string]any{
		"type":        "user",
		"name":        "user_role",
		"description": "senior Go engineer",
		"content":     "10 yrs Go; new to Rust",
		"user_id":     "u1",
	})
	r, err := save.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, r.IsError, "save should succeed: %s", r.Content)
	assert.Contains(t, r.Content, "user")

	raw, _ = json.Marshal(map[string]any{"name": "user_role", "user_id": "u1"})
	r, err = get.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, r.IsError, "get should succeed: %s", r.Content)
	assert.Contains(t, r.Content, "senior Go engineer")
	assert.Contains(t, r.Content, "10 yrs Go")
}

func TestMemoryNoteSave_InvalidType(t *testing.T) {
	tool := &MemoryNoteSaveTool{manager: newNotesManager(t)}
	raw, _ := json.Marshal(map[string]any{
		"type": "bogus", "name": "x", "description": "d", "content": "c",
	})
	r, err := tool.Execute(context.Background(), raw)
	require.NoError(t, err)
	assert.True(t, r.IsError)
	assert.Contains(t, r.Content, "user, feedback, project, reference")
}

func TestMemoryNoteSave_InvalidName(t *testing.T) {
	tool := &MemoryNoteSaveTool{manager: newNotesManager(t)}
	raw, _ := json.Marshal(map[string]any{
		"type": "user", "name": "Has Spaces", "description": "d", "content": "c",
	})
	r, err := tool.Execute(context.Background(), raw)
	require.NoError(t, err)
	assert.True(t, r.IsError)
	assert.Contains(t, r.Content, "name must be lowercase")
}

func TestMemoryNoteList(t *testing.T) {
	mgr := newNotesManager(t)
	save := &MemoryNoteSaveTool{manager: mgr}
	list := &MemoryNoteListTool{manager: mgr}

	raws := []map[string]any{
		{"type": "user", "name": "role", "description": "engineer", "content": "x", "user_id": "u1"},
		{"type": "feedback", "name": "brief", "description": "short replies", "content": "x", "user_id": "u1"},
		{"type": "project", "name": "alpha", "description": "launch Q2", "content": "x", "user_id": "u1"},
	}
	for _, r := range raws {
		raw, _ := json.Marshal(r)
		res, err := save.Execute(context.Background(), raw)
		require.NoError(t, err)
		require.False(t, res.IsError)
	}

	// List all.
	raw, _ := json.Marshal(map[string]any{"user_id": "u1"})
	res, err := list.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.Contains(t, res.Content, "3 note")
	assert.Contains(t, res.Content, "role")
	assert.Contains(t, res.Content, "brief")
	assert.Contains(t, res.Content, "alpha")

	// Filter by type.
	raw, _ = json.Marshal(map[string]any{"type": "feedback", "user_id": "u1"})
	res, err = list.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.Contains(t, res.Content, "1 note")
	assert.Contains(t, res.Content, "brief")
	assert.NotContains(t, res.Content, "role")
}

func TestMemoryNoteList_Empty(t *testing.T) {
	tool := &MemoryNoteListTool{manager: newNotesManager(t)}
	raw, _ := json.Marshal(map[string]any{"user_id": "u1"})
	res, err := tool.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.Contains(t, res.Content, "no notes")
}

func TestMemoryNoteUpdate(t *testing.T) {
	mgr := newNotesManager(t)
	save := &MemoryNoteSaveTool{manager: mgr}
	upd := &MemoryNoteUpdateTool{manager: mgr}
	get := &MemoryNoteGetTool{manager: mgr}

	raw, _ := json.Marshal(map[string]any{
		"type": "user", "name": "role", "description": "engineer", "content": "v1", "user_id": "u1",
	})
	_, _ = save.Execute(context.Background(), raw)

	raw, _ = json.Marshal(map[string]any{
		"name": "role", "content": "v2", "user_id": "u1",
	})
	res, err := upd.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, res.IsError, "update failed: %s", res.Content)

	raw, _ = json.Marshal(map[string]any{"name": "role", "user_id": "u1"})
	res, err = get.Execute(context.Background(), raw)
	require.NoError(t, err)
	assert.Contains(t, res.Content, "v2")
	// description untouched because we omitted it
	assert.Contains(t, res.Content, "engineer")
}

func TestMemoryNoteUpdate_RequiresOneField(t *testing.T) {
	tool := &MemoryNoteUpdateTool{manager: newNotesManager(t)}
	raw, _ := json.Marshal(map[string]any{"name": "role"})
	res, err := tool.Execute(context.Background(), raw)
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, strings.ToLower(res.Content), "at least one")
}

func TestMemoryNoteDelete(t *testing.T) {
	mgr := newNotesManager(t)
	save := &MemoryNoteSaveTool{manager: mgr}
	del := &MemoryNoteDeleteTool{manager: mgr}
	get := &MemoryNoteGetTool{manager: mgr}

	raw, _ := json.Marshal(map[string]any{
		"type": "user", "name": "gone", "description": "d", "content": "c", "user_id": "u1",
	})
	_, _ = save.Execute(context.Background(), raw)

	raw, _ = json.Marshal(map[string]any{"name": "gone", "user_id": "u1"})
	res, err := del.Execute(context.Background(), raw)
	require.NoError(t, err)
	require.False(t, res.IsError, "delete failed: %s", res.Content)

	raw, _ = json.Marshal(map[string]any{"name": "gone", "user_id": "u1"})
	res, err = get.Execute(context.Background(), raw)
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "no note named")
}

func TestMemoryNoteDelete_NonExistent(t *testing.T) {
	tool := &MemoryNoteDeleteTool{manager: newNotesManager(t)}
	raw, _ := json.Marshal(map[string]any{"name": "nope", "user_id": "u1"})
	res, err := tool.Execute(context.Background(), raw)
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "not found")
}

// TestMemoryNoteSave_UsesContextUserID covers the v0.9.4 fix: when the
// LLM emits memory_note_save without a user_id field, the tool must
// fall back to the session user ID the agent attached via
// tools.WithUserID rather than dumping everything into the "default"
// bucket. This was the bug that prevented Telegram bot users from
// seeing their own memory after a restart.
func TestMemoryNoteSave_UsesContextUserID(t *testing.T) {
	mgr := newNotesManager(t)
	save := &MemoryNoteSaveTool{manager: mgr}
	list := &MemoryNoteListTool{manager: mgr}

	// Args deliberately omit user_id — simulates the LLM's normal output.
	raw, _ := json.Marshal(map[string]any{
		"type":        "user",
		"name":        "from_ctx",
		"description": "created under ctx userID",
		"content":     "Master Sadewa is the maker",
	})
	ctx := tools.WithUserID(context.Background(), "tg-810485832")
	res, err := save.Execute(ctx, raw)
	require.NoError(t, err)
	require.False(t, res.IsError, "save failed: %s", res.Content)

	// List under the ctx user ID — the note must be there.
	listRaw, _ := json.Marshal(map[string]any{"user_id": "tg-810485832"})
	listRes, err := list.Execute(context.Background(), listRaw)
	require.NoError(t, err)
	require.False(t, listRes.IsError)
	assert.Contains(t, listRes.Content, "from_ctx")

	// List under "default" — must be empty, proving no bleed.
	defaultRaw, _ := json.Marshal(map[string]any{"user_id": "default"})
	defaultRes, err := list.Execute(context.Background(), defaultRaw)
	require.NoError(t, err)
	assert.Contains(t, defaultRes.Content, "no notes")
}

func TestMemoryNoteSave_ExplicitArgsWinOverContext(t *testing.T) {
	mgr := newNotesManager(t)
	save := &MemoryNoteSaveTool{manager: mgr}

	raw, _ := json.Marshal(map[string]any{
		"type":        "user",
		"name":        "explicit_user",
		"description": "args.user_id should win",
		"content":     "whoever is in args wins",
		"user_id":     "alice", // explicit — must override ctx
	})
	ctx := tools.WithUserID(context.Background(), "bob")
	res, err := save.Execute(ctx, raw)
	require.NoError(t, err)
	require.False(t, res.IsError)

	// The note should exist under "alice", not "bob".
	get := &MemoryNoteGetTool{manager: mgr}
	getRaw, _ := json.Marshal(map[string]any{"name": "explicit_user", "user_id": "alice"})
	getRes, err := get.Execute(context.Background(), getRaw)
	require.NoError(t, err)
	require.False(t, getRes.IsError, "note not found under explicit user: %s", getRes.Content)

	getRawBob, _ := json.Marshal(map[string]any{"name": "explicit_user", "user_id": "bob"})
	bobRes, err := get.Execute(context.Background(), getRawBob)
	require.NoError(t, err)
	assert.True(t, bobRes.IsError, "note must not appear under ctx user when args are explicit")
}
