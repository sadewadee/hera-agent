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

// MattermostAdapter connects to Mattermost via its REST API v4.
type MattermostAdapter struct {
	BaseAdapter
	serverURL string
	token     string
	client    *http.Client
	cancel    context.CancelFunc
	logger    *slog.Logger
	botUserID string
	channels  []string // channels to monitor
	lastTS    int64    // last seen post timestamp (ms)
}

// MattermostConfig holds configuration for the Mattermost adapter.
type MattermostConfig struct {
	ServerURL string   `json:"server_url" yaml:"server_url"`
	Token     string   `json:"token" yaml:"token"`
	Channels  []string `json:"channels" yaml:"channels"`
}

// NewMattermostAdapter creates a Mattermost adapter.
func NewMattermostAdapter(cfg MattermostConfig) *MattermostAdapter {
	return &MattermostAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "mattermost"},
		serverURL:   cfg.ServerURL,
		token:       cfg.Token,
		channels:    cfg.Channels,
		client:      &http.Client{Timeout: 30 * time.Second},
		logger:      slog.Default(),
	}
}

func (m *MattermostAdapter) Connect(ctx context.Context) error {
	if m.serverURL == "" || m.token == "" {
		return fmt.Errorf("mattermost: server URL and token are required")
	}

	// Verify connectivity and get bot user ID.
	meURL := m.serverURL + "/api/v4/users/me"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)
	if err != nil {
		return fmt.Errorf("mattermost: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("mattermost: connect to %s: %w", m.serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mattermost: auth failed with status %d", resp.StatusCode)
	}

	var me struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return fmt.Errorf("mattermost: decode user info: %w", err)
	}

	m.botUserID = me.ID
	m.lastTS = time.Now().UnixMilli()
	m.SetConnected(true)
	m.logger.Info("mattermost adapter connected", "server", m.serverURL, "user", me.Username)

	pollCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.pollPosts(pollCtx)

	return nil
}

func (m *MattermostAdapter) Disconnect(_ context.Context) error {
	m.SetConnected(false)
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

func (m *MattermostAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !m.IsConnected() {
		return fmt.Errorf("mattermost: not connected")
	}

	postURL := m.serverURL + "/api/v4/posts"
	payload := map[string]string{
		"channel_id": chatID,
		"message":    msg.Text,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mattermost: marshal post: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("mattermost: create post request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("mattermost: send post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("mattermost: post returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (m *MattermostAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	chURL := m.serverURL + "/api/v4/channels/" + chatID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chURL, nil)
	if err != nil {
		return nil, fmt.Errorf("mattermost: create channel request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mattermost: get channel: %w", err)
	}
	defer resp.Body.Close()

	title := "Mattermost Channel"
	chatType := "group"
	if resp.StatusCode == http.StatusOK {
		var ch struct {
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ch); err == nil {
			if ch.DisplayName != "" {
				title = ch.DisplayName
			}
			if ch.Type == "D" {
				chatType = "private"
			}
		}
	}

	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    title,
		Type:     chatType,
		Platform: "mattermost",
	}, nil
}

func (m *MattermostAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

func (m *MattermostAdapter) pollPosts(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, chID := range m.channels {
				m.fetchChannelPosts(ctx, chID)
			}
		}
	}
}

func (m *MattermostAdapter) fetchChannelPosts(ctx context.Context, channelID string) {
	postsURL := fmt.Sprintf("%s/api/v4/channels/%s/posts?since=%d", m.serverURL, channelID, m.lastTS)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, postsURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var postsResp struct {
		Order []string          `json:"order"`
		Posts map[string]mmPost `json:"posts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&postsResp); err != nil {
		return
	}

	handler := m.Handler()
	if handler == nil {
		return
	}

	for _, postID := range postsResp.Order {
		post, ok := postsResp.Posts[postID]
		if !ok || post.UserID == m.botUserID || post.Message == "" {
			continue
		}

		createTS := post.CreateAt
		if createTS > m.lastTS {
			m.lastTS = createTS
		}

		handler(ctx, gateway.IncomingMessage{
			Platform:  "mattermost",
			ChatID:    post.ChannelID,
			UserID:    post.UserID,
			Username:  post.UserID,
			Text:      post.Message,
			Timestamp: time.UnixMilli(createTS),
		})
	}
}

type mmPost struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	CreateAt  int64  `json:"create_at"`
}
