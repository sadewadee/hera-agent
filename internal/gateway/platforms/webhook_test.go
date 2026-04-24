package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestNewWebhookAdapter(t *testing.T) {
	a := NewWebhookAdapter()
	require.NotNil(t, a)
	assert.Equal(t, "webhook", a.Name())
	assert.False(t, a.IsConnected())
}

func TestNewWebhookAdapter_WithConfig(t *testing.T) {
	a := NewWebhookAdapter(WebhookConfig{
		Addr:   ":9090",
		Secret: "my-secret",
	})
	assert.Equal(t, ":9090", a.addr)
	assert.Equal(t, "my-secret", a.secret)
}

func TestWebhookAdapter_GetChatInfo(t *testing.T) {
	a := NewWebhookAdapter()
	info, err := a.GetChatInfo(t.Context(), "test-chat")
	require.NoError(t, err)
	assert.Equal(t, "test-chat", info.ID)
	assert.Equal(t, "Webhook Chat", info.Title)
	assert.Equal(t, "private", info.Type)
	assert.Equal(t, "webhook", info.Platform)
}

func TestWebhookAdapter_SupportedMedia(t *testing.T) {
	a := NewWebhookAdapter()
	media := a.SupportedMedia()
	assert.Len(t, media, 2)
	assert.Contains(t, media, gateway.MediaPhoto)
	assert.Contains(t, media, gateway.MediaFile)
}

func TestWebhookAdapter_HandleWebhook_MethodNotAllowed(t *testing.T) {
	a := NewWebhookAdapter()
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestWebhookAdapter_HandleWebhook_Unauthorized(t *testing.T) {
	a := NewWebhookAdapter(WebhookConfig{Secret: "correct-secret"})
	body := `{"text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Webhook-Secret", "wrong-secret")
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhookAdapter_HandleWebhook_InvalidJSON(t *testing.T) {
	a := NewWebhookAdapter()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookAdapter_HandleWebhook_EmptyText(t *testing.T) {
	a := NewWebhookAdapter()
	body := `{"text":""}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookAdapter_HandleWebhook_NoHandler(t *testing.T) {
	a := NewWebhookAdapter()
	body := `{"text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestWebhookAdapter_HandleWebhook_Success(t *testing.T) {
	a := NewWebhookAdapter()

	a.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		// Simulate immediate response via Send.
		_ = a.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "response"})
	})

	body := `{"text":"hello","user_id":"u1","chat_id":"c1"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "response", resp["response"])
	assert.Equal(t, "c1", resp["chat_id"])
}

func TestWebhookAdapter_HandleWebhook_WithSecret(t *testing.T) {
	a := NewWebhookAdapter(WebhookConfig{Secret: "my-secret"})

	a.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		_ = a.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "ok"})
	})

	body := `{"text":"hello","chat_id":"c1"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Webhook-Secret", "my-secret")
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWebhookAdapter_HandleWebhook_DefaultUserAndChat(t *testing.T) {
	a := NewWebhookAdapter()

	var receivedUserID, receivedChatID string
	a.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedUserID = msg.UserID
		receivedChatID = msg.ChatID
		_ = a.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "ok"})
	})

	body := `{"text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleWebhook(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "webhook-user", receivedUserID)
	assert.True(t, strings.HasPrefix(receivedChatID, "webhook-webhook-user-"))
}

func TestWebhookAdapter_HandleHealth(t *testing.T) {
	a := NewWebhookAdapter()
	req := httptest.NewRequest(http.MethodGet, "/webhook/health", nil)
	rec := httptest.NewRecorder()

	a.handleHealth(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "webhook", resp["platform"])
}

func TestWebhookAdapter_ConnectDisconnect(t *testing.T) {
	a := NewWebhookAdapter(WebhookConfig{Addr: ":0"})
	err := a.Connect(t.Context())
	require.NoError(t, err)
	assert.True(t, a.IsConnected())

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	err = a.Disconnect(t.Context())
	require.NoError(t, err)
	assert.False(t, a.IsConnected())
}
