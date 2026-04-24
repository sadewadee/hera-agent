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

// BlueBubblesAdapter integrates with BlueBubbles (iMessage bridge) via its REST API.
type BlueBubblesAdapter struct {
	BaseAdapter
	apiURL   string
	password string
	client   *http.Client
	cancel   context.CancelFunc
	logger   *slog.Logger
	lastTS   int64 // last seen message timestamp (ms)
}

// BlueBubblesConfig holds configuration for the BlueBubbles adapter.
type BlueBubblesConfig struct {
	APIURL   string `json:"api_url" yaml:"api_url"`
	Password string `json:"password" yaml:"password"`
}

// NewBlueBubblesAdapter creates a BlueBubbles adapter.
func NewBlueBubblesAdapter(cfg BlueBubblesConfig) *BlueBubblesAdapter {
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:1234"
	}
	return &BlueBubblesAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "bluebubbles"},
		apiURL:      cfg.APIURL,
		password:    cfg.Password,
		client:      &http.Client{Timeout: 30 * time.Second},
		logger:      slog.Default(),
	}
}

func (b *BlueBubblesAdapter) Connect(ctx context.Context) error {
	if b.password == "" {
		return fmt.Errorf("bluebubbles: password is required")
	}

	// Verify connectivity via server info.
	infoURL := b.apiURL + "/api/v1/server?password=" + b.password
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return fmt.Errorf("bluebubbles: create info request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("bluebubbles: connect to %s: %w", b.apiURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bluebubbles: server returned status %d", resp.StatusCode)
	}

	b.lastTS = time.Now().UnixMilli()
	b.SetConnected(true)
	b.logger.Info("bluebubbles adapter connected", "url", b.apiURL)

	pollCtx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.pollMessages(pollCtx)

	return nil
}

func (b *BlueBubblesAdapter) Disconnect(_ context.Context) error {
	b.SetConnected(false)
	if b.cancel != nil {
		b.cancel()
	}
	return nil
}

func (b *BlueBubblesAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !b.IsConnected() {
		return fmt.Errorf("bluebubbles: not connected")
	}

	sendURL := b.apiURL + "/api/v1/message/text?password=" + b.password
	payload := map[string]string{
		"chatGuid": chatID,
		"message":  msg.Text,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("bluebubbles: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("bluebubbles: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("bluebubbles: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("bluebubbles: send returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (b *BlueBubblesAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	chatURL := b.apiURL + "/api/v1/chat/" + chatID + "?password=" + b.password
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chatURL, nil)
	if err != nil {
		return nil, fmt.Errorf("bluebubbles: create chat request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bluebubbles: get chat: %w", err)
	}
	defer resp.Body.Close()

	title := "iMessage Chat"
	chatType := "private"
	if resp.StatusCode == http.StatusOK {
		var chatResp struct {
			Data struct {
				DisplayName string `json:"displayName"`
				GroupName   string `json:"groupName"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err == nil {
			if chatResp.Data.GroupName != "" {
				title = chatResp.Data.GroupName
				chatType = "group"
			} else if chatResp.Data.DisplayName != "" {
				title = chatResp.Data.DisplayName
			}
		}
	}

	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    title,
		Type:     chatType,
		Platform: "bluebubbles",
	}, nil
}

func (b *BlueBubblesAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

func (b *BlueBubblesAdapter) pollMessages(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.fetchMessages(ctx)
		}
	}
}

type bbMessage struct {
	GUID        string `json:"guid"`
	Text        string `json:"text"`
	IsFromMe    bool   `json:"isFromMe"`
	DateCreated int64  `json:"dateCreated"`
	Handle      struct {
		Address string `json:"address"`
	} `json:"handle"`
	Chats []struct {
		GUID string `json:"guid"`
	} `json:"chats"`
}

func (b *BlueBubblesAdapter) fetchMessages(ctx context.Context) {
	msgsURL := b.apiURL + "/api/v1/message?password=" + b.password +
		"&after=" + strconv.FormatInt(b.lastTS, 10) +
		"&limit=50&sort=ASC"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, msgsURL, nil)
	if err != nil {
		return
	}

	resp, err := b.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []bbMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	handler := b.Handler()
	if handler == nil {
		return
	}

	for _, msg := range result.Data {
		if msg.IsFromMe || msg.Text == "" {
			continue
		}

		if msg.DateCreated > b.lastTS {
			b.lastTS = msg.DateCreated
		}

		chatID := ""
		if len(msg.Chats) > 0 {
			chatID = msg.Chats[0].GUID
		}

		handler(ctx, gateway.IncomingMessage{
			Platform:  "bluebubbles",
			ChatID:    chatID,
			UserID:    msg.Handle.Address,
			Username:  msg.Handle.Address,
			Text:      msg.Text,
			Timestamp: time.UnixMilli(msg.DateCreated),
		})
	}
}
