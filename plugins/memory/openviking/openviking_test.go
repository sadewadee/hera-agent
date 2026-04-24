package openviking

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openviking.IsAvailable checks OPENVIKING_ENDPOINT (non-empty = available).

func TestIsAvailable_WithEndpoint(t *testing.T) {
	t.Setenv("OPENVIKING_ENDPOINT", "http://localhost:1933")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithoutEndpoint(t *testing.T) {
	t.Setenv("OPENVIKING_ENDPOINT", "")
	p := New()
	assert.False(t, p.IsAvailable())
}

// TestSearch_HappyPath verifies that HandleToolCall("viking_search") issues a
// POST to /api/v1/search/find and returns the server's JSON response.
//
// Initialize() calls healthCheck() which issues a GET /health before any tool
// call, so the test handler must accept both requests.
func TestSearch_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			// Health check issued by Initialize — just respond 200.
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/search/find", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	t.Setenv("OPENVIKING_ENDPOINT", server.URL)
	t.Setenv("OPENVIKING_API_KEY", "test-key")

	p := New()
	err := p.Initialize("sess-1")
	require.NoError(t, err)

	result, err := p.HandleToolCall("viking_search", map[string]interface{}{
		"query": "Go concurrency patterns",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// TestRemember_HappyPath verifies that HandleToolCall("viking_remember") POSTs
// the content to the sessions messages endpoint.
//
// Initialize() calls healthCheck() first, so the handler ignores the GET
// /health request and captures only the subsequent POST path.
func TestRemember_HappyPath(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	t.Setenv("OPENVIKING_ENDPOINT", server.URL)
	t.Setenv("OPENVIKING_API_KEY", "test-key")

	p := New()
	err := p.Initialize("sess-2")
	require.NoError(t, err)

	result, err := p.HandleToolCall("viking_remember", map[string]interface{}{
		"content":  "user prefers dark mode",
		"category": "preference",
	})
	require.NoError(t, err)
	assert.Contains(t, capturedPath, "sess-2", "path should include the session ID")
	assert.NotEmpty(t, result)
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	p := New()
	_, err := p.HandleToolCall("nope", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
