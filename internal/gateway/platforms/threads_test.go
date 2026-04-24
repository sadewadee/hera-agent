package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestNewThreadsAdapter(t *testing.T) {
	a := NewThreadsAdapter("test-token")
	require.NotNil(t, a)
	assert.Equal(t, "threads", a.Name())
	assert.False(t, a.IsConnected())
}

func TestThreadsAdapter_SupportedMedia(t *testing.T) {
	a := NewThreadsAdapter("test-token")
	media := a.SupportedMedia()
	assert.Len(t, media, 2)
	assert.Contains(t, media, gateway.MediaPhoto)
	assert.Contains(t, media, gateway.MediaVideo)
}

func TestThreadsAdapter_SendNotConnected(t *testing.T) {
	a := NewThreadsAdapter("test-token")
	err := a.Send(context.Background(), "", gateway.OutgoingMessage{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestThreadsAdapter_SendEmptyText(t *testing.T) {
	a := NewThreadsAdapter("test-token")
	a.SetConnected(true)
	err := a.Send(context.Background(), "", gateway.OutgoingMessage{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty text")
}

func TestThreadsAdapter_GetChatInfoNotConnected(t *testing.T) {
	a := NewThreadsAdapter("test-token")
	_, err := a.GetChatInfo(context.Background(), "thread-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestThreadsAdapter_Disconnect(t *testing.T) {
	a := NewThreadsAdapter("test-token")
	a.SetConnected(true)
	require.NoError(t, a.Disconnect(context.Background()))
	assert.False(t, a.IsConnected())
}

func TestThreadsAdapter_ConnectMissingToken(t *testing.T) {
	a := NewThreadsAdapter("")
	err := a.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access token")
}

// TestThreadsAdapter_SendPublishFlow exercises the two-step create+publish
// against a mock Graph API.
func TestThreadsAdapter_SendPublishFlow(t *testing.T) {
	var createHit, publishHit bool
	var gotAuth string
	var gotText string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = r.ParseForm()
		switch {
		case strings.HasSuffix(r.URL.Path, "/me/threads"):
			createHit = true
			gotText = r.Form.Get("text")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "container-1"})
		case strings.HasSuffix(r.URL.Path, "/me/threads_publish"):
			publishHit = true
			assert.Equal(t, "container-1", r.Form.Get("creation_id"))
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "published-1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	a := NewThreadsAdapter("secret-token")
	a.httpClient = srv.Client()
	a.SetConnected(true)
	// Redirect base URL by shadowing the package constant via a short-lived
	// request: simplest is to swap the httpClient Transport.
	a.httpClient.Transport = rewriteHostTransport{target: srv.URL, rt: http.DefaultTransport}

	err := a.Send(context.Background(), "", gateway.OutgoingMessage{Text: "hello threads"})
	require.NoError(t, err)
	assert.True(t, createHit, "create endpoint not hit")
	assert.True(t, publishHit, "publish endpoint not hit")
	assert.Equal(t, "Bearer secret-token", gotAuth)
	assert.Equal(t, "hello threads", gotText)
}

// rewriteHostTransport redirects any outbound request to target while keeping
// the original request path and query.
type rewriteHostTransport struct {
	target string
	rt     http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newURL := t.target + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return t.rt.RoundTrip(newReq)
}
