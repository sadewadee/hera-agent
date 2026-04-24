package platforms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// DingTalkAdapter integrates with DingTalk via its Robot REST API.
// Outbound messages are sent to the robot webhook. Inbound messages
// are received through an HTTP callback server.
type DingTalkAdapter struct {
	BaseAdapter
	accessToken  string
	secret       string
	callbackAddr string
	client       *http.Client
	server       *http.Server
	logger       *slog.Logger
}

// DingTalkConfig holds configuration for the DingTalk adapter.
type DingTalkConfig struct {
	AccessToken  string `json:"access_token" yaml:"access_token"`
	Secret       string `json:"secret" yaml:"secret"`
	CallbackAddr string `json:"callback_addr" yaml:"callback_addr"`
}

// NewDingTalkAdapter creates a DingTalk adapter.
func NewDingTalkAdapter(cfg DingTalkConfig) *DingTalkAdapter {
	if cfg.CallbackAddr == "" {
		cfg.CallbackAddr = ":8088"
	}
	return &DingTalkAdapter{
		BaseAdapter:  BaseAdapter{AdapterName: "dingtalk"},
		accessToken:  cfg.AccessToken,
		secret:       cfg.Secret,
		callbackAddr: cfg.CallbackAddr,
		client:       &http.Client{Timeout: 30 * time.Second},
		logger:       slog.Default(),
	}
}

func (d *DingTalkAdapter) Connect(_ context.Context) error {
	if d.accessToken == "" {
		return fmt.Errorf("dingtalk: access token is required")
	}

	d.SetConnected(true)
	d.logger.Info("dingtalk adapter connected")

	mux := http.NewServeMux()
	mux.HandleFunc("/dingtalk/callback", d.handleCallback)
	mux.HandleFunc("/dingtalk/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "platform": "dingtalk"})
	})

	d.server = &http.Server{
		Addr:         d.callbackAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.Error("dingtalk callback server error", "error", err)
		}
	}()

	return nil
}

func (d *DingTalkAdapter) Disconnect(ctx context.Context) error {
	d.SetConnected(false)
	if d.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return d.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (d *DingTalkAdapter) Send(ctx context.Context, _ string, msg gateway.OutgoingMessage) error {
	if !d.IsConnected() {
		return fmt.Errorf("dingtalk: not connected")
	}

	sendURL := "https://oapi.dingtalk.com/robot/send?access_token=" + d.accessToken

	// Add HMAC-SHA256 signature if secret is configured.
	if d.secret != "" {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		stringToSign := ts + "\n" + d.secret
		mac := hmac.New(sha256.New, []byte(d.secret))
		mac.Write([]byte(stringToSign))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		sendURL += "&timestamp=" + ts + "&sign=" + url.QueryEscape(sign)
	}

	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": msg.Text,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("dingtalk: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("dingtalk: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("dingtalk: API returned %d: %s", resp.StatusCode, string(body))
	}

	// Check DingTalk-specific error in response body.
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.ErrCode != 0 {
		return fmt.Errorf("dingtalk: error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

func (d *DingTalkAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "DingTalk Chat",
		Type:     "group",
		Platform: "dingtalk",
	}, nil
}

func (d *DingTalkAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

type dingtalkCallbackPayload struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	SenderNick     string `json:"senderNick"`
	SenderID       string `json:"senderId"`
	ConversationID string `json:"conversationId"`
	CreateAt       int64  `json:"createAt"`
}

func (d *DingTalkAdapter) handleCallback(w http.ResponseWriter, r *http.Request) {
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

	var payload dingtalkCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	text := payload.Text.Content
	if text == "" {
		http.Error(w, `{"error":"empty message"}`, http.StatusBadRequest)
		return
	}

	chatID := payload.ConversationID
	if chatID == "" {
		chatID = "dingtalk:" + payload.SenderID
	}

	handler := d.Handler()
	if handler != nil {
		ts := time.Now()
		if payload.CreateAt > 0 {
			ts = time.UnixMilli(payload.CreateAt)
		}
		handler(r.Context(), gateway.IncomingMessage{
			Platform:  "dingtalk",
			ChatID:    chatID,
			UserID:    payload.SenderID,
			Username:  payload.SenderNick,
			Text:      text,
			Timestamp: ts,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
