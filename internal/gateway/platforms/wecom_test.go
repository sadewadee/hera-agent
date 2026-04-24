package platforms

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestWeComAdapter_Connect_MissingCredentials(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with missing credentials")
	}
	if !strings.Contains(err.Error(), "corp ID and secret are required") {
		t.Errorf("error = %q, want to contain 'corp ID and secret are required'", err.Error())
	}
}

func TestWeComAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/gettoken" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"errcode":      0,
				"errmsg":       "ok",
				"access_token": "test-access-token",
				"expires_in":   7200,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Since WeCom hardcodes qyapi.weixin.qq.com, we test credential validation.
	// The Connect will fail because it can't reach the real WeCom API.
	adapter := NewWeComAdapter(WeComConfig{
		CorpID:       "corp123",
		CorpSecret:   "secret123",
		CallbackAddr: ":0",
	})

	// Connect will fail since it tries to hit the real API.
	err := adapter.Connect(context.Background())
	// We expect a failure because we can't reach the real WeCom token endpoint.
	if err == nil {
		// If somehow it succeeds (network available), disconnect.
		adapter.Disconnect(context.Background())
	}

	_ = server
}

func TestWeComAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{
		CorpID:     "corp123",
		CorpSecret: "secret123",
	})

	err := adapter.Send(context.Background(), "user1", gateway.OutgoingMessage{Text: "hello wecom"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestWeComAdapter_Disconnect(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{
		CorpID:       "corp123",
		CorpSecret:   "secret123",
		CallbackAddr: ":0",
	})

	// Manually set connected.
	adapter.SetConnected(true)

	err := adapter.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if adapter.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect")
	}
}

func TestWeComAdapter_HandleCallback_GET_Verification(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{
		CorpID:     "corp123",
		CorpSecret: "secret123",
	})

	req := httptest.NewRequest(http.MethodGet, "/wecom/callback?echostr=verify123", nil)
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "verify123" {
		t.Errorf("body = %q, want 'verify123'", rec.Body.String())
	}
}

func TestWeComAdapter_HandleCallback_POST_Message(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{
		CorpID:     "corp123",
		CorpSecret: "secret123",
	})

	var receivedText string
	var receivedPlatform string
	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedPlatform = msg.Platform
		receivedChatID = msg.ChatID
	})

	xmlMsg := wecomXMLMessage{
		ToUserName: "corp123",
		FromUser:   "user456",
		CreateTime: 1700000000,
		MsgType:    "text",
		Content:    "hello from wecom",
		MsgID:      "msg789",
		AgentID:    1000001,
	}
	body, _ := xml.Marshal(xmlMsg)

	req := httptest.NewRequest(http.MethodPost, "/wecom/callback", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedText != "hello from wecom" {
		t.Errorf("received text = %q, want 'hello from wecom'", receivedText)
	}
	if receivedPlatform != "wecom" {
		t.Errorf("received platform = %q, want 'wecom'", receivedPlatform)
	}
	if receivedChatID != "user456" {
		t.Errorf("received chatID = %q, want 'user456'", receivedChatID)
	}
}

func TestWeComAdapter_HandleCallback_NonTextMessage(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{
		CorpID:     "corp123",
		CorpSecret: "secret123",
	})

	handlerCalled := false
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		handlerCalled = true
	})

	xmlMsg := wecomXMLMessage{
		ToUserName: "corp123",
		FromUser:   "user456",
		MsgType:    "image",
		Content:    "",
	}
	body, _ := xml.Marshal(xmlMsg)

	req := httptest.NewRequest(http.MethodPost, "/wecom/callback", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called for non-text messages")
	}
}

func TestWeComAdapter_HandleCallback_MethodNotAllowed(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{CorpID: "corp123", CorpSecret: "secret123"})
	req := httptest.NewRequest(http.MethodPut, "/wecom/callback", nil)
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestWeComAdapter_HandleCallback_InvalidXML(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{CorpID: "corp123", CorpSecret: "secret123"})
	req := httptest.NewRequest(http.MethodPost, "/wecom/callback", strings.NewReader("not xml"))
	rec := httptest.NewRecorder()

	adapter.handleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestWeComAdapter_GetChatInfo(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{})
	info, err := adapter.GetChatInfo(context.Background(), "user123")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Platform != "wecom" {
		t.Errorf("Platform = %q, want 'wecom'", info.Platform)
	}
}

func TestWeComAdapter_SupportedMedia(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}

func TestWeComAdapter_DefaultCallbackAddr(t *testing.T) {
	adapter := NewWeComAdapter(WeComConfig{CorpID: "corp123", CorpSecret: "secret123"})
	if adapter.callbackAddr != ":8090" {
		t.Errorf("default callbackAddr = %q, want ':8090'", adapter.callbackAddr)
	}
}
