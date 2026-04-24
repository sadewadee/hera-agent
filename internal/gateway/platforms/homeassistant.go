package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// HomeAssistantAdapter integrates with Home Assistant via its REST API.
// Notifications are sent via the notify service; incoming messages are
// received through an HTTP webhook that HA automations can POST to.
type HomeAssistantAdapter struct {
	BaseAdapter
	haURL       string
	token       string
	webhookAddr string
	client      *http.Client
	server      *http.Server
	logger      *slog.Logger
}

// HomeAssistantConfig holds configuration for the Home Assistant adapter.
type HomeAssistantConfig struct {
	HAURL       string `json:"ha_url" yaml:"ha_url"`
	Token       string `json:"token" yaml:"token"`
	WebhookAddr string `json:"webhook_addr" yaml:"webhook_addr"`
}

// NewHomeAssistantAdapter creates a Home Assistant adapter.
func NewHomeAssistantAdapter(cfg HomeAssistantConfig) *HomeAssistantAdapter {
	if cfg.WebhookAddr == "" {
		cfg.WebhookAddr = ":8087"
	}
	return &HomeAssistantAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "homeassistant"},
		haURL:       cfg.HAURL,
		token:       cfg.Token,
		webhookAddr: cfg.WebhookAddr,
		client:      &http.Client{Timeout: 30 * time.Second},
		logger:      slog.Default(),
	}
}

func (h *HomeAssistantAdapter) Connect(ctx context.Context) error {
	if h.haURL == "" {
		return fmt.Errorf("homeassistant: HA URL is required")
	}

	// Verify connectivity.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.haURL+"/api/", nil)
	if err != nil {
		return fmt.Errorf("homeassistant: create health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("homeassistant: connect to %s: %w", h.haURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("homeassistant: invalid token")
	}

	h.SetConnected(true)
	h.logger.Info("homeassistant adapter connected", "url", h.haURL)

	// Start webhook server for incoming events.
	mux := http.NewServeMux()
	mux.HandleFunc("/ha/webhook", h.handleWebhook)
	mux.HandleFunc("/ha/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "platform": "homeassistant"})
	})

	h.server = &http.Server{
		Addr:         h.webhookAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("homeassistant webhook server error", "error", err)
		}
	}()

	return nil
}

func (h *HomeAssistantAdapter) Disconnect(ctx context.Context) error {
	h.SetConnected(false)
	if h.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return h.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (h *HomeAssistantAdapter) Send(ctx context.Context, _ string, msg gateway.OutgoingMessage) error {
	if !h.IsConnected() {
		return fmt.Errorf("homeassistant: not connected")
	}

	notifyURL := h.haURL + "/api/services/notify/notify"
	payload := map[string]any{
		"message": msg.Text,
		"title":   "Hera",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("homeassistant: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, notifyURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("homeassistant: create notify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.token)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("homeassistant: send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("homeassistant: notify returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (h *HomeAssistantAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "Home Assistant",
		Type:     "private",
		Platform: "homeassistant",
	}, nil
}

func (h *HomeAssistantAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{}
}

type haWebhookPayload struct {
	Event   string `json:"event"`
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

func (h *HomeAssistantAdapter) handleWebhook(w http.ResponseWriter, r *http.Request) {
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

	var payload haWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if payload.Message == "" {
		http.Error(w, `{"error":"empty message"}`, http.StatusBadRequest)
		return
	}

	userID := payload.UserID
	if userID == "" {
		userID = "ha-user"
	}

	handler := h.Handler()
	if handler != nil {
		handler(r.Context(), gateway.IncomingMessage{
			Platform:  "homeassistant",
			ChatID:    "ha:" + userID,
			UserID:    userID,
			Username:  userID,
			Text:      payload.Message,
			Timestamp: time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
