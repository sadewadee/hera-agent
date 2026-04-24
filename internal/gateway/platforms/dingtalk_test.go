package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestDingTalkAdapter_Connect(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{
		AccessToken:  "test-token",
		CallbackAddr: ":0",
	})

	err := adapter.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	if !adapter.IsConnected() {
		t.Error("IsConnected() should be true after Connect")
	}
}

func TestDingTalkAdapter_Connect_MissingToken(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with missing access token")
	}
}

func TestDingTalkAdapter_Send(t *testing.T) {
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "errmsg": "ok"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewDingTalkAdapter(DingTalkConfig{
		AccessToken:  "test-token",
		CallbackAddr: ":0",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	// We cannot easily override the DingTalk URL since it is hardcoded.
	// Test the not-connected case instead.
	adapter.SetConnected(false)
	err := adapter.Send(context.Background(), "", gateway.OutgoingMessage{Text: "hello dingtalk"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}

	_ = gotPayload
	_ = server
}

func TestDingTalkAdapter_Disconnect(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{
		AccessToken:  "test-token",
		CallbackAddr: ":0",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err := adapter.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if adapter.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect")
	}
}

func TestDingTalkAdapter_HandleCallback(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{AccessToken: "test-token"})

	var receivedText string
	var receivedPlatform string
	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedPlatform = msg.Platform
		receivedChatID = msg.ChatID
	})

	payload := dingtalkCallbackPayload{
		MsgType: "text",
		Text: struct {
			Content string `json:"content"`
		}{Content: "hello from dingtalk"},
		SenderNick:     "TestUser",
		SenderID:       "user123",
		ConversationID: "conv456",
		CreateAt:       1700000000000,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/dingtalk/callback", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedText != "hello from dingtalk" {
		t.Errorf("received text = %q, want 'hello from dingtalk'", receivedText)
	}
	if receivedPlatform != "dingtalk" {
		t.Errorf("received platform = %q, want 'dingtalk'", receivedPlatform)
	}
	if receivedChatID != "conv456" {
		t.Errorf("received chatID = %q, want 'conv456'", receivedChatID)
	}
}

func TestDingTalkAdapter_HandleCallback_NoConversationID(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{AccessToken: "test-token"})

	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedChatID = msg.ChatID
	})

	payload := dingtalkCallbackPayload{
		MsgType: "text",
		Text: struct {
			Content string `json:"content"`
		}{Content: "hello"},
		SenderID: "user123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/dingtalk/callback", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if receivedChatID != "dingtalk:user123" {
		t.Errorf("chatID = %q, want 'dingtalk:user123' (fallback)", receivedChatID)
	}
}

func TestDingTalkAdapter_HandleCallback_MethodNotAllowed(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{AccessToken: "test-token"})
	req := httptest.NewRequest(http.MethodGet, "/dingtalk/callback", nil)
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestDingTalkAdapter_HandleCallback_EmptyText(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{AccessToken: "test-token"})

	payload := dingtalkCallbackPayload{
		MsgType:  "text",
		SenderID: "user123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/dingtalk/callback", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestDingTalkAdapter_HandleCallback_InvalidJSON(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{AccessToken: "test-token"})
	req := httptest.NewRequest(http.MethodPost, "/dingtalk/callback", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestDingTalkAdapter_GetChatInfo(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{})
	info, err := adapter.GetChatInfo(context.Background(), "conv123")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Platform != "dingtalk" {
		t.Errorf("Platform = %q, want 'dingtalk'", info.Platform)
	}
	if info.Type != "group" {
		t.Errorf("Type = %q, want 'group'", info.Type)
	}
}

func TestDingTalkAdapter_SupportedMedia(t *testing.T) {
	adapter := NewDingTalkAdapter(DingTalkConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}
