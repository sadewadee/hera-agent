package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// WhatsAppAdapter integrates with WhatsApp via the Business Cloud API (Meta).
type WhatsAppAdapter struct {
	BaseAdapter
	phoneNumberID string
	accessToken   string
	verifyToken   string
	callbackAddr  string
	client        *http.Client
	server        *http.Server
	logger        *slog.Logger
}

// WhatsAppConfig holds configuration for the WhatsApp Business API adapter.
type WhatsAppConfig struct {
	PhoneNumberID string `json:"phone_number_id" yaml:"phone_number_id"`
	AccessToken   string `json:"access_token" yaml:"access_token"`
	VerifyToken   string `json:"verify_token" yaml:"verify_token"`
	CallbackAddr  string `json:"callback_addr" yaml:"callback_addr"`
}

// NewWhatsAppAdapter creates a WhatsApp adapter using the Business Cloud API.
func NewWhatsAppAdapter(cfg WhatsAppConfig) *WhatsAppAdapter {
	if cfg.CallbackAddr == "" {
		cfg.CallbackAddr = ":8091"
	}
	return &WhatsAppAdapter{
		BaseAdapter:   BaseAdapter{AdapterName: "whatsapp"},
		phoneNumberID: cfg.PhoneNumberID,
		accessToken:   cfg.AccessToken,
		verifyToken:   cfg.VerifyToken,
		callbackAddr:  cfg.CallbackAddr,
		client:        &http.Client{Timeout: 30 * time.Second},
		logger:        slog.Default(),
	}
}

func (w *WhatsAppAdapter) Connect(_ context.Context) error {
	if w.phoneNumberID == "" || w.accessToken == "" {
		return fmt.Errorf("whatsapp: phone number ID and access token are required")
	}

	w.SetConnected(true)
	w.logger.Info("whatsapp adapter connected", "phone_number_id", w.phoneNumberID)

	mux := http.NewServeMux()
	mux.HandleFunc("/whatsapp/webhook", w.handleWebhook)
	mux.HandleFunc("/whatsapp/health", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(map[string]string{"status": "ok", "platform": "whatsapp"})
	})

	w.server = &http.Server{
		Addr:         w.callbackAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.logger.Error("whatsapp webhook server error", "error", err)
		}
	}()

	return nil
}

func (w *WhatsAppAdapter) Disconnect(ctx context.Context) error {
	w.SetConnected(false)
	if w.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return w.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (w *WhatsAppAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !w.IsConnected() {
		return fmt.Errorf("whatsapp: not connected")
	}

	sendURL := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", w.phoneNumberID)
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                chatID,
		"type":              "text",
		"text": map[string]string{
			"body": msg.Text,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("whatsapp: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("whatsapp: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("whatsapp: send returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *WhatsAppAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "WhatsApp: " + chatID,
		Type:     "private",
		Platform: "whatsapp",
	}, nil
}

func (w *WhatsAppAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaVideo, gateway.MediaAudio, gateway.MediaFile}
}

// handleWebhook handles both verification (GET) and incoming messages (POST).
func (w *WhatsAppAdapter) handleWebhook(rw http.ResponseWriter, r *http.Request) {
	// GET request = webhook verification.
	if r.Method == http.MethodGet {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode == "subscribe" && token == w.verifyToken {
			rw.WriteHeader(http.StatusOK)
			fmt.Fprint(rw, challenge)
			return
		}
		http.Error(rw, "Forbidden", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(rw, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(rw, `{"error":"read body failed"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var webhook waWebhookPayload
	if err := json.Unmarshal(body, &webhook); err != nil {
		http.Error(rw, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	handler := w.Handler()
	if handler == nil {
		rw.WriteHeader(http.StatusOK)
		return
	}

	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" || msg.Text.Body == "" {
					continue
				}

				ts := time.Now()
				if msg.Timestamp != "" {
					if epoch, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
						ts = time.Unix(epoch, 0)
					}
				}

				handler(r.Context(), gateway.IncomingMessage{
					Platform:  "whatsapp",
					ChatID:    msg.From,
					UserID:    msg.From,
					Username:  msg.From,
					Text:      msg.Text.Body,
					Timestamp: ts,
				})
			}
		}
	}

	rw.WriteHeader(http.StatusOK)
}

// waWebhookPayload represents the WhatsApp webhook payload structure.
type waWebhookPayload struct {
	Entry []struct {
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
	} `json:"entry"`
}
