package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLSafetyTool_Name(t *testing.T) {
	tool := &URLSafetyTool{}
	assert.Equal(t, "url_safety", tool.Name())
}

func TestURLSafetyTool_Description(t *testing.T) {
	tool := &URLSafetyTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestURLSafetyTool_InvalidArgs(t *testing.T) {
	tool := &URLSafetyTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestURLSafetyTool_NoAPIKey(t *testing.T) {
	tool := &URLSafetyTool{}
	t.Setenv("GOOGLE_SAFE_BROWSING_KEY", "")
	args, _ := json.Marshal(urlSafetyArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "must error when API key not configured")
	assert.Contains(t, result.Content, "GOOGLE_SAFE_BROWSING_KEY")
}

func TestURLSafetyTool_SafeURL(t *testing.T) {
	// Mock Safe Browsing returns empty object — no matches field → safe.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	orig := safeBrowsingEndpoint
	safeBrowsingEndpoint = srv.URL
	defer func() { safeBrowsingEndpoint = orig }()

	tool := &URLSafetyTool{apiKey: "test-key", client: &http.Client{}, clientVersion: "0.13.2"}
	args, _ := json.Marshal(urlSafetyArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "safe URL must not be error: %s", result.Content)
	assert.Contains(t, result.Content, "SAFE")
	assert.Contains(t, result.Content, "example.com")
}

func TestURLSafetyTool_UnsafeURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"matches":[{"threatType":"MALWARE"}]}`))
	}))
	defer srv.Close()

	orig := safeBrowsingEndpoint
	safeBrowsingEndpoint = srv.URL
	defer func() { safeBrowsingEndpoint = orig }()

	tool := &URLSafetyTool{apiKey: "test-key", client: &http.Client{}, clientVersion: "0.13.2"}
	args, _ := json.Marshal(urlSafetyArgs{URL: "https://malware.example"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "unsafe URL must set IsError")
	assert.Contains(t, result.Content, "UNSAFE")
	assert.Contains(t, result.Content, "MALWARE")
}

func TestURLSafetyTool_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	orig := safeBrowsingEndpoint
	safeBrowsingEndpoint = srv.URL
	defer func() { safeBrowsingEndpoint = orig }()

	tool := &URLSafetyTool{apiKey: "bad-key", client: &http.Client{}, clientVersion: "0.13.2"}
	args, _ := json.Marshal(urlSafetyArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "401")
}

func TestURLSafetyTool_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	orig := safeBrowsingEndpoint
	safeBrowsingEndpoint = srv.URL
	defer func() { safeBrowsingEndpoint = orig }()

	tool := &URLSafetyTool{apiKey: "test-key", client: &http.Client{}, clientVersion: "0.13.2"}
	args, _ := json.Marshal(urlSafetyArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "5xx must surface as error to caller")
	assert.Contains(t, result.Content, "500")
}

func TestURLSafetyTool_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"matches": [notvalid`))
	}))
	defer srv.Close()

	orig := safeBrowsingEndpoint
	safeBrowsingEndpoint = srv.URL
	defer func() { safeBrowsingEndpoint = orig }()

	tool := &URLSafetyTool{apiKey: "test-key", client: &http.Client{}, clientVersion: "0.13.2"}
	args, _ := json.Marshal(urlSafetyArgs{URL: "https://example.com"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "malformed JSON must surface as error, not silent SAFE")
	assert.Contains(t, result.Content, "decode response")
}

// TestURLSafetyTool_NewConstructor verifies the constructor produces a tool
// with both apiKey and client initialised so Execute is safe for concurrent
// use without per-call lazy mutation.
func TestURLSafetyTool_NewConstructor(t *testing.T) {
	t.Setenv("GOOGLE_SAFE_BROWSING_KEY", "from-env")
	tool := NewURLSafetyTool("0.13.2")
	assert.Equal(t, "from-env", tool.apiKey)
	assert.NotNil(t, tool.client)
	assert.Equal(t, "0.13.2", tool.clientVersion)
}
