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

func TestEmailAdapter_Connect(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{
		SMTPHost:    "smtp.example.com",
		SMTPPort:    "587",
		FromAddress: "bot@example.com",
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

func TestEmailAdapter_Connect_MissingSMTPHost(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with empty SMTP host")
	}
	if !strings.Contains(err.Error(), "SMTP host is required") {
		t.Errorf("error = %q, want to contain 'SMTP host is required'", err.Error())
	}
}

func TestEmailAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{
		SMTPHost: "smtp.example.com",
	})

	err := adapter.Send(context.Background(), "user@example.com", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestEmailAdapter_Disconnect(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{
		SMTPHost:    "smtp.example.com",
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

func TestEmailAdapter_HandleIncoming(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{
		SMTPHost:    "smtp.example.com",
		WebhookAddr: ":0",
	})

	var receivedText string
	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedChatID = msg.ChatID
	})

	payload := map[string]string{
		"from":    "sender@example.com",
		"to":      "bot@example.com",
		"subject": "Test Subject",
		"body":    "Test Body",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/email/incoming", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleIncoming(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	if receivedText != "Test Subject\n\nTest Body" {
		t.Errorf("received text = %q, want 'Test Subject\\n\\nTest Body'", receivedText)
	}
	if receivedChatID != "sender@example.com" {
		t.Errorf("received chatID = %q, want 'sender@example.com'", receivedChatID)
	}
}

func TestEmailAdapter_HandleIncoming_MethodNotAllowed(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/email/incoming", nil)
	rec := httptest.NewRecorder()

	adapter.handleIncoming(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestEmailAdapter_HandleIncoming_EmptyMessage(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})
	payload := map[string]string{
		"from": "sender@example.com",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/email/incoming", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleIncoming(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestEmailAdapter_HandleIncoming_InvalidJSON(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})
	req := httptest.NewRequest(http.MethodPost, "/email/incoming", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleIncoming(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestEmailAdapter_HandleIncoming_SubjectOnly(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})

	var receivedText string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
	})

	payload := map[string]string{
		"from":    "sender@example.com",
		"subject": "Subject Only",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/email/incoming", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleIncoming(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedText != "Subject Only" {
		t.Errorf("received text = %q, want 'Subject Only'", receivedText)
	}
}

func TestEmailAdapter_HandleIncoming_EmailAddressParsing(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})

	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedChatID = msg.ChatID
	})

	payload := map[string]string{
		"from": "Display Name <sender@example.com>",
		"body": "Test Body",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/email/incoming", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.handleIncoming(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if receivedChatID != "sender@example.com" {
		t.Errorf("chatID = %q, want 'sender@example.com' (parsed from Display Name format)", receivedChatID)
	}
}

func TestEmailAdapter_GetChatInfo(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})
	info, err := adapter.GetChatInfo(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.ID != "user@example.com" {
		t.Errorf("ID = %q, want 'user@example.com'", info.ID)
	}
	if info.Platform != "email" {
		t.Errorf("Platform = %q, want 'email'", info.Platform)
	}
}

func TestEmailAdapter_SupportedMedia(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 1 {
		t.Errorf("SupportedMedia() len = %d, want 1", len(media))
	}
}

func TestEmailAdapter_DefaultPorts(t *testing.T) {
	adapter := NewEmailAdapter(EmailConfig{SMTPHost: "smtp.example.com"})
	if adapter.smtpPort != "587" {
		t.Errorf("default smtpPort = %q, want '587'", adapter.smtpPort)
	}
	if adapter.webhookAddr != ":8085" {
		t.Errorf("default webhookAddr = %q, want ':8085'", adapter.webhookAddr)
	}
}
