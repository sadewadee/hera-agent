// Package matrix_integration provides integration tests for the Matrix platform adapter.
//
// Reference implementation: Tests use httptest.Server to mock the Matrix Client-Server API.
// No real Matrix homeserver or credentials are required.
//
// To run against a real Matrix server:
//
//	HERA_TEST_MATRIX_URL=https://matrix.example.com \
//	HERA_TEST_MATRIX_USER=@bot:example.com \
//	HERA_TEST_MATRIX_TOKEN=<access_token> \
//	go test ./tests/integration/platforms/matrix/...
package matrix_integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/gateway/platforms"
)

// matrixSyncBody is the JSON structure returned by /_matrix/client/v3/sync.
// Mirrors the internal matrixSyncResponse — duplicated here for test isolation.
type matrixSyncBody struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]struct {
			Timeline struct {
				Events []matrixEvent `json:"events"`
			} `json:"timeline"`
		} `json:"join"`
	} `json:"rooms"`
}

type matrixEvent struct {
	Type     string          `json:"type"`
	Sender   string          `json:"sender"`
	Content  json.RawMessage `json:"content"`
	EventID  string          `json:"event_id"`
	OriginTS int64           `json:"origin_server_ts"`
}

// TestMatrixAdapter_ConnectSuccess verifies that Connect succeeds when the mock
// homeserver returns a 200 OK to the /whoami health probe.
func TestMatrixAdapter_ConnectSuccess(t *testing.T) {
	// Mock Matrix server that answers /whoami with 200.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_matrix/client/v3/account/whoami":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@bot:localhost"})
		case r.URL.Path == "/_matrix/client/v3/sync":
			// Return empty sync body to allow the goroutine to proceed without blocking.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(matrixSyncBody{NextBatch: "s1"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	adapter := platforms.NewMatrixAdapter(platforms.MatrixConfig{
		HomeserverURL: srv.URL,
		UserID:        "@bot:localhost",
		AccessToken:   "test-token",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := adapter.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, adapter.IsConnected())

	_ = adapter.Disconnect(ctx)
}

// TestMatrixAdapter_ConnectFailsInvalidToken verifies that Connect returns an error
// when the homeserver returns 401 Unauthorized.
func TestMatrixAdapter_ConnectFailsInvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/client/v3/account/whoami" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	adapter := platforms.NewMatrixAdapter(platforms.MatrixConfig{
		HomeserverURL: srv.URL,
		UserID:        "@bot:localhost",
		AccessToken:   "bad-token",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := adapter.Connect(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid access token")
	assert.False(t, adapter.IsConnected())
}

// TestMatrixAdapter_ConnectFailsMissingURL verifies that Connect validates config.
func TestMatrixAdapter_ConnectFailsMissingURL(t *testing.T) {
	adapter := platforms.NewMatrixAdapter(platforms.MatrixConfig{
		UserID:      "@bot:localhost",
		AccessToken: "token",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.Connect(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "homeserver URL is required")
}

// TestMatrixAdapter_Send verifies that Send correctly calls the Matrix send API.
func TestMatrixAdapter_Send(t *testing.T) {
	var (
		mu         sync.Mutex
		sentBodies []map[string]string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/_matrix/client/v3/account/whoami":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@bot:localhost"})
		case r.URL.Path == "/_matrix/client/v3/sync":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(matrixSyncBody{NextBatch: "s1"})
		default:
			// Capture send events: PUT /_matrix/client/v3/rooms/{roomID}/send/m.room.message/{txnID}
			if r.Method == http.MethodPut {
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
					mu.Lock()
					sentBodies = append(sentBodies, body)
					mu.Unlock()
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"event_id": "$ev1"})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	adapter := platforms.NewMatrixAdapter(platforms.MatrixConfig{
		HomeserverURL: srv.URL,
		UserID:        "@bot:localhost",
		AccessToken:   "test-token",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, adapter.Connect(ctx))
	defer adapter.Disconnect(ctx) //nolint:errcheck

	err := adapter.Send(ctx, "!room1:localhost", gateway.OutgoingMessage{Text: "hello matrix"})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, sentBodies, 1)
	assert.Equal(t, "hello matrix", sentBodies[0]["body"])
	assert.Equal(t, "m.text", sentBodies[0]["msgtype"])
}

// TestMatrixAdapter_NotConnected_SendFails verifies Send returns error when not connected.
func TestMatrixAdapter_NotConnected_SendFails(t *testing.T) {
	adapter := platforms.NewMatrixAdapter(platforms.MatrixConfig{
		HomeserverURL: "http://localhost:9999",
		UserID:        "@bot:localhost",
		AccessToken:   "token",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := adapter.Send(ctx, "!room:localhost", gateway.OutgoingMessage{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}
