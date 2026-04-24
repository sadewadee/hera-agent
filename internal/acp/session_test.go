package acp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionManager(t *testing.T) {
	m := NewSessionManager()
	require.NotNil(t, m)
	assert.Empty(t, m.ListSessions())
}

func TestCreateSession(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("/home/user", "gpt-4o")
	require.NotNil(t, s)
	assert.NotEmpty(t, s.SessionID)
	assert.Equal(t, "/home/user", s.CWD)
	assert.Equal(t, "gpt-4o", s.Model)
}

func TestCreateSession_DefaultCWD(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("", "gpt-4o")
	assert.Equal(t, ".", s.CWD)
}

func TestGetSession_Exists(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("/tmp", "claude-3")
	got := m.GetSession(s.SessionID)
	require.NotNil(t, got)
	assert.Equal(t, s.SessionID, got.SessionID)
}

func TestGetSession_NotFound(t *testing.T) {
	m := NewSessionManager()
	got := m.GetSession("non-existent")
	assert.Nil(t, got)
}

func TestRemoveSession_Exists(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("/tmp", "gpt-4o")
	existed := m.RemoveSession(s.SessionID)
	assert.True(t, existed)
	assert.Nil(t, m.GetSession(s.SessionID))
}

func TestRemoveSession_NotFound(t *testing.T) {
	m := NewSessionManager()
	existed := m.RemoveSession("not-here")
	assert.False(t, existed)
}

func TestForkSession(t *testing.T) {
	m := NewSessionManager()
	orig := m.CreateSession("/home", "gpt-4o")
	m.AppendHistory(orig.SessionID, HistoryEntry{Role: "user", Content: "hello"})

	forked := m.ForkSession(orig.SessionID, "/other")
	require.NotNil(t, forked)
	assert.NotEqual(t, orig.SessionID, forked.SessionID)
	assert.Equal(t, "/other", forked.CWD)
	assert.Equal(t, orig.Model, forked.Model)
	assert.Len(t, forked.History, 1)
}

func TestForkSession_InheritsCWD(t *testing.T) {
	m := NewSessionManager()
	orig := m.CreateSession("/original", "gpt-4o")
	forked := m.ForkSession(orig.SessionID, "")
	require.NotNil(t, forked)
	assert.Equal(t, "/original", forked.CWD)
}

func TestForkSession_NotFound(t *testing.T) {
	m := NewSessionManager()
	forked := m.ForkSession("nonexistent", "")
	assert.Nil(t, forked)
}

func TestListSessions(t *testing.T) {
	m := NewSessionManager()
	m.CreateSession("/a", "model-1")
	m.CreateSession("/b", "model-2")

	sessions := m.ListSessions()
	assert.Len(t, sessions, 2)
}

func TestUpdateCWD(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("/original", "gpt-4o")
	updated := m.UpdateCWD(s.SessionID, "/new-path")
	require.NotNil(t, updated)
	assert.Equal(t, "/new-path", updated.CWD)
}

func TestUpdateCWD_NotFound(t *testing.T) {
	m := NewSessionManager()
	result := m.UpdateCWD("nonexistent", "/path")
	assert.Nil(t, result)
}

func TestAppendHistory(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("/tmp", "gpt-4o")
	m.AppendHistory(s.SessionID, HistoryEntry{Role: "user", Content: "hello"})
	m.AppendHistory(s.SessionID, HistoryEntry{Role: "assistant", Content: "world"})

	got := m.GetSession(s.SessionID)
	assert.Len(t, got.History, 2)
}

func TestAppendHistory_NotFound(t *testing.T) {
	m := NewSessionManager()
	// Should not panic
	m.AppendHistory("nonexistent", HistoryEntry{Role: "user", Content: "hello"})
}

func TestGetHistory(t *testing.T) {
	m := NewSessionManager()
	s := m.CreateSession("/tmp", "gpt-4o")
	m.AppendHistory(s.SessionID, HistoryEntry{Role: "user", Content: "test"})

	data, err := m.GetHistory(s.SessionID)
	require.NoError(t, err)

	var history []HistoryEntry
	require.NoError(t, json.Unmarshal(data, &history))
	assert.Len(t, history, 1)
	assert.Equal(t, "user", history[0].Role)
}

func TestGetHistory_NotFound(t *testing.T) {
	m := NewSessionManager()
	data, err := m.GetHistory("nonexistent")
	require.NoError(t, err)
	// Returns empty array
	var history []HistoryEntry
	require.NoError(t, json.Unmarshal(data, &history))
	assert.Empty(t, history)
}

func TestCleanup(t *testing.T) {
	m := NewSessionManager()
	m.CreateSession("/a", "model-1")
	m.CreateSession("/b", "model-2")
	m.Cleanup()
	assert.Empty(t, m.ListSessions())
}

func TestSessionInfo_Fields(t *testing.T) {
	info := SessionInfo{
		SessionID:  "sid",
		CWD:        "/tmp",
		Model:      "gpt-4o",
		HistoryLen: 5,
	}
	assert.Equal(t, "sid", info.SessionID)
	assert.Equal(t, "/tmp", info.CWD)
	assert.Equal(t, "gpt-4o", info.Model)
	assert.Equal(t, 5, info.HistoryLen)
}

func TestConcurrentSessions(t *testing.T) {
	m := NewSessionManager()
	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func() {
			s := m.CreateSession("/tmp", "gpt-4o")
			m.GetSession(s.SessionID)
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Len(t, m.ListSessions(), 10)
}
