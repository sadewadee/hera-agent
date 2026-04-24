package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// EmailAdapter sends messages via SMTP and receives via an incoming webhook.
type EmailAdapter struct {
	BaseAdapter
	smtpHost      string
	smtpPort      string
	smtpUser      string
	smtpPassword  string
	fromAddress   string
	webhookAddr   string
	webhookServer *http.Server
	logger        *slog.Logger
}

// EmailConfig holds configuration for the Email adapter.
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort     string `json:"smtp_port" yaml:"smtp_port"`
	SMTPUser     string `json:"smtp_user" yaml:"smtp_user"`
	SMTPPassword string `json:"smtp_password" yaml:"smtp_password"`
	FromAddress  string `json:"from_address" yaml:"from_address"`
	WebhookAddr  string `json:"webhook_addr" yaml:"webhook_addr"`
}

// NewEmailAdapter creates an Email adapter.
func NewEmailAdapter(cfg EmailConfig) *EmailAdapter {
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}
	if cfg.WebhookAddr == "" {
		cfg.WebhookAddr = ":8085"
	}
	return &EmailAdapter{
		BaseAdapter:  BaseAdapter{AdapterName: "email"},
		smtpHost:     cfg.SMTPHost,
		smtpPort:     cfg.SMTPPort,
		smtpUser:     cfg.SMTPUser,
		smtpPassword: cfg.SMTPPassword,
		fromAddress:  cfg.FromAddress,
		webhookAddr:  cfg.WebhookAddr,
		logger:       slog.Default(),
	}
}

func (e *EmailAdapter) Connect(_ context.Context) error {
	if e.smtpHost == "" {
		return fmt.Errorf("email: SMTP host is required")
	}

	e.SetConnected(true)
	e.logger.Info("email adapter connected", "smtp", e.smtpHost+":"+e.smtpPort, "from", e.fromAddress)

	mux := http.NewServeMux()
	mux.HandleFunc("/email/incoming", e.handleIncoming)
	mux.HandleFunc("/email/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "platform": "email"})
	})

	e.webhookServer = &http.Server{
		Addr:         e.webhookAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := e.webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			e.logger.Error("email webhook server error", "error", err)
		}
	}()

	return nil
}

func (e *EmailAdapter) Disconnect(ctx context.Context) error {
	e.SetConnected(false)
	if e.webhookServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return e.webhookServer.Shutdown(shutdownCtx)
	}
	return nil
}

func (e *EmailAdapter) Send(_ context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !e.IsConnected() {
		return fmt.Errorf("email: not connected")
	}

	to := chatID
	subject := "Message from Hera"
	mime := "MIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n"
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n%s%s",
		e.fromAddress, to, subject, mime, msg.Text)

	auth := smtp.PlainAuth("", e.smtpUser, e.smtpPassword, e.smtpHost)
	addr := e.smtpHost + ":" + e.smtpPort

	if err := smtp.SendMail(addr, auth, e.fromAddress, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("email: send to %s: %w", to, err)
	}

	e.logger.Info("email sent", "to", to)
	return nil
}

func (e *EmailAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "Email: " + chatID,
		Type:     "private",
		Platform: "email",
	}, nil
}

func (e *EmailAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaFile}
}

// WebhookHandler returns the http.HandlerFunc for the /email/incoming endpoint.
// This is exported for integration testing without starting a real HTTP server.
func (e *EmailAdapter) WebhookHandler() http.HandlerFunc {
	return e.handleIncoming
}

type emailWebhookPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (e *EmailAdapter) handleIncoming(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(w, `{"error":"read body failed"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload emailWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	text := payload.Body
	if payload.Subject != "" && text != "" {
		text = payload.Subject + "\n\n" + text
	} else if payload.Subject != "" {
		text = payload.Subject
	}

	if text == "" {
		http.Error(w, `{"error":"empty message"}`, http.StatusBadRequest)
		return
	}

	from := payload.From
	if idx := strings.Index(from, "<"); idx != -1 {
		if end := strings.Index(from, ">"); end > idx {
			from = from[idx+1 : end]
		}
	}

	handler := e.Handler()
	if handler != nil {
		handler(r.Context(), gateway.IncomingMessage{
			Platform:  "email",
			ChatID:    from,
			UserID:    from,
			Username:  payload.From,
			Text:      text,
			Timestamp: time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
