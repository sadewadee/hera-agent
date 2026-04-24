package platforms

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// WeComAdapter integrates with WeCom (WeChat Work) via its REST API.
type WeComAdapter struct {
	BaseAdapter
	corpID       string
	corpSecret   string
	agentID      int
	callbackAddr string
	verifyToken  string
	accessToken  string
	tokenExpiry  time.Time
	tokenMu      sync.Mutex
	client       *http.Client
	server       *http.Server
	logger       *slog.Logger
}

// WeComConfig holds configuration for the WeCom adapter.
type WeComConfig struct {
	CorpID       string `json:"corp_id" yaml:"corp_id"`
	CorpSecret   string `json:"corp_secret" yaml:"corp_secret"`
	AgentID      int    `json:"agent_id" yaml:"agent_id"`
	VerifyToken  string `json:"verify_token" yaml:"verify_token"`
	CallbackAddr string `json:"callback_addr" yaml:"callback_addr"`
}

// NewWeComAdapter creates a WeCom adapter.
func NewWeComAdapter(cfg WeComConfig) *WeComAdapter {
	if cfg.CallbackAddr == "" {
		cfg.CallbackAddr = ":8090"
	}
	return &WeComAdapter{
		BaseAdapter:  BaseAdapter{AdapterName: "wecom"},
		corpID:       cfg.CorpID,
		corpSecret:   cfg.CorpSecret,
		agentID:      cfg.AgentID,
		verifyToken:  cfg.VerifyToken,
		callbackAddr: cfg.CallbackAddr,
		client:       &http.Client{Timeout: 30 * time.Second},
		logger:       slog.Default(),
	}
}

func (w *WeComAdapter) Connect(ctx context.Context) error {
	if w.corpID == "" || w.corpSecret == "" {
		return fmt.Errorf("wecom: corp ID and secret are required")
	}

	// Obtain initial access token.
	if err := w.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("wecom: get token: %w", err)
	}

	w.SetConnected(true)
	w.logger.Info("wecom adapter connected", "corp_id", w.corpID)

	mux := http.NewServeMux()
	mux.HandleFunc("/wecom/callback", w.handleCallback)
	mux.HandleFunc("/wecom/health", func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(map[string]string{"status": "ok", "platform": "wecom"})
	})

	w.server = &http.Server{
		Addr:         w.callbackAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.logger.Error("wecom callback server error", "error", err)
		}
	}()

	return nil
}

func (w *WeComAdapter) Disconnect(ctx context.Context) error {
	w.SetConnected(false)
	if w.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return w.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (w *WeComAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !w.IsConnected() {
		return fmt.Errorf("wecom: not connected")
	}

	token, err := w.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("wecom: get token for send: %w", err)
	}

	sendURL := "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + token
	payload := map[string]any{
		"touser":  chatID,
		"msgtype": "text",
		"agentid": w.agentID,
		"text": map[string]string{
			"content": msg.Text,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("wecom: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("wecom: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("wecom: send message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.ErrCode != 0 {
		return fmt.Errorf("wecom: error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

func (w *WeComAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "WeCom Chat",
		Type:     "private",
		Platform: "wecom",
	}, nil
}

func (w *WeComAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

func (w *WeComAdapter) refreshAccessToken(ctx context.Context) error {
	tokenURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		w.corpID, w.corpSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return err
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wecom token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	w.tokenMu.Lock()
	w.accessToken = result.AccessToken
	w.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	w.tokenMu.Unlock()

	return nil
}

func (w *WeComAdapter) getAccessToken(ctx context.Context) (string, error) {
	w.tokenMu.Lock()
	if time.Now().Before(w.tokenExpiry) {
		token := w.accessToken
		w.tokenMu.Unlock()
		return token, nil
	}
	w.tokenMu.Unlock()

	if err := w.refreshAccessToken(ctx); err != nil {
		return "", err
	}

	w.tokenMu.Lock()
	defer w.tokenMu.Unlock()
	return w.accessToken, nil
}

// wecomXMLMessage is the XML format WeCom uses for callback messages.
type wecomXMLMessage struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	FromUser   string   `xml:"FromUserName"`
	CreateTime int64    `xml:"CreateTime"`
	MsgType    string   `xml:"MsgType"`
	Content    string   `xml:"Content"`
	MsgID      string   `xml:"MsgId"`
	AgentID    int      `xml:"AgentID"`
}

func (w *WeComAdapter) handleCallback(rw http.ResponseWriter, r *http.Request) {
	// GET requests are URL verification (echostr).
	if r.Method == http.MethodGet {
		echoStr := r.URL.Query().Get("echostr")
		rw.WriteHeader(http.StatusOK)
		fmt.Fprint(rw, echoStr)
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

	var xmlMsg wecomXMLMessage
	if err := xml.Unmarshal(body, &xmlMsg); err != nil {
		http.Error(rw, `{"error":"invalid XML"}`, http.StatusBadRequest)
		return
	}

	if xmlMsg.Content == "" || xmlMsg.MsgType != "text" {
		rw.WriteHeader(http.StatusOK)
		return
	}

	handler := w.Handler()
	if handler != nil {
		handler(r.Context(), gateway.IncomingMessage{
			Platform:  "wecom",
			ChatID:    xmlMsg.FromUser,
			UserID:    xmlMsg.FromUser,
			Username:  xmlMsg.FromUser,
			Text:      xmlMsg.Content,
			Timestamp: time.Unix(xmlMsg.CreateTime, 0),
		})
	}

	rw.WriteHeader(http.StatusOK)
}
