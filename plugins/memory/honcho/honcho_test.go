package honcho

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearHonchoEnv unsets all honcho-related env vars to give each test a clean slate.
func clearHonchoEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HONCHO_API_KEY",
		"HONCHO_BASE_URL",
		"HERA_HOME",
		"HOME",
	} {
		t.Setenv(key, "")
	}
}

func TestIsAvailable_WithAPIKey(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HONCHO_API_KEY", "test-key")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithBaseURL(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HONCHO_BASE_URL", "http://localhost:9999")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithoutCreds(t *testing.T) {
	clearHonchoEnv(t)
	p := New()
	assert.False(t, p.IsAvailable())
}

// TestHandleToolCall_Profile_HappyPath exercises the honcho_profile tool via
// a mock HTTP server injected through HONCHO_BASE_URL.
func TestHandleToolCall_Profile_HappyPath(t *testing.T) {
	clearHonchoEnv(t)
	// Point HERA_HOME to a temp dir so LoadConfig finds no stale config files.
	t.Setenv("HERA_HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/peer/card", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"card":"user prefers dark mode"}`))
	}))
	defer server.Close()

	t.Setenv("HONCHO_API_KEY", "test-key")
	t.Setenv("HONCHO_BASE_URL", server.URL)

	p := New()
	require.NoError(t, p.Initialize("session-1"))

	result, err := p.HandleToolCall("honcho_profile", map[string]interface{}{})
	require.NoError(t, err)

	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(result), &resp))
	assert.Equal(t, "user prefers dark mode", resp["result"])
}

// TestHandleToolCall_Search_HappyPath exercises the honcho_search tool.
func TestHandleToolCall_Search_HappyPath(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", t.TempDir())

	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"context":"user likes Go and Rust"}`))
	}))
	defer server.Close()

	t.Setenv("HONCHO_API_KEY", "test-key")
	t.Setenv("HONCHO_BASE_URL", server.URL)

	p := New()
	require.NoError(t, p.Initialize("session-2"))

	result, err := p.HandleToolCall("honcho_search", map[string]interface{}{
		"query":      "programming languages",
		"max_tokens": float64(400),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, "programming languages", capturedBody["query"])
	assert.Equal(t, float64(400), capturedBody["max_tokens"])
}

// TestHandleToolCall_Conclude_HappyPath exercises the honcho_conclude tool.
func TestHandleToolCall_Conclude_HappyPath(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", t.TempDir())

	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/conclusions", r.URL.Path)
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	t.Setenv("HONCHO_API_KEY", "test-key")
	t.Setenv("HONCHO_BASE_URL", server.URL)

	p := New()
	require.NoError(t, p.Initialize("session-3"))

	result, err := p.HandleToolCall("honcho_conclude", map[string]interface{}{
		"conclusion": "user writes Go daily",
	})
	require.NoError(t, err)
	assert.Equal(t, "user writes Go daily", capturedBody["content"])

	var resp map[string]string
	require.NoError(t, json.Unmarshal([]byte(result), &resp))
	assert.Contains(t, resp["result"], "user writes Go daily")
}

// TestHandleToolCall_MissingQuery_ReturnsError ensures missing params return errors.
func TestHandleToolCall_MissingQuery_ReturnsError(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", t.TempDir())
	t.Setenv("HONCHO_API_KEY", "test-key")
	t.Setenv("HONCHO_BASE_URL", "http://localhost:0")

	p := New()
	require.NoError(t, p.Initialize("session-4"))

	_, err := p.HandleToolCall("honcho_search", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query")

	_, err = p.HandleToolCall("honcho_conclude", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conclusion")

	_, err = p.HandleToolCall("honcho_context", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query")
}

// TestGetToolSchemas_HybridMode returns all 4 tools in the default hybrid recall mode.
func TestGetToolSchemas_HybridMode(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", t.TempDir())
	t.Setenv("HONCHO_API_KEY", "test-key")
	t.Setenv("HONCHO_BASE_URL", "http://localhost:0")

	p := New()
	require.NoError(t, p.Initialize("session-5"))

	schemas := p.GetToolSchemas()
	require.Len(t, schemas, 4)
	names := map[string]bool{}
	for _, s := range schemas {
		names[s.Name] = true
	}
	assert.True(t, names["honcho_profile"])
	assert.True(t, names["honcho_search"])
	assert.True(t, names["honcho_context"])
	assert.True(t, names["honcho_conclude"])
}

// TestLoadConfig_ReadsEnvOverrides verifies that env vars override file defaults.
func TestLoadConfig_ReadsEnvOverrides(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", t.TempDir())
	t.Setenv("HONCHO_API_KEY", "env-key")
	t.Setenv("HONCHO_BASE_URL", "http://env-url:1234")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "http://env-url:1234", cfg.BaseURL)
	assert.True(t, cfg.Enabled)
}

// TestLoadConfig_ReadsJSONFile verifies that a config file is picked up from HERA_HOME.
func TestLoadConfig_ReadsJSONFile(t *testing.T) {
	dir := t.TempDir()
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", dir)

	configData := `{"apiKey":"file-key","baseUrl":"http://file-url:9999","enabled":true}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "honcho.json"), []byte(configData), 0o600))

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "file-key", cfg.APIKey)
	assert.Equal(t, "http://file-url:9999", cfg.BaseURL)
	assert.True(t, cfg.Enabled)
}

// TestUnknownTool_ReturnsError confirms unknown tools produce errors.
func TestUnknownTool_ReturnsError(t *testing.T) {
	clearHonchoEnv(t)
	t.Setenv("HERA_HOME", t.TempDir())
	t.Setenv("HONCHO_API_KEY", "test-key")
	t.Setenv("HONCHO_BASE_URL", "http://localhost:0")

	p := New()
	require.NoError(t, p.Initialize("session-6"))

	_, err := p.HandleToolCall("nonexistent_tool", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
