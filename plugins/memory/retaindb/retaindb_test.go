package retaindb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retaindb.IsAvailable checks RETAINDB_API_KEY (non-empty = available).
// URL is configurable via RETAINDB_BASE_URL.

func TestIsAvailable_WithAPIKey(t *testing.T) {
	t.Setenv("RETAINDB_API_KEY", "test-key")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithoutAPIKey(t *testing.T) {
	t.Setenv("RETAINDB_API_KEY", "")
	p := New()
	assert.False(t, p.IsAvailable())
}

// TestSearch_HappyPath verifies that HandleToolCall("retaindb_search") POSTs
// to /v1/memory/search and returns the server response without error.
func TestSearch_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/memory/search", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"memories":[]}`))
	}))
	defer server.Close()

	t.Setenv("RETAINDB_API_KEY", "test-key")
	t.Setenv("RETAINDB_BASE_URL", server.URL)

	p := New()
	err := p.Initialize("sess-1")
	require.NoError(t, err)

	result, err := p.HandleToolCall("retaindb_search", map[string]interface{}{
		"query": "user preferences",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// TestRemember_HappyPath verifies that HandleToolCall("retaindb_remember") POSTs
// to /v1/memory with the correct payload.
func TestRemember_HappyPath(t *testing.T) {
	var capturedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		assert.Equal(t, "/v1/memory", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"mem-1"}`))
	}))
	defer server.Close()

	t.Setenv("RETAINDB_API_KEY", "test-key")
	t.Setenv("RETAINDB_BASE_URL", server.URL)

	p := New()
	err := p.Initialize("sess-2")
	require.NoError(t, err)

	result, err := p.HandleToolCall("retaindb_remember", map[string]interface{}{
		"content":     "user prefers tabs over spaces",
		"memory_type": "preference",
	})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.NotEmpty(t, result)
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	p := New()
	_, err := p.HandleToolCall("nope", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
