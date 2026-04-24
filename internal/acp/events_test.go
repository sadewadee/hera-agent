package acp

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ToolCallIDTracker ---

func TestToolCallIDTracker_PushPop(t *testing.T) {
	tracker := NewToolCallIDTracker()
	tracker.Push("search", "id-1")
	tracker.Push("search", "id-2")

	assert.Equal(t, "id-1", tracker.Pop("search"))
	assert.Equal(t, "id-2", tracker.Pop("search"))
	assert.Equal(t, "", tracker.Pop("search"))
}

func TestToolCallIDTracker_PopEmpty(t *testing.T) {
	tracker := NewToolCallIDTracker()
	assert.Equal(t, "", tracker.Pop("nonexistent"))
}

func TestToolCallIDTracker_DifferentTools(t *testing.T) {
	tracker := NewToolCallIDTracker()
	tracker.Push("search", "s-1")
	tracker.Push("file", "f-1")
	tracker.Push("search", "s-2")

	assert.Equal(t, "s-1", tracker.Pop("search"))
	assert.Equal(t, "f-1", tracker.Pop("file"))
	assert.Equal(t, "s-2", tracker.Pop("search"))
}

func TestToolCallIDTracker_Concurrent(t *testing.T) {
	tracker := NewToolCallIDTracker()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			tracker.Push("tool", "id")
			tracker.Pop("tool")
		}(i)
	}
	wg.Wait()
}

// --- MakeToolProgressCallback ---

func TestMakeToolProgressCallback_ToolStarted(t *testing.T) {
	tracker := NewToolCallIDTracker()
	var emitted []byte

	emitFn := func(sessionID string, event []byte) {
		emitted = event
	}

	cb := MakeToolProgressCallback("sess-1", tracker, emitFn)
	cb("tool.started", "web_search", "Searching...", map[string]any{"query": "test"})

	require.NotNil(t, emitted)
	var event ToolCallEvent
	require.NoError(t, json.Unmarshal(emitted, &event))
	assert.Equal(t, "web_search", event.ToolName)
	assert.Equal(t, "started", event.Status)
	assert.NotEmpty(t, event.ID)
}

func TestMakeToolProgressCallback_IgnoresOtherEvents(t *testing.T) {
	tracker := NewToolCallIDTracker()
	var emitted []byte

	emitFn := func(sessionID string, event []byte) {
		emitted = event
	}

	cb := MakeToolProgressCallback("sess-1", tracker, emitFn)
	cb("tool.completed", "search", "", nil)

	assert.Nil(t, emitted)
}

// --- MakeThinkingCallback ---

func TestMakeThinkingCallback_EmitsText(t *testing.T) {
	var emitted []byte
	emitFn := func(sessionID string, event []byte) {
		emitted = event
	}

	cb := MakeThinkingCallback("sess-1", emitFn)
	cb("I need to think about this...")

	require.NotNil(t, emitted)
	var event ThinkingEvent
	require.NoError(t, json.Unmarshal(emitted, &event))
	assert.Equal(t, "I need to think about this...", event.Text)
}

func TestMakeThinkingCallback_IgnoresEmpty(t *testing.T) {
	var emitted []byte
	emitFn := func(sessionID string, event []byte) {
		emitted = event
	}

	cb := MakeThinkingCallback("sess-1", emitFn)
	cb("")
	assert.Nil(t, emitted)
}

// --- MakeStepCallback ---

func TestMakeStepCallback_CompletesTools(t *testing.T) {
	tracker := NewToolCallIDTracker()
	tracker.Push("search", "tc-abc")

	var emitted [][]byte
	emitFn := func(sessionID string, event []byte) {
		emitted = append(emitted, event)
	}

	cb := MakeStepCallback("sess-1", tracker, emitFn)
	cb(1, []ToolStepResult{{Name: "search", Result: "found results"}})

	require.Len(t, emitted, 1)
	var event ToolCallEvent
	require.NoError(t, json.Unmarshal(emitted[0], &event))
	assert.Equal(t, "tc-abc", event.ID)
	assert.Equal(t, "search", event.ToolName)
	assert.Equal(t, "completed", event.Status)
	assert.Equal(t, "found results", event.Result)
}

func TestMakeStepCallback_NoMatchingID(t *testing.T) {
	tracker := NewToolCallIDTracker()
	var emitted [][]byte
	emitFn := func(sessionID string, event []byte) {
		emitted = append(emitted, event)
	}

	cb := MakeStepCallback("sess-1", tracker, emitFn)
	cb(1, []ToolStepResult{{Name: "unknown_tool", Result: "data"}})

	assert.Empty(t, emitted)
}

// --- MakeMessageCallback ---

func TestMakeMessageCallback_EmitsText(t *testing.T) {
	var emitted []byte
	emitFn := func(sessionID string, event []byte) {
		emitted = event
	}

	cb := MakeMessageCallback("sess-1", emitFn)
	cb("Hello, world!")

	require.NotNil(t, emitted)
	var event MessageEvent
	require.NoError(t, json.Unmarshal(emitted, &event))
	assert.Equal(t, "Hello, world!", event.Text)
}

func TestMakeMessageCallback_IgnoresEmpty(t *testing.T) {
	var emitted []byte
	emitFn := func(sessionID string, event []byte) {
		emitted = event
	}

	cb := MakeMessageCallback("sess-1", emitFn)
	cb("")
	assert.Nil(t, emitted)
}

// --- generateToolCallID ---

func TestGenerateToolCallID_Format(t *testing.T) {
	id := generateToolCallID()
	assert.True(t, len(id) > 0)
	assert.Contains(t, id, "tc-")
}

func TestGenerateToolCallID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateToolCallID()
		assert.False(t, ids[id], "duplicate ID generated: %s", id)
		ids[id] = true
	}
}
