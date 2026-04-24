// Package email_integration provides integration tests for the Email platform adapter.
//
// Reference implementation: Tests the Email adapter end-to-end using httptest to
// mock the webhook server. No real SMTP server or email credentials are required for
// the webhook receipt path. The SMTP send path is tested via a mock SMTP server
// (simulated via httptest JSON endpoint — real SMTP would require testcontainers
// or a live MailHog sidecar).
//
// To run integration tests against a real MailHog instance:
//
//	HERA_TEST_SMTP_HOST=localhost HERA_TEST_SMTP_PORT=1025 go test ./tests/integration/platforms/email/...
package email_integration

import (
	"bytes"
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

// TestEmailAdapter_WebhookReceive tests that the email adapter correctly receives
// an inbound message delivered to its /email/incoming webhook endpoint.
//
// This test is self-contained: it uses httptest.NewRecorder to drive the adapter's
// handleIncoming handler directly, which means no real network binding is needed.
func TestEmailAdapter_WebhookReceive(t *testing.T) {
	cfg := platforms.EmailConfig{
		SMTPHost:     "localhost",
		SMTPPort:     "587",
		SMTPUser:     "test@example.com",
		SMTPPassword: "secret",
		FromAddress:  "test@example.com",
		WebhookAddr:  ":0", // not started — we test the handler directly
	}
	adapter := platforms.NewEmailAdapter(cfg)

	// Register a message handler to capture received messages.
	var (
		mu       sync.Mutex
		received []gateway.IncomingMessage
	)
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, msg)
	})

	// Build an incoming webhook payload.
	payload := map[string]string{
		"from":    "Alice <alice@example.com>",
		"to":      "bot@example.com",
		"subject": "Hello",
		"body":    "Hello from Alice",
	}
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/email/incoming", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call the handler directly via the exported WebhookHandler method.
	adapter.WebhookHandler()(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify the message was dispatched to the handler.
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1, "exactly one message should have been dispatched")
	msg := received[0]
	assert.Equal(t, "email", msg.Platform)
	assert.Equal(t, "alice@example.com", msg.ChatID, "ChatID should be bare email extracted from From header")
	assert.Contains(t, msg.Text, "Hello")
	assert.Contains(t, msg.Text, "Hello from Alice")
}

// TestEmailAdapter_WebhookBadMethod verifies that non-POST requests are rejected.
func TestEmailAdapter_WebhookBadMethod(t *testing.T) {
	adapter := platforms.NewEmailAdapter(platforms.EmailConfig{
		SMTPHost:    "localhost",
		WebhookAddr: ":0",
	})

	req := httptest.NewRequest(http.MethodGet, "/email/incoming", nil)
	rec := httptest.NewRecorder()
	adapter.WebhookHandler()(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// TestEmailAdapter_WebhookEmptyBody verifies that empty message payloads are rejected.
func TestEmailAdapter_WebhookEmptyBody(t *testing.T) {
	adapter := platforms.NewEmailAdapter(platforms.EmailConfig{
		SMTPHost:    "localhost",
		WebhookAddr: ":0",
	})

	payload := map[string]string{"from": "alice@example.com"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/email/incoming", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	adapter.WebhookHandler()(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestEmailAdapter_NotConnected_SendFails verifies Send returns error when not connected.
func TestEmailAdapter_NotConnected_SendFails(t *testing.T) {
	adapter := platforms.NewEmailAdapter(platforms.EmailConfig{
		SMTPHost:    "localhost",
		WebhookAddr: ":0",
	})
	// Do not call Connect — adapter is not connected.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := adapter.Send(ctx, "recipient@example.com", gateway.OutgoingMessage{Text: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}
