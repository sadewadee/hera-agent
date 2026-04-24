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

func TestFeishuAdapter_Connect(t *testing.T) {
	// Since refreshToken hardcodes the Feishu URL, we can only test
	// credential validation and the missing-credential path.
	adapter := NewFeishuAdapter(FeishuConfig{
		AppID:        "cli_test",
		AppSecret:    "secret_test",
		CallbackAddr: ":0",
	})

	// Connect will fail because it cannot reach the real Feishu token endpoint,
	// but it should NOT fail with "app ID and secret are required".
	err := adapter.Connect(context.Background())
	if err != nil && strings.Contains(err.Error(), "app ID and secret are required") {
		t.Fatalf("should not get credential error when both are set: %v", err)
	}
}

func TestFeishuAdapter_Connect_MissingCredentials(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with missing credentials")
	}
	if !strings.Contains(err.Error(), "app ID and secret are required") {
		t.Errorf("error = %q, want to contain 'app ID and secret are required'", err.Error())
	}
}

func TestFeishuAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{
		AppID:     "cli_test",
		AppSecret: "secret_test",
	})

	err := adapter.Send(context.Background(), "chat123", gateway.OutgoingMessage{Text: "hello feishu"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestFeishuAdapter_Disconnect(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{
		AppID:        "cli_test",
		AppSecret:    "secret_test",
		CallbackAddr: ":0",
	})

	// Manually set connected since we cannot hit the real Feishu API.
	adapter.SetConnected(true)

	err := adapter.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if adapter.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect")
	}
}

func TestFeishuAdapter_HandleEvent(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{
		AppID:     "cli_test",
		AppSecret: "secret_test",
	})

	var receivedText string
	var receivedPlatform string
	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedPlatform = msg.Platform
		receivedChatID = msg.ChatID
	})

	contentJSON, _ := json.Marshal(map[string]string{"text": "hello from feishu"})
	payload := map[string]any{
		"header": map[string]string{
			"event_type": "im.message.receive_v1",
		},
		"event": map[string]any{
			"message": map[string]string{
				"chat_id":      "oc_chat123",
				"message_type": "text",
				"content":      string(contentJSON),
			},
			"sender": map[string]any{
				"sender_id": map[string]string{
					"open_id": "ou_user123",
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/feishu/event", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedText != "hello from feishu" {
		t.Errorf("received text = %q, want 'hello from feishu'", receivedText)
	}
	if receivedPlatform != "feishu" {
		t.Errorf("received platform = %q, want 'feishu'", receivedPlatform)
	}
	if receivedChatID != "oc_chat123" {
		t.Errorf("received chatID = %q, want 'oc_chat123'", receivedChatID)
	}
}

func TestFeishuAdapter_HandleEvent_Challenge(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{AppID: "cli_test", AppSecret: "secret_test"})

	payload := map[string]string{
		"challenge": "challenge-token-123",
		"type":      "url_verification",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/feishu/event", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["challenge"] != "challenge-token-123" {
		t.Errorf("challenge response = %q, want 'challenge-token-123'", resp["challenge"])
	}
}

func TestFeishuAdapter_HandleEvent_MethodNotAllowed(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{AppID: "cli_test", AppSecret: "secret_test"})
	req := httptest.NewRequest(http.MethodGet, "/feishu/event", nil)
	rec := httptest.NewRecorder()

	adapter.handleEvent(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestFeishuAdapter_HandleEvent_InvalidJSON(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{AppID: "cli_test", AppSecret: "secret_test"})
	req := httptest.NewRequest(http.MethodPost, "/feishu/event", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	adapter.handleEvent(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestFeishuAdapter_GetChatInfo(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	info, err := adapter.GetChatInfo(context.Background(), "oc_chat123")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Platform != "feishu" {
		t.Errorf("Platform = %q, want 'feishu'", info.Platform)
	}
}

func TestFeishuAdapter_SupportedMedia(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}
