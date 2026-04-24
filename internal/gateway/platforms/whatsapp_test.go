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

func TestWhatsAppAdapter_Connect(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "test-token",
		VerifyToken:   "verify-me",
		CallbackAddr:  ":0",
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

func TestWhatsAppAdapter_Connect_MissingCredentials(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with missing credentials")
	}
	if !strings.Contains(err.Error(), "phone number ID and access token are required") {
		t.Errorf("error = %q, want to contain 'phone number ID and access token are required'", err.Error())
	}
}

func TestWhatsAppAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
	})

	err := adapter.Send(context.Background(), "+1234567890", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestWhatsAppAdapter_Disconnect(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "test-token",
		CallbackAddr:  ":0",
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

func TestWhatsAppAdapter_HandleWebhook_Verification(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
		VerifyToken:   "my-verify-token",
	})

	req := httptest.NewRequest(http.MethodGet,
		"/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge123",
		nil)
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "challenge123" {
		t.Errorf("body = %q, want 'challenge123'", rec.Body.String())
	}
}

func TestWhatsAppAdapter_HandleWebhook_VerificationFailed(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
		VerifyToken:   "my-verify-token",
	})

	req := httptest.NewRequest(http.MethodGet,
		"/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge123",
		nil)
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestWhatsAppAdapter_HandleWebhook_IncomingMessage(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
		VerifyToken:   "verify",
	})

	var receivedText string
	var receivedChatID string
	var receivedPlatform string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedChatID = msg.ChatID
		receivedPlatform = msg.Platform
	})

	payload := waWebhookPayload{
		Entry: []struct {
			Changes []struct {
				Value struct {
					Messages []struct {
						From      string `json:"from"`
						Timestamp string `json:"timestamp"`
						Type      string `json:"type"`
						Text      struct {
							Body string `json:"body"`
						} `json:"text"`
					} `json:"messages"`
				} `json:"value"`
			} `json:"changes"`
		}{
			{
				Changes: []struct {
					Value struct {
						Messages []struct {
							From      string `json:"from"`
							Timestamp string `json:"timestamp"`
							Type      string `json:"type"`
							Text      struct {
								Body string `json:"body"`
							} `json:"text"`
						} `json:"messages"`
					} `json:"value"`
				}{
					{
						Value: struct {
							Messages []struct {
								From      string `json:"from"`
								Timestamp string `json:"timestamp"`
								Type      string `json:"type"`
								Text      struct {
									Body string `json:"body"`
								} `json:"text"`
							} `json:"messages"`
						}{
							Messages: []struct {
								From      string `json:"from"`
								Timestamp string `json:"timestamp"`
								Type      string `json:"type"`
								Text      struct {
									Body string `json:"body"`
								} `json:"text"`
							}{
								{
									From:      "+5555555555",
									Timestamp: "1700000000",
									Type:      "text",
									Text: struct {
										Body string `json:"body"`
									}{Body: "hello from whatsapp"},
								},
							},
						},
					},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/whatsapp/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedText != "hello from whatsapp" {
		t.Errorf("received text = %q, want 'hello from whatsapp'", receivedText)
	}
	if receivedChatID != "+5555555555" {
		t.Errorf("received chatID = %q, want '+5555555555'", receivedChatID)
	}
	if receivedPlatform != "whatsapp" {
		t.Errorf("received platform = %q, want 'whatsapp'", receivedPlatform)
	}
}

func TestWhatsAppAdapter_HandleWebhook_MethodNotAllowed(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
	})
	req := httptest.NewRequest(http.MethodPut, "/whatsapp/webhook", nil)
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestWhatsAppAdapter_HandleWebhook_InvalidJSON(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
	})
	req := httptest.NewRequest(http.MethodPost, "/whatsapp/webhook", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestWhatsAppAdapter_HandleWebhook_NoHandler(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123456",
		AccessToken:   "token",
	})
	// No handler set.

	payload := waWebhookPayload{}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/whatsapp/webhook", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	adapter.handleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (graceful when no handler)", rec.Code)
	}
}

func TestWhatsAppAdapter_GetChatInfo(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{})
	info, err := adapter.GetChatInfo(context.Background(), "+1234567890")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Platform != "whatsapp" {
		t.Errorf("Platform = %q, want 'whatsapp'", info.Platform)
	}
}

func TestWhatsAppAdapter_SupportedMedia(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 4 {
		t.Errorf("SupportedMedia() len = %d, want 4", len(media))
	}
}

func TestWhatsAppAdapter_DefaultCallbackAddr(t *testing.T) {
	adapter := NewWhatsAppAdapter(WhatsAppConfig{
		PhoneNumberID: "123",
		AccessToken:   "token",
	})
	if adapter.callbackAddr != ":8091" {
		t.Errorf("default callbackAddr = %q, want ':8091'", adapter.callbackAddr)
	}
}
