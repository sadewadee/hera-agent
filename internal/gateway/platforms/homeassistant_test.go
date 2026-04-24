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

func TestHomeAssistantAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/" && r.Method == http.MethodGet {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer ha-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "API running."})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{
		HAURL:       server.URL,
		Token:       "ha-token",
		WebhookAddr: ":0",
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

func TestHomeAssistantAdapter_Connect_EmptyURL(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with empty HA URL")
	}
}

func TestHomeAssistantAdapter_Connect_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{
		HAURL: server.URL,
		Token: "bad-token",
	})

	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with invalid token")
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Errorf("error = %q, want to contain 'invalid token'", err.Error())
	}
}

func TestHomeAssistantAdapter_Send(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/services/notify/notify" && r.Method == http.MethodPost {
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{
		HAURL:       server.URL,
		Token:       "ha-token",
		WebhookAddr: ":0",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "", gateway.OutgoingMessage{Text: "hello HA"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotPath != "/api/services/notify/notify" {
		t.Errorf("Send path = %q, want '/api/services/notify/notify'", gotPath)
	}
	if gotAuth != "Bearer ha-token" {
		t.Errorf("Authorization = %q, want 'Bearer ha-token'", gotAuth)
	}
	if gotPayload["message"] != "hello HA" {
		t.Errorf("payload message = %v, want 'hello HA'", gotPayload["message"])
	}
}

func TestHomeAssistantAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{
		HAURL: "http://localhost:1",
		Token: "token",
	})

	err := adapter.Send(context.Background(), "", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestHomeAssistantAdapter_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{
		HAURL:       server.URL,
		Token:       "ha-token",
		WebhookAddr: ":0",
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

func TestHomeAssistantAdapter_HandleWebhook(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{
		HAURL: "http://localhost:1",
		Token: "ha-token",
	})

	var receivedText string
	var receivedPlatform string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedPlatform = msg.Platform
	})

	payload := haWebhookPayload{
		Event:   "automation_triggered",
		Message: "doorbell rang",
		UserID:  "user123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/ha/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedText != "doorbell rang" {
		t.Errorf("received text = %q, want 'doorbell rang'", receivedText)
	}
	if receivedPlatform != "homeassistant" {
		t.Errorf("received platform = %q, want 'homeassistant'", receivedPlatform)
	}
}

func TestHomeAssistantAdapter_HandleWebhook_MethodNotAllowed(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{HAURL: "http://localhost:1"})
	req := httptest.NewRequest(http.MethodGet, "/ha/webhook", nil)
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHomeAssistantAdapter_HandleWebhook_EmptyMessage(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{HAURL: "http://localhost:1"})
	payload := haWebhookPayload{Event: "test"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/ha/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHomeAssistantAdapter_HandleWebhook_DefaultUserID(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{HAURL: "http://localhost:1"})

	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedChatID = msg.ChatID
	})

	payload := haWebhookPayload{Message: "test message"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/ha/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if receivedChatID != "ha:ha-user" {
		t.Errorf("chatID = %q, want 'ha:ha-user' (default user)", receivedChatID)
	}
}

func TestHomeAssistantAdapter_GetChatInfo(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{})
	info, err := adapter.GetChatInfo(context.Background(), "ha:user1")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Platform != "homeassistant" {
		t.Errorf("Platform = %q, want 'homeassistant'", info.Platform)
	}
}

func TestHomeAssistantAdapter_SupportedMedia(t *testing.T) {
	adapter := NewHomeAssistantAdapter(HomeAssistantConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 0 {
		t.Errorf("SupportedMedia() len = %d, want 0", len(media))
	}
}
