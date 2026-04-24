package builtin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// newSessionManager spins up a real SQLite-backed manager and seeds
// two sessions for a test user so session_list + session_recall have
// something concrete to exercise.
func newSessionManager(t *testing.T, userID string) *memory.Manager {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "sessions.db")
	prov, err := memory.NewSQLiteProvider(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { prov.Close() })

	mgr := memory.NewManager(prov, &mockSummarizer{})

	ctx := context.Background()
	require.NoError(t, mgr.SaveConversation(ctx, "telegram:"+userID+":old", []llm.Message{
		{Role: llm.RoleUser, Content: "bahas setup telegram", Timestamp: time.Now().Add(-48 * time.Hour)},
		{Role: llm.RoleAssistant, Content: "done, here's the token", Timestamp: time.Now().Add(-48 * time.Hour)},
	}))
	require.NoError(t, mgr.SaveConversation(ctx, "telegram:"+userID+":recent", []llm.Message{
		{Role: llm.RoleUser, Content: "debug memory bug kemarin", Timestamp: time.Now().Add(-2 * time.Hour)},
		{Role: llm.RoleAssistant, Content: "fix was tilde expansion", Timestamp: time.Now().Add(-2 * time.Hour)},
	}))
	// Another user's session that must not leak.
	require.NoError(t, mgr.SaveConversation(ctx, "telegram:otheruser:leak", []llm.Message{
		{Role: llm.RoleUser, Content: "secret", Timestamp: time.Now()},
	}))
	return mgr
}

func TestSessionListTool_ScopedToCurrentUser(t *testing.T) {
	mgr := newSessionManager(t, "alice")
	tool := &SessionListTool{manager: mgr}

	ctx := tools.WithUserID(context.Background(), "alice")
	raw, _ := json.Marshal(sessionListArgs{Limit: 10})
	res, err := tool.Execute(ctx, raw)
	require.NoError(t, err)
	require.False(t, res.IsError, "unexpected error: %s", res.Content)

	// Two sessions for alice, neither leaks otheruser.
	assert.Contains(t, res.Content, "2 past session(s)")
	assert.Contains(t, res.Content, "telegram:alice:old")
	assert.Contains(t, res.Content, "telegram:alice:recent")
	assert.NotContains(t, res.Content, "otheruser")
}

func TestSessionListTool_EmptyForUnknownUser(t *testing.T) {
	mgr := newSessionManager(t, "alice")
	tool := &SessionListTool{manager: mgr}
	ctx := tools.WithUserID(context.Background(), "ghost")

	raw, _ := json.Marshal(sessionListArgs{})
	res, err := tool.Execute(ctx, raw)
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.Contains(t, res.Content, "no past sessions")
}

func TestSessionRecallTool_Summarize(t *testing.T) {
	mgr := newSessionManager(t, "alice")
	tool := &SessionRecallTool{manager: mgr}
	ctx := tools.WithUserID(context.Background(), "alice")

	raw, _ := json.Marshal(sessionRecallArgs{SessionID: "telegram:alice:recent"})
	res, err := tool.Execute(ctx, raw)
	require.NoError(t, err)
	require.False(t, res.IsError, "unexpected error: %s", res.Content)
	// mockSummarizer.Summarize returns "mock summary".
	assert.Contains(t, res.Content, "summary")
	assert.Contains(t, res.Content, "telegram:alice:recent")
}

func TestSessionRecallTool_RawTranscript(t *testing.T) {
	mgr := newSessionManager(t, "alice")
	tool := &SessionRecallTool{manager: mgr}
	ctx := tools.WithUserID(context.Background(), "alice")

	raw, _ := json.Marshal(map[string]any{
		"session_id": "telegram:alice:recent",
		"summarize":  false,
	})
	res, err := tool.Execute(ctx, raw)
	require.NoError(t, err)
	require.False(t, res.IsError, "unexpected error: %s", res.Content)
	assert.Contains(t, res.Content, "debug memory bug kemarin")
	assert.Contains(t, res.Content, "fix was tilde expansion")
}

func TestSessionRecallTool_RefusesOtherUserSession(t *testing.T) {
	mgr := newSessionManager(t, "alice")
	tool := &SessionRecallTool{manager: mgr}

	// ctx says alice but we try to recall otheruser's session
	ctx := tools.WithUserID(context.Background(), "alice")
	raw, _ := json.Marshal(sessionRecallArgs{SessionID: "telegram:otheruser:leak"})
	res, err := tool.Execute(ctx, raw)
	require.NoError(t, err)
	assert.True(t, res.IsError, "must refuse cross-user recall")
	assert.Contains(t, res.Content, "does not belong")
}

func TestSessionRecallTool_MissingSessionID(t *testing.T) {
	mgr := newSessionManager(t, "alice")
	tool := &SessionRecallTool{manager: mgr}
	ctx := tools.WithUserID(context.Background(), "alice")
	raw, _ := json.Marshal(map[string]any{})
	res, err := tool.Execute(ctx, raw)
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "required")
}
