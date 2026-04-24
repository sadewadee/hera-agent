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

func TestWebsitePolicyTool_Name(t *testing.T) {
	tool := &WebsitePolicyTool{}
	assert.Equal(t, "website_policy", tool.Name())
}

func TestWebsitePolicyTool_Description(t *testing.T) {
	tool := &WebsitePolicyTool{}
	assert.Contains(t, tool.Description(), "robots.txt")
}

func TestWebsitePolicyTool_InvalidArgs(t *testing.T) {
	tool := &WebsitePolicyTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWebsitePolicyTool_FetchRobotsTxt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.Write([]byte("User-agent: *\nDisallow: /private/\n"))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	tool := &WebsitePolicyTool{client: srv.Client()}
	args, _ := json.Marshal(websitePolicyArgs{URL: srv.URL})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "User-agent")
	assert.Contains(t, result.Content, "Disallow")
}

func TestRegisterWebsitePolicy(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterWebsitePolicy(registry)
	_, ok := registry.Get("website_policy")
	assert.True(t, ok)
}
