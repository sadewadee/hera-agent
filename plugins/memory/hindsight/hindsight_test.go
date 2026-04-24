package hindsight

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAvailable_WithAPIKey(t *testing.T) {
	t.Setenv("HINDSIGHT_API_KEY", "test-key")
	t.Setenv("HINDSIGHT_MODE", "")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_LocalMode(t *testing.T) {
	t.Setenv("HINDSIGHT_API_KEY", "")
	t.Setenv("HINDSIGHT_MODE", "local")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithoutCreds(t *testing.T) {
	t.Setenv("HINDSIGHT_API_KEY", "")
	t.Setenv("HINDSIGHT_MODE", "")
	p := New()
	assert.False(t, p.IsAvailable())
}

func TestMemorize_HappyPath(t *testing.T) {
	var capturedMethod, capturedPath string
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	t.Setenv("HINDSIGHT_API_KEY", "test-key")
	t.Setenv("HINDSIGHT_API_URL", server.URL)
	t.Setenv("HINDSIGHT_MODE", "cloud")

	p := New()
	require.NoError(t, p.Initialize("session-1"))

	result, err := p.HandleToolCall("hindsight_memorize", map[string]interface{}{
		"content": "the user prefers Go",
	})
	require.NoError(t, err)

	assert.Equal(t, "POST", capturedMethod)
	assert.Equal(t, "/v1/memorize", capturedPath)
	assert.Equal(t, "the user prefers Go", capturedBody["content"])

	var decoded map[string]string
	require.NoError(t, json.Unmarshal([]byte(result), &decoded))
	assert.Equal(t, "Memorized successfully.", decoded["result"])
}

func TestRecall_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/recall", r.URL.Path)
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-key", authHeader)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"memories":[{"content":"recalled fact"}]}`))
	}))
	defer server.Close()

	t.Setenv("HINDSIGHT_API_KEY", "test-key")
	t.Setenv("HINDSIGHT_API_URL", server.URL)
	t.Setenv("HINDSIGHT_MODE", "cloud")

	p := New()
	require.NoError(t, p.Initialize("session-1"))

	result, err := p.HandleToolCall("hindsight_recall", map[string]interface{}{
		"query": "user preferences",
		"limit": float64(5),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "recalled fact")
}

func TestRecall_MissingQuery_ReturnsError(t *testing.T) {
	t.Setenv("HINDSIGHT_API_KEY", "test-key")
	t.Setenv("HINDSIGHT_MODE", "cloud")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	_, err := p.HandleToolCall("hindsight_recall", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query")
}

func TestMemorize_MissingContent_ReturnsError(t *testing.T) {
	t.Setenv("HINDSIGHT_API_KEY", "test-key")
	t.Setenv("HINDSIGHT_MODE", "cloud")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	_, err := p.HandleToolCall("hindsight_memorize", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

func TestLocalMode_UsesLocalURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"memories":[]}`))
	}))
	defer server.Close()

	// Local mode: no API key required; URL injected via env.
	t.Setenv("HINDSIGHT_API_KEY", "")
	t.Setenv("HINDSIGHT_MODE", "local")
	t.Setenv("HINDSIGHT_API_URL", server.URL)

	p := New()
	require.NoError(t, p.Initialize("session-local"))

	assert.Equal(t, server.URL, p.apiURL)
	assert.Equal(t, "local", p.mode)
	// Empty API key means no Authorization header is set; request should still succeed.
	result, err := p.HandleToolCall("hindsight_recall", map[string]interface{}{
		"query": "anything",
	})
	require.NoError(t, err)
	// No memories but no error — result is the JSON of an empty result set.
	assert.NotEmpty(t, result)
}

func TestGetToolSchemas_ReturnsExpectedTools(t *testing.T) {
	p := New()
	schemas := p.GetToolSchemas()
	require.Len(t, schemas, 2)
	names := map[string]bool{}
	for _, s := range schemas {
		names[s.Name] = true
	}
	assert.True(t, names["hindsight_recall"])
	assert.True(t, names["hindsight_memorize"])
}

func TestUnknownTool_ReturnsError(t *testing.T) {
	t.Setenv("HINDSIGHT_API_KEY", "test-key")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	_, err := p.HandleToolCall("unknown_tool", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
