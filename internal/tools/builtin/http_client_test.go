package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientTool_Name(t *testing.T) {
	tool := &HTTPClientTool{}
	assert.Equal(t, "http_request", tool.Name())
}

func TestHTTPClientTool_Description(t *testing.T) {
	tool := &HTTPClientTool{}
	assert.Contains(t, tool.Description(), "HTTP")
}

func TestHTTPClientTool_InvalidArgs(t *testing.T) {
	tool := &HTTPClientTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHTTPClientTool_GetRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("X-Test", "value")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	tool := &HTTPClientTool{client: srv.Client()}
	args, _ := json.Marshal(httpClientArgs{Method: "GET", URL: srv.URL})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "200")
	assert.Contains(t, result.Content, `{"status":"ok"}`)
}

func TestHTTPClientTool_PostRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(201)
		w.Write([]byte("created"))
	}))
	defer srv.Close()

	tool := &HTTPClientTool{client: srv.Client()}
	args, _ := json.Marshal(httpClientArgs{
		Method:  "POST",
		URL:     srv.URL,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"name":"test"}`,
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "201")
	assert.Contains(t, result.Content, "created")
}

func TestHTTPClientTool_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	tool := &HTTPClientTool{client: srv.Client()}
	args, _ := json.Marshal(httpClientArgs{
		Method:  "GET",
		URL:     srv.URL,
		Headers: map[string]string{"Authorization": "Bearer token123"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestRegisterHTTPClient(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterHTTPClient(registry)
	_, ok := registry.Get("http_request")
	assert.True(t, ok)
}
