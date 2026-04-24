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

func TestBrowserTool_Name(t *testing.T) {
	tool := &BrowserTool{}
	assert.Equal(t, "browser", tool.Name())
}

func TestBrowserTool_Description(t *testing.T) {
	tool := &BrowserTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestBrowserTool_Parameters(t *testing.T) {
	tool := &BrowserTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestBrowserTool_InvalidArgs(t *testing.T) {
	tool := &BrowserTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBrowserTool_FetchPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Hello World</h1><p>Test content</p></body></html>"))
	}))
	defer srv.Close()

	tool := &BrowserTool{client: srv.Client()}
	args, _ := json.Marshal(browserArgs{URL: srv.URL})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Hello World")
	assert.Contains(t, result.Content, "Test content")
}

func TestBrowserTool_StripsTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("<script>alert('xss')</script><p>Safe content</p>"))
	}))
	defer srv.Close()

	tool := &BrowserTool{client: srv.Client()}
	args, _ := json.Marshal(browserArgs{URL: srv.URL})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Safe content")
	assert.NotContains(t, result.Content, "<script>")
}

func TestBrowserTool_MaxLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		data := make([]byte, 20000)
		for i := range data {
			data[i] = 'x'
		}
		w.Write(data)
	}))
	defer srv.Close()

	tool := &BrowserTool{client: srv.Client()}
	args, _ := json.Marshal(browserArgs{URL: srv.URL, MaxLength: 100})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "[truncated]")
}

func TestBrowserTool_InvalidURL(t *testing.T) {
	tool := &BrowserTool{}
	args, _ := json.Marshal(browserArgs{URL: "http://invalid.invalid.invalid:99999"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
