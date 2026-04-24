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

// FeishuAdapter integrates with Feishu (Lark) via its Open Platform REST API.
type FeishuAdapter struct {
	BaseAdapter
	appID             string
	appSecret         string
	verificationToken string
	callbackAddr      string
	tenantToken       string
	tokenExpiry       time.Time
	tokenMu           sync.Mutex
	client            *http.Client
	server            *http.Server
	logger            *slog.Logger
}

// FeishuConfig holds configuration for the Feishu adapter.
type FeishuConfig struct {
	AppID             string `json:"app_id" yaml:"app_id"`
	AppSecret         string `json:"app_secret" yaml:"app_secret"`
	VerificationToken string `json:"verification_token" yaml:"verification_token"`
	CallbackAddr      string `json:"callback_addr" yaml:"callback_addr"`
}

// NewFeishuAdapter creates a Feishu adapter.
func NewFeishuAdapter(cfg FeishuConfig) *FeishuAdapter {
	if cfg.CallbackAddr == "" {
		cfg.CallbackAddr = ":8089"
	}
	return &FeishuAdapter{
		BaseAdapter:       BaseAdapter{AdapterName: "feishu"},
		appID:             cfg.AppID,
		appSecret:         cfg.AppSecret,
		verificationToken: cfg.VerificationToken,
		callbackAddr:      cfg.CallbackAddr,
		client:            &http.Client{Timeout: 30 * time.Second},
		logger:            slog.Default(),
	}
}

func (f *FeishuAdapter) Connect(ctx context.Context) error {
	if f.appID == "" || f.appSecret == "" {
		return fmt.Errorf("feishu: app ID and secret are required")
	}

	// Obtain initial tenant access token.
	if err := f.refreshToken(ctx); err != nil {
		return fmt.Errorf("feishu: get token: %w", err)
	}

	f.SetConnected(true)
	f.logger.Info("feishu adapter connected", "app_id", f.appID)

	mux := http.NewServeMux()
	mux.HandleFunc("/feishu/event", f.handleEvent)
	mux.HandleFunc("/feishu/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "platform": "feishu"})
	})

	f.server = &http.Server{
		Addr:         f.callbackAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := f.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			f.logger.Error("feishu callback server error", "error", err)
		}
	}()

	return nil
}

func (f *FeishuAdapter) Disconnect(ctx context.Context) error {
	f.SetConnected(false)
	if f.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return f.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (f *FeishuAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !f.IsConnected() {
		return fmt.Errorf("feishu: not connected")
	}

	token, err := f.getToken(ctx)
	if err != nil {
		return fmt.Errorf("feishu: get token for send: %w", err)
	}

	sendURL := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
	content, _ := json.Marshal(map[string]string{"text": msg.Text})
	payload := map[string]string{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    string(content),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("feishu: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("feishu: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("feishu: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("feishu: send returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (f *FeishuAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "Feishu Chat",
		Type:     "group",
		Platform: "feishu",
	}, nil
}

func (f *FeishuAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

func (f *FeishuAdapter) refreshToken(ctx context.Context) error {
	tokenURL := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	payload := map[string]string{
		"app_id":     f.appID,
		"app_secret": f.appSecret,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, byteReadCloser(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("feishu token error %d: %s", result.Code, result.Msg)
	}

	f.tokenMu.Lock()
	f.tenantToken = result.TenantAccessToken
	f.tokenExpiry = time.Now().Add(time.Duration(result.Expire-60) * time.Second)
	f.tokenMu.Unlock()

	return nil
}

func (f *FeishuAdapter) getToken(ctx context.Context) (string, error) {
	f.tokenMu.Lock()
	if time.Now().Before(f.tokenExpiry) {
		token := f.tenantToken
		f.tokenMu.Unlock()
		return token, nil
	}
	f.tokenMu.Unlock()

	if err := f.refreshToken(ctx); err != nil {
		return "", err
	}

	f.tokenMu.Lock()
	defer f.tokenMu.Unlock()
	return f.tenantToken, nil
}

type feishuEventPayload struct {
	Challenge string `json:"challenge"` // URL verification
	Type      string `json:"type"`
	Event     struct {
		Message struct {
			ChatID      string `json:"chat_id"`
			MessageType string `json:"message_type"`
			Content     string `json:"content"`
		} `json:"message"`
		Sender struct {
			SenderID struct {
				OpenID string `json:"open_id"`
			} `json:"sender_id"`
		} `json:"sender"`
	} `json:"event"`
	Header struct {
		EventType string `json:"event_type"`
	} `json:"header"`
}

func (f *FeishuAdapter) handleEvent(w http.ResponseWriter, r *http.Request) {
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

	var payload feishuEventPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// Handle URL verification challenge.
	if payload.Challenge != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"challenge": payload.Challenge})
		return
	}

	// Extract message text from content JSON.
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(payload.Event.Message.Content), &content); err != nil || content.Text == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	handler := f.Handler()
	if handler != nil {
		handler(r.Context(), gateway.IncomingMessage{
			Platform:  "feishu",
			ChatID:    payload.Event.Message.ChatID,
			UserID:    payload.Event.Sender.SenderID.OpenID,
			Username:  payload.Event.Sender.SenderID.OpenID,
			Text:      content.Text,
			Timestamp: time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
