package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// WebhookAdapter receives incoming webhook POST requests and routes them
// through the gateway message handler. Responses are sent via a configured
// callback URL or returned in the HTTP response.
type WebhookAdapter struct {
	BaseAdapter
	addr        string
	secret      string // optional webhook secret for verification
	callbackURL string // optional URL to POST responses to
	server      *http.Server
	mu          sync.Mutex
	pending     map[string]chan string
}

// WebhookConfig holds configuration for the webhook adapter.
type WebhookConfig struct {
	Addr        string `json:"addr" yaml:"addr"`
	Secret      string `json:"secret" yaml:"secret"`
	CallbackURL string `json:"callback_url" yaml:"callback_url"`
}

// NewWebhookAdapter creates a webhook adapter.
func NewWebhookAdapter(cfg ...WebhookConfig) *WebhookAdapter {
	var c WebhookConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Addr == "" {
		c.Addr = ":8081"
	}
	return &WebhookAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "webhook"},
		addr:        c.Addr,
		secret:      c.Secret,
		callbackURL: c.CallbackURL,
		pending:     make(map[string]chan string),
	}
}

// webhookPayload is the expected JSON structure for incoming webhooks.
type webhookPayload struct {
	Event   string `json:"event"`
	Text    string `json:"text"`
	UserID  string `json:"user_id"`
	ChatID  string `json:"chat_id"`
	ReplyTo string `json:"reply_to,omitempty"`
}

func (w *WebhookAdapter) Connect(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", w.handleWebhook)
	mux.HandleFunc("/webhook/health", w.handleHealth)

	w.server = &http.Server{
		Addr:         w.addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	w.SetConnected(true)
	slog.Info("Webhook adapter starting", "addr", w.addr)

	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Webhook server error", "error", err)
			w.SetConnected(false)
		}
	}()

	return nil
}

func (w *WebhookAdapter) Disconnect(ctx context.Context) error {
	w.SetConnected(false)
	if w.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return w.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (w *WebhookAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	// If we have a pending response channel, use it (synchronous webhook).
	w.mu.Lock()
	ch, ok := w.pending[chatID]
	w.mu.Unlock()

	if ok {
		select {
		case ch <- msg.Text:
			return nil
		default:
		}
	}

	// Otherwise, POST to callback URL if configured.
	if w.callbackURL != "" {
		return w.postCallback(ctx, chatID, msg)
	}

	return nil
}

func (w *WebhookAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "Webhook Chat",
		Type:     "private",
		Platform: "webhook",
	}, nil
}

func (w *WebhookAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

func (w *WebhookAdapter) handleWebhook(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Verify webhook secret if configured.
	if w.secret != "" {
		provided := r.Header.Get("X-Webhook-Secret")
		if provided != w.secret {
			http.Error(rw, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		writeWebhookJSON(rw, http.StatusBadRequest, map[string]string{"error": "read body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeWebhookJSON(rw, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if payload.Text == "" {
		writeWebhookJSON(rw, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	userID := payload.UserID
	if userID == "" {
		userID = "webhook-user"
	}
	chatID := payload.ChatID
	if chatID == "" {
		chatID = fmt.Sprintf("webhook-%s-%d", userID, time.Now().UnixNano())
	}

	// Create response channel.
	responseCh := make(chan string, 1)
	w.mu.Lock()
	w.pending[chatID] = responseCh
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.pending, chatID)
		w.mu.Unlock()
	}()

	handler := w.Handler()
	if handler == nil {
		writeWebhookJSON(rw, http.StatusServiceUnavailable, map[string]string{"error": "no handler"})
		return
	}

	handler(r.Context(), gateway.IncomingMessage{
		Platform:  "webhook",
		ChatID:    chatID,
		UserID:    userID,
		Username:  userID,
		Text:      payload.Text,
		Timestamp: time.Now(),
	})

	select {
	case response := <-responseCh:
		writeWebhookJSON(rw, http.StatusOK, map[string]any{
			"response": response,
			"chat_id":  chatID,
		})
	case <-time.After(120 * time.Second):
		writeWebhookJSON(rw, http.StatusGatewayTimeout, map[string]string{"error": "timeout"})
	case <-r.Context().Done():
		writeWebhookJSON(rw, http.StatusRequestTimeout, map[string]string{"error": "cancelled"})
	}
}

func (w *WebhookAdapter) handleHealth(rw http.ResponseWriter, r *http.Request) {
	writeWebhookJSON(rw, http.StatusOK, map[string]any{
		"status":   "ok",
		"platform": "webhook",
	})
}

func (w *WebhookAdapter) postCallback(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	payload := map[string]string{
		"chat_id":  chatID,
		"response": msg.Text,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal callback: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.callbackURL, io.NopCloser(
		io.LimitReader(jsonReader(data), 1024*1024)))
	if err != nil {
		return fmt.Errorf("create callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("callback request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}
	return nil
}

func writeWebhookJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonReader(data []byte) io.Reader {
	return io.NopCloser(io.LimitReader(nopReader(data), int64(len(data))))
}

type bytesReaderCloser struct {
	data []byte
	pos  int
}

func nopReader(data []byte) io.Reader {
	return &bytesReaderCloser{data: data}
}

func (r *bytesReaderCloser) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
