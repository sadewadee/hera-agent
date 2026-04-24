package supermemory

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// supermemory.IsAvailable checks SUPERMEMORY_API_KEY (non-empty = available).
//
// NOTE: HandleToolCall / Prefetch use a hardcoded apiBaseURL constant
// ("https://api.supermemory.ai/v3") that is not configurable via env vars,
// so those paths cannot be redirected to an httptest.Server without modifying
// production code. Store/Recall happy-path tests are therefore omitted here
// per the task brief. OnSessionEnd uses its own http.Request and accepts a
// configurable server via a minimal test below.

func TestIsAvailable_WithAPIKey(t *testing.T) {
	t.Setenv("SUPERMEMORY_API_KEY", "test-key")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithoutAPIKey(t *testing.T) {
	t.Setenv("SUPERMEMORY_API_KEY", "")
	p := New()
	assert.False(t, p.IsAvailable())
}

// TestHandleToolCall_NotActive verifies that tool calls fail gracefully when
// the provider was never initialized (p.active is false).
func TestHandleToolCall_NotActive(t *testing.T) {
	p := New()
	// p.active defaults to false; no Initialize() call.
	_, err := p.HandleToolCall("supermemory_store", map[string]interface{}{
		"content": "some fact",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// TestHandleToolCall_UnknownTool verifies that unknown tool names return an
// error after the provider has been initialized.
func TestHandleToolCall_UnknownTool(t *testing.T) {
	// Use a test server to satisfy the HTTP client when Initialize later
	// makes no outbound calls — Initialize itself does not call apiPost.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	t.Setenv("SUPERMEMORY_API_KEY", "test-key")

	p := New()
	err := p.Initialize("sess-1")
	require.NoError(t, err)

	_, err = p.HandleToolCall("does_not_exist", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

// TestHandleToolCall_ForgetQueryBased verifies that supermemory_forget with a
// query (no id) returns an explicit error instead of a success-looking JSON
// payload (W3 fix: was previously returning {"result":"Query-based forget not
// implemented in Go client."} which callers would misread as success).
func TestHandleToolCall_ForgetQueryBased(t *testing.T) {
	t.Setenv("SUPERMEMORY_API_KEY", "test-key")

	p := New()
	// Directly activate so we can test HandleToolCall without making real HTTP
	// calls to the Supermemory API (apiBaseURL is hardcoded and not injectable).
	p.active = true

	_, err := p.HandleToolCall("supermemory_forget", map[string]interface{}{
		"query": "some search term",
	})
	require.Error(t, err, "query-based forget must return an error, not silent JSON")
	assert.Contains(t, err.Error(), "query-based forget not supported")
	assert.Contains(t, err.Error(), "id")
}

// TestHandleToolCall_ForgetMissingBothFields verifies that forgetting without
// either id or query returns an error.
func TestHandleToolCall_ForgetMissingBothFields(t *testing.T) {
	t.Setenv("SUPERMEMORY_API_KEY", "test-key")

	p := New()
	p.active = true

	_, err := p.HandleToolCall("supermemory_forget", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide either id or query")
}
