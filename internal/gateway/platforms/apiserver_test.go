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

func TestNewAPIServerAdapter(t *testing.T) {
	a := NewAPIServerAdapter()
	require.NotNil(t, a)
	assert.Equal(t, "apiserver", a.Name())
	assert.Equal(t, ":8080", a.addr)
	assert.False(t, a.IsConnected())
}

func TestNewAPIServerAdapter_CustomAddr(t *testing.T) {
	a := NewAPIServerAdapter(":9999")
	assert.Equal(t, ":9999", a.addr)
}

func TestAPIServerAdapter_GetChatInfo(t *testing.T) {
	a := NewAPIServerAdapter()
	info, err := a.GetChatInfo(t.Context(), "chat-42")
	require.NoError(t, err)
	assert.Equal(t, "chat-42", info.ID)
	assert.Equal(t, "API Chat chat-42", info.Title)
	assert.Equal(t, "private", info.Type)
	assert.Equal(t, "apiserver", info.Platform)
}

func TestAPIServerAdapter_SupportedMedia(t *testing.T) {
	a := NewAPIServerAdapter()
	media := a.SupportedMedia()
	assert.Len(t, media, 2)
	assert.Contains(t, media, gateway.MediaPhoto)
	assert.Contains(t, media, gateway.MediaFile)
}

func TestAPIServerAdapter_HandleMessages_MethodNotAllowed(t *testing.T) {
	a := NewAPIServerAdapter()
	req := httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	rec := httptest.NewRecorder()

	a.handleMessages(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestAPIServerAdapter_HandleMessages_InvalidJSON(t *testing.T) {
	a := NewAPIServerAdapter()
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader("bad json"))
	rec := httptest.NewRecorder()

	a.handleMessages(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIServerAdapter_HandleMessages_EmptyText(t *testing.T) {
	a := NewAPIServerAdapter()
	body := `{"text":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleMessages(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIServerAdapter_HandleMessages_NoHandler(t *testing.T) {
	a := NewAPIServerAdapter()
	body := `{"text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleMessages(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestAPIServerAdapter_HandleMessages_Success(t *testing.T) {
	a := NewAPIServerAdapter()

	a.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		_ = a.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "bot reply"})
	})

	body := `{"text":"hello","user_id":"u1","chat_id":"c1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleMessages(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp apiMessageResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "bot reply", resp.Response)
	assert.Equal(t, "c1", resp.ChatID)
	assert.Equal(t, "u1", resp.UserID)
}

func TestAPIServerAdapter_HandleMessages_DefaultUserAndChat(t *testing.T) {
	a := NewAPIServerAdapter()

	var receivedUserID string
	a.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedUserID = msg.UserID
		_ = a.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "ok"})
	})

	body := `{"text":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	rec := httptest.NewRecorder()

	a.handleMessages(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "api-user", receivedUserID)
}

func TestAPIServerAdapter_HandleConversations(t *testing.T) {
	a := NewAPIServerAdapter()
	req := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
	rec := httptest.NewRecorder()

	a.handleConversations(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["active_requests"])
	assert.Equal(t, "apiserver", resp["platform"])
}

func TestAPIServerAdapter_HandleConversations_MethodNotAllowed(t *testing.T) {
	a := NewAPIServerAdapter()
	req := httptest.NewRequest(http.MethodPost, "/api/conversations", nil)
	rec := httptest.NewRecorder()

	a.handleConversations(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestAPIServerAdapter_HandleHealth(t *testing.T) {
	a := NewAPIServerAdapter()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	a.handleHealth(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "apiserver", resp["platform"])
}

func TestAPIServerAdapter_ConnectDisconnect(t *testing.T) {
	a := NewAPIServerAdapter(":0")
	err := a.Connect(t.Context())
	require.NoError(t, err)
	assert.True(t, a.IsConnected())

	time.Sleep(50 * time.Millisecond)

	err = a.Disconnect(t.Context())
	require.NoError(t, err)
	assert.False(t, a.IsConnected())
}

func TestAPIServerAdapter_Send_NoPending(t *testing.T) {
	a := NewAPIServerAdapter()
	// Send to a chatID with no pending request should not error.
	err := a.Send(t.Context(), "no-chat", gateway.OutgoingMessage{Text: "nobody home"})
	assert.NoError(t, err)
}
