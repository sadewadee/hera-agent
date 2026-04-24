// Package platforms provides platform adapter implementations for the gateway.
//
// weixin.go implements the Weixin (WeChat) platform adapter using Tencent's
// iLink Bot API. It uses long-polling for inbound messages and supports
// text, image, voice, video, and file message types with AES-128-ECB CDN
// encryption for media files.
package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

const (
	ilinkBaseURL            = "https://ilinkai.weixin.qq.com"
	weixinCDNBaseURL        = "https://novac2c.cdn.weixin.qq.com/c2c"
	ilinkAppID              = "bot"
	channelVersion          = "2.2.0"
	ilinkAppClientVersion   = (2 << 16) | (2 << 8) | 0
	epGetUpdates            = "ilink/bot/getupdates"
	epSendMessage           = "ilink/bot/sendmessage"
	epSendTyping            = "ilink/bot/sendtyping"
	epGetConfig             = "ilink/bot/getconfig"
	epGetUploadURL          = "ilink/bot/getuploadurl"
	epGetBotQR              = "ilink/bot/get_bot_qrcode"
	epGetQRStatus           = "ilink/bot/get_qrcode_status"
	weixinLongPollTimeout   = 35 * time.Second
	weixinAPITimeout        = 15 * time.Second
	weixinMaxConsecFailures = 3
	weixinRetryDelay        = 2 * time.Second
	weixinBackoffDelay      = 30 * time.Second
	weixinSessionExpired    = -14
	weixinDedupTTL          = 300 * time.Second
	mediaImage              = 1
	mediaVideo              = 2
	mediaFile               = 3
	mediaVoice              = 4
	itemText                = 1
	itemImage               = 2
)

// WeixinAdapter implements the WeChat personal account adapter via iLink Bot API.
type WeixinAdapter struct {
	BaseAdapter
	botToken       string
	httpClient     *http.Client
	dedup          *MessageDeduplicator
	contextTokens  map[string]string
	contextMu      sync.Mutex
	running        bool
	runMu          sync.Mutex
	pollCancel     context.CancelFunc
	consecutiveFails int
}

// NewWeixinAdapter creates a new Weixin adapter with the given bot token.
func NewWeixinAdapter(botToken string) *WeixinAdapter {
	return &WeixinAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "weixin"},
		botToken:    botToken,
		httpClient: &http.Client{
			Timeout: weixinLongPollTimeout + 5*time.Second,
		},
		dedup:         NewMessageDeduplicator(2000, weixinDedupTTL),
		contextTokens: make(map[string]string),
	}
}

// Connect starts the long-polling loop for inbound messages.
func (a *WeixinAdapter) Connect(ctx context.Context) error {
	if a.botToken == "" {
		return fmt.Errorf("weixin: bot_token is required")
	}

	a.runMu.Lock()
	a.running = true
	a.runMu.Unlock()

	pollCtx, cancel := context.WithCancel(ctx)
	a.pollCancel = cancel

	go a.pollLoop(pollCtx)
	a.SetConnected(true)
	slog.Info("Weixin adapter connected")
	return nil
}

// Disconnect stops the polling loop.
func (a *WeixinAdapter) Disconnect(ctx context.Context) error {
	a.runMu.Lock()
	a.running = false
	a.runMu.Unlock()

	if a.pollCancel != nil {
		a.pollCancel()
	}
	a.SetConnected(false)
	slog.Info("Weixin adapter disconnected")
	return nil
}

// Send sends a text message to a Weixin user.
func (a *WeixinAdapter) Send(ctx context.Context, chatID, content string) error {
	a.contextMu.Lock()
	ctxToken := a.contextTokens[chatID]
	a.contextMu.Unlock()

	payload := map[string]any{
		"to_user_name":  chatID,
		"context_token": ctxToken,
		"content": []map[string]any{
			{
				"type": itemText,
				"text": content,
			},
		},
	}
	_, err := a.apiCall(ctx, epSendMessage, payload)
	return err
}

func (a *WeixinAdapter) pollLoop(ctx context.Context) {
	offset := ""
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		a.runMu.Lock()
		running := a.running
		a.runMu.Unlock()
		if !running {
			return
		}

		updates, newOffset, err := a.getUpdates(ctx, offset)
		if err != nil {
			a.consecutiveFails++
			if a.consecutiveFails >= weixinMaxConsecFailures {
				slog.Warn("Weixin consecutive failures, backing off",
					"failures", a.consecutiveFails,
				)
				select {
				case <-ctx.Done():
					return
				case <-time.After(weixinBackoffDelay):
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-time.After(weixinRetryDelay):
				}
			}
			continue
		}

		a.consecutiveFails = 0
		if newOffset != "" {
			offset = newOffset
		}

		for _, update := range updates {
			a.processUpdate(update)
		}
	}
}

func (a *WeixinAdapter) getUpdates(ctx context.Context, offset string) ([]map[string]any, string, error) {
	payload := map[string]any{
		"offset":  offset,
		"timeout": int(weixinLongPollTimeout.Milliseconds()),
	}
	result, err := a.apiCall(ctx, epGetUpdates, payload)
	if err != nil {
		return nil, "", err
	}

	errCode, _ := result["errcode"].(float64)
	if int(errCode) == weixinSessionExpired {
		return nil, "", fmt.Errorf("weixin: session expired")
	}

	var updates []map[string]any
	if list, ok := result["msg_list"].([]any); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				updates = append(updates, m)
			}
		}
	}
	newOffset, _ := result["next_key"].(string)
	return updates, newOffset, nil
}

func (a *WeixinAdapter) processUpdate(update map[string]any) {
	msgID := fmt.Sprintf("%v", update["msg_id"])
	if a.dedup.IsDuplicate(msgID) {
		return
	}

	fromUser, _ := update["from_user_name"].(string)
	content, _ := update["content"].(string)
	ctxToken, _ := update["context_token"].(string)

	if ctxToken != "" {
		a.contextMu.Lock()
		a.contextTokens[fromUser] = ctxToken
		a.contextMu.Unlock()
	}

	handler := a.Handler()
	if handler != nil {
		handler(context.Background(), gateway.IncomingMessage{
			Platform: "weixin",
			ChatID:   fromUser,
			Text:     content,
			ReplyTo:  msgID,
		})
	}
}

func (a *WeixinAdapter) apiCall(ctx context.Context, endpoint string, payload map[string]any) (map[string]any, error) {
	payload["bot_token"] = a.botToken
	payload["app_id"] = ilinkAppID

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("weixin: marshal error: %w", err)
	}

	url := fmt.Sprintf("%s/%s", ilinkBaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("weixin: request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weixin: HTTP error: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("weixin: decode error: %w", err)
	}
	return result, nil
}

// GetBotQRCode fetches a QR code for bot login.
func (a *WeixinAdapter) GetBotQRCode(ctx context.Context) (string, error) {
	result, err := a.apiCall(ctx, epGetBotQR, map[string]any{})
	if err != nil {
		return "", err
	}
	qrURL, _ := result["qr_url"].(string)
	return qrURL, nil
}

// GetQRCodeStatus checks the status of QR code login.
func (a *WeixinAdapter) GetQRCodeStatus(ctx context.Context, uuid string) (string, error) {
	result, err := a.apiCall(ctx, epGetQRStatus, map[string]any{"uuid": uuid})
	if err != nil {
		return "", err
	}
	status, _ := result["status"].(string)
	return status, nil
}
