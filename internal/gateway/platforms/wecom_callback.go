// Package platforms provides platform adapter implementations for the gateway.
//
// wecom_callback.go implements the WeCom callback-mode adapter for self-built
// enterprise applications. WeCom POSTs encrypted XML to an HTTP endpoint;
// the adapter decrypts, queues, and acknowledges. Replies are sent via the
// proactive message/send API.
package platforms

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	wecomDefaultHost = "0.0.0.0"
	wecomDefaultPort = 8645
	wecomDefaultPath = "/wecom/callback"
	wecomTokenTTL    = 7200
	wecomDedupTTL    = 300 * time.Second
)

// WecomCallbackAdapter handles WeCom callback-mode for enterprise applications.
type WecomCallbackAdapter struct {
	BaseAdapter
	host         string
	port         int
	path         string
	apps         []map[string]string
	server       *http.Server
	httpClient   *http.Client
	msgCh        chan *wecomCallbackEvent
	seen         *MessageDeduplicator
	userAppMap   map[string]string
	accessTokens map[string]*wecomToken
	tokenMu      sync.Mutex
}

type wecomCallbackEvent struct {
	ChatID    string
	UserID    string
	Content   string
	MessageID string
	Raw       string
}

type wecomToken struct {
	Token     string
	ExpiresAt time.Time
}

// WecomCallbackXML represents the XML structure of a WeCom callback message.
type WecomCallbackXML struct {
	XMLName      xml.Name `xml:"xml"`
	Encrypt      string   `xml:"Encrypt"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   string   `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        string   `xml:"MsgId"`
	Event        string   `xml:"Event"`
}

// NewWecomCallbackAdapter creates a new WeCom callback adapter.
func NewWecomCallbackAdapter(extra map[string]any) *WecomCallbackAdapter {
	host := wecomDefaultHost
	port := wecomDefaultPort
	path := wecomDefaultPath

	if h, ok := extra["host"].(string); ok && h != "" {
		host = h
	}
	if p, ok := extra["port"]; ok {
		switch v := p.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				port = n
			}
		}
	}
	if p, ok := extra["path"].(string); ok && p != "" {
		path = p
	}

	adapter := &WecomCallbackAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "wecom_callback"},
		host:        host,
		port:        port,
		path:        path,
		apps:        normalizeWecomApps(extra),
		httpClient:  &http.Client{Timeout: 20 * time.Second},
		msgCh:       make(chan *wecomCallbackEvent, 100),
		seen:        NewMessageDeduplicator(2000, wecomDedupTTL),
		userAppMap:  make(map[string]string),
		accessTokens: make(map[string]*wecomToken),
	}
	return adapter
}

func normalizeWecomApps(extra map[string]any) []map[string]string {
	if apps, ok := extra["apps"].([]any); ok && len(apps) > 0 {
		var result []map[string]string
		for _, a := range apps {
			if m, ok := a.(map[string]any); ok {
				app := make(map[string]string)
				for k, v := range m {
					app[k] = fmt.Sprintf("%v", v)
				}
				result = append(result, app)
			}
		}
		return result
	}
	if corpID, ok := extra["corp_id"].(string); ok && corpID != "" {
		return []map[string]string{{
			"name":             getStr(extra, "name", "default"),
			"corp_id":          corpID,
			"corp_secret":      getStr(extra, "corp_secret", ""),
			"agent_id":         getStr(extra, "agent_id", ""),
			"token":            getStr(extra, "token", ""),
			"encoding_aes_key": getStr(extra, "encoding_aes_key", ""),
		}}
	}
	return nil
}

// Connect starts the HTTP callback server.
func (a *WecomCallbackAdapter) Connect(ctx context.Context) error {
	if len(a.apps) == 0 {
		return fmt.Errorf("wecom_callback: no apps configured")
	}

	// Quick port-in-use check.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", a.port), time.Second)
	if err == nil {
		conn.Close()
		return fmt.Errorf("wecom_callback: port %d already in use", a.port)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc(a.path, a.handleCallback)

	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.host, a.port),
		Handler: mux,
	}

	go func() {
		slog.Info("WecomCallback HTTP server starting",
			"host", a.host,
			"port", a.port,
			"path", a.path,
		)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("WecomCallback server error", "error", err)
		}
	}()

	// Refresh initial tokens for all apps.
	for _, app := range a.apps {
		if err := a.refreshAccessToken(ctx, app); err != nil {
			slog.Warn("WecomCallback initial token refresh failed",
				"app", app["name"],
				"error", err,
			)
		}
	}

	a.SetConnected(true)
	return nil
}

// Disconnect shuts down the HTTP server.
func (a *WecomCallbackAdapter) Disconnect(ctx context.Context) error {
	a.SetConnected(false)
	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	}
	return nil
}

// Send sends a message via the WeCom proactive message API.
func (a *WecomCallbackAdapter) Send(ctx context.Context, chatID, content string) error {
	app := a.resolveAppForChat(chatID)
	toUser := chatID
	if idx := strings.Index(chatID, ":"); idx >= 0 {
		toUser = chatID[idx+1:]
	}

	token, err := a.getAccessToken(ctx, app)
	if err != nil {
		return fmt.Errorf("wecom_callback: token error: %w", err)
	}

	if len(content) > 2048 {
		content = content[:2048]
	}

	agentID, _ := strconv.Atoi(app["agent_id"])
	payload := map[string]any{
		"touser":  toUser,
		"msgtype": "text",
		"agentid": agentID,
		"text":    map[string]string{"content": content},
		"safe":    0,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wecom_callback: send failed: %s (code=%d)", result.ErrMsg, result.ErrCode)
	}
	return nil
}

func (a *WecomCallbackAdapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok","platform":"wecom_callback"}`))
}

func (a *WecomCallbackAdapter) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// URL verification handshake.
		a.handleVerify(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// POST: encrypted callback.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	msgSig := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	for _, app := range a.apps {
		crypt, err := NewWXBizMsgCrypt(
			app["token"],
			app["encoding_aes_key"],
			app["corp_id"],
		)
		if err != nil {
			continue
		}

		var envelope WecomCallbackXML
		if err := xml.Unmarshal(body, &envelope); err != nil {
			continue
		}

		plain, err := crypt.Decrypt(msgSig, timestamp, nonce, envelope.Encrypt)
		if err != nil {
			continue
		}

		event := a.buildEvent(app, string(plain))
		if event != nil {
			a.msgCh <- event
		}
		_, _ = w.Write([]byte("success"))
		return
	}
	http.Error(w, "invalid callback payload", http.StatusBadRequest)
}

func (a *WecomCallbackAdapter) handleVerify(w http.ResponseWriter, r *http.Request) {
	msgSig := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echoStr := r.URL.Query().Get("echostr")

	for _, app := range a.apps {
		crypt, err := NewWXBizMsgCrypt(
			app["token"],
			app["encoding_aes_key"],
			app["corp_id"],
		)
		if err != nil {
			continue
		}
		plain, err := crypt.VerifyURL(msgSig, timestamp, nonce, echoStr)
		if err != nil {
			continue
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(plain))
		return
	}
	http.Error(w, "signature verification failed", http.StatusForbidden)
}

func (a *WecomCallbackAdapter) buildEvent(app map[string]string, xmlText string) *wecomCallbackEvent {
	var msg WecomCallbackXML
	if err := xml.Unmarshal([]byte(xmlText), &msg); err != nil {
		return nil
	}
	msgType := strings.ToLower(msg.MsgType)
	if msgType == "event" {
		event := strings.ToLower(msg.Event)
		if event == "enter_agent" || event == "subscribe" {
			return nil
		}
	}
	if msgType != "text" && msgType != "event" {
		return nil
	}

	userID := msg.FromUserName
	corpID := msg.ToUserName
	if corpID == "" {
		corpID = app["corp_id"]
	}
	chatID := userAppKey(corpID, userID)
	content := strings.TrimSpace(msg.Content)
	if content == "" && msgType == "event" {
		content = "/start"
	}
	msgID := msg.MsgId
	if msgID == "" {
		msgID = fmt.Sprintf("%s:%s", userID, msg.CreateTime)
	}

	if a.seen.IsDuplicate(msgID) {
		return nil
	}

	a.userAppMap[chatID] = app["name"]

	return &wecomCallbackEvent{
		ChatID:    chatID,
		UserID:    userID,
		Content:   content,
		MessageID: msgID,
		Raw:       xmlText,
	}
}

func (a *WecomCallbackAdapter) resolveAppForChat(chatID string) map[string]string {
	appName := a.userAppMap[chatID]
	if appName == "" && !strings.Contains(chatID, ":") {
		for k, v := range a.userAppMap {
			if strings.HasSuffix(k, ":"+chatID) {
				appName = v
				break
			}
		}
	}
	if appName != "" {
		for _, app := range a.apps {
			if app["name"] == appName {
				return app
			}
		}
	}
	if len(a.apps) > 0 {
		return a.apps[0]
	}
	return map[string]string{}
}

func (a *WecomCallbackAdapter) getAccessToken(ctx context.Context, app map[string]string) (string, error) {
	a.tokenMu.Lock()
	cached := a.accessTokens[app["name"]]
	a.tokenMu.Unlock()

	if cached != nil && time.Now().Add(time.Minute).Before(cached.ExpiresAt) {
		return cached.Token, nil
	}
	return a.refreshAccessTokenStr(ctx, app)
}

func (a *WecomCallbackAdapter) refreshAccessToken(ctx context.Context, app map[string]string) error {
	_, err := a.refreshAccessTokenStr(ctx, app)
	return err
}

func (a *WecomCallbackAdapter) refreshAccessTokenStr(ctx context.Context, app map[string]string) (string, error) {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		app["corp_id"], app["corp_secret"])
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wecom token refresh failed: %s (code=%d)", result.ErrMsg, result.ErrCode)
	}

	expiresIn := result.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = wecomTokenTTL
	}

	a.tokenMu.Lock()
	a.accessTokens[app["name"]] = &wecomToken{
		Token:     result.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
	a.tokenMu.Unlock()

	slog.Info("WecomCallback token refreshed",
		"app", app["name"],
		"corp_id", app["corp_id"],
		"expires_in", expiresIn,
	)
	return result.AccessToken, nil
}

func userAppKey(corpID, userID string) string {
	if corpID != "" {
		return corpID + ":" + userID
	}
	return userID
}

func getStr(m map[string]any, key, fallback string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return fallback
}
