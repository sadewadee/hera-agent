package builtin

import (
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
)

func TestRegisterSessionSearch_NilDB(t *testing.T) {
	registry := tools.NewRegistry()
	// Should not panic when db is nil.
	RegisterSessionSearch(registry, nil)
	// No tool should be registered.
	_, ok := registry.Get("session_search")
	assert.False(t, ok)
}

func TestHiddenSessionSources(t *testing.T) {
	assert.Contains(t, HiddenSessionSources, "tool")
}

func TestSessionSearchConstants(t *testing.T) {
	assert.Equal(t, 100_000, maxSessionChars)
	assert.Equal(t, 10000, maxSummaryTokens)
}

// --- Data types ---

func TestSessionMatch_Fields(t *testing.T) {
	m := SessionMatch{
		SessionID:      "sess-1",
		Source:         "cli",
		Model:          "gpt-4",
		SessionStarted: 1700000000,
	}
	assert.Equal(t, "sess-1", m.SessionID)
	assert.Equal(t, "cli", m.Source)
}

func TestSessionMeta_Fields(t *testing.T) {
	m := SessionMeta{
		ID:              "sess-1",
		Source:          "telegram",
		ParentSessionID: "parent-1",
	}
	assert.Equal(t, "sess-1", m.ID)
	assert.Equal(t, "parent-1", m.ParentSessionID)
}

func TestSessionMessage_Fields(t *testing.T) {
	m := SessionMessage{
		Role:     "assistant",
		Content:  "Hello",
		ToolName: "search",
	}
	assert.Equal(t, "assistant", m.Role)
	assert.Equal(t, "search", m.ToolName)
}

func TestSessionListEntry_Fields(t *testing.T) {
	e := SessionListEntry{
		ID:           "sess-1",
		Title:        "Test Session",
		Source:       "cli",
		MessageCount: 10,
		Preview:      "Hello...",
	}
	assert.Equal(t, "sess-1", e.ID)
	assert.Equal(t, 10, e.MessageCount)
}
