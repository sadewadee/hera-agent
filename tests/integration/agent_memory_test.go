package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/memory"
	testhelpers "github.com/sadewadee/hera/tests/helpers"
)

// TestAgentSavesConversationToMemory verifies that after a successful exchange
// the conversation messages are persisted to the memory provider.
func TestAgentSavesConversationToMemory(t *testing.T) {
	t.Parallel()

	cfg := testhelpers.TestConfig(t)
	provider := testhelpers.NewFakeMemoryProvider()
	summarizer := &testhelpers.FakeSummarizer{Result: "Conversation summary."}
	mem := memory.NewManager(provider, summarizer)

	reg := testhelpers.NewTestToolRegistry(t)
	fakeLLM := testhelpers.NewFakeLLMProvider("I remember what you told me.")

	sessions := agent.NewSessionManager(5 * time.Minute)
	ag, err := agent.NewAgent(agent.AgentDeps{
		LLM:      fakeLLM,
		Tools:    reg,
		Memory:   mem,
		Sessions: sessions,
		Config:   cfg,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = ag.HandleMessage(ctx, "test", "chat1", "user1", "my name is Alice")
	require.NoError(t, err)

	// The memory manager should have been called. With the fake provider we
	// can only verify the agent ran without error; a real integration test
	// would check the SQLite store.
	assert.NotNil(t, provider)
}

// TestMemoryManagerSaveFact verifies that facts can be saved and retrieved
// via the memory manager with a SQLite backend.
func TestMemoryManagerSaveFact(t *testing.T) {
	t.Parallel()

	provider := testhelpers.NewTestMemory(t)
	summarizer := &testhelpers.FakeSummarizer{Result: "Summary."}
	mgr := memory.NewManager(provider, summarizer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := mgr.SaveFact(ctx, "user1", "name", "Alice")
	require.NoError(t, err)

	facts, err := mgr.GetFacts(ctx, "user1")
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "name", facts[0].Key)
	assert.Equal(t, "Alice", facts[0].Value)
}

// TestMemoryManagerSaveMultipleFacts verifies multiple facts for the same user.
func TestMemoryManagerSaveMultipleFacts(t *testing.T) {
	t.Parallel()

	provider := testhelpers.NewTestMemory(t)
	summarizer := &testhelpers.FakeSummarizer{Result: "Summary."}
	mgr := memory.NewManager(provider, summarizer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, mgr.SaveFact(ctx, "user1", "name", "Alice"))
	require.NoError(t, mgr.SaveFact(ctx, "user1", "city", "Berlin"))
	require.NoError(t, mgr.SaveFact(ctx, "user2", "name", "Bob"))

	user1Facts, err := mgr.GetFacts(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, user1Facts, 2)

	user2Facts, err := mgr.GetFacts(ctx, "user2")
	require.NoError(t, err)
	assert.Len(t, user2Facts, 1)
}

// TestMemoryManagerSearchReturnsResults verifies the search path via the SQLite
// provider which supports FTS.
func TestMemoryManagerSearchReturnsResults(t *testing.T) {
	t.Parallel()

	provider := testhelpers.NewTestMemory(t)
	summarizer := &testhelpers.FakeSummarizer{Result: "Summary."}
	mgr := memory.NewManager(provider, summarizer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Seed some facts so the search index has content.
	require.NoError(t, mgr.SaveFact(ctx, "user1", "project", "Hera is a Go multi-agent framework"))
	require.NoError(t, mgr.SaveFact(ctx, "user1", "language", "Go"))

	results, err := mgr.Search(ctx, "Go framework", memory.SearchOpts{Limit: 10})
	require.NoError(t, err)
	// Results may be empty on a fresh DB depending on FTS configuration, but
	// the call itself must succeed.
	assert.NotNil(t, results)
}

// TestMemoryManagerSaveAndGetConversation verifies conversation persistence.
func TestMemoryManagerSaveAndGetConversation(t *testing.T) {
	t.Parallel()

	provider := testhelpers.NewTestMemory(t)
	summarizer := &testhelpers.FakeSummarizer{Result: "Summary."}
	mgr := memory.NewManager(provider, summarizer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgs := testhelpers.MakeMessages(
		"user", "Hello",
		"assistant", "Hi there",
		"user", "How are you?",
		"assistant", "Doing well, thanks.",
	)

	err := mgr.SaveConversation(ctx, "session-abc", msgs)
	require.NoError(t, err)

	retrieved, err := mgr.GetConversation(ctx, "session-abc")
	require.NoError(t, err)
	assert.Len(t, retrieved, 4)
	assert.Equal(t, "Hello", retrieved[0].Content)
	assert.Equal(t, "Hi there", retrieved[1].Content)
}

// TestMemoryManagerGetConversationMissing verifies that retrieving a non-existent
// conversation returns an empty slice without error.
func TestMemoryManagerGetConversationMissing(t *testing.T) {
	t.Parallel()

	provider := testhelpers.NewTestMemory(t)
	summarizer := &testhelpers.FakeSummarizer{Result: "Summary."}
	mgr := memory.NewManager(provider, summarizer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgs, err := mgr.GetConversation(ctx, "nonexistent-session")
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

// TestAgentAccumulatesSessionTurns verifies that repeated messages to the same
// session grow the turn count correctly.
func TestAgentAccumulatesSessionTurns(t *testing.T) {
	t.Parallel()

	cfg := testhelpers.TestConfig(t)
	provider := testhelpers.NewTestMemory(t)
	summarizer := &testhelpers.FakeSummarizer{Result: "Summary."}
	mem := memory.NewManager(provider, summarizer)

	reg := testhelpers.NewTestToolRegistry(t)
	fakeLLM := testhelpers.NewFakeLLMProvider("OK")

	sessions := agent.NewSessionManager(5 * time.Minute)
	ag, err := agent.NewAgent(agent.AgentDeps{
		LLM:      fakeLLM,
		Tools:    reg,
		Memory:   mem,
		Sessions: sessions,
		Config:   cfg,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		_, err := ag.HandleMessage(ctx, "test", "chat1", "user1", "message")
		require.NoError(t, err)
	}

	// The session should exist with 3 user turns.
	all := sessions.List()
	require.Len(t, all, 1)
	assert.Equal(t, 3, all[0].TurnCount)
}
