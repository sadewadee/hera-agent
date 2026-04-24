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

	"github.com/google/uuid"
	"github.com/sadewadee/hera/internal/gateway"
)

// MatrixAdapter connects to a Matrix homeserver via the Client-Server REST API.
type MatrixAdapter struct {
	BaseAdapter
	homeserverURL string
	userID        string
	accessToken   string
	client        *http.Client
	cancel        context.CancelFunc
	logger        *slog.Logger
	syncToken     string
	tokenMu       sync.Mutex
}

// MatrixConfig holds configuration for the Matrix adapter.
type MatrixConfig struct {
	HomeserverURL string `json:"homeserver_url" yaml:"homeserver_url"`
	UserID        string `json:"user_id" yaml:"user_id"`
	AccessToken   string `json:"access_token" yaml:"access_token"`
}

// NewMatrixAdapter creates a Matrix adapter using the Client-Server REST API.
func NewMatrixAdapter(cfg MatrixConfig) *MatrixAdapter {
	return &MatrixAdapter{
		BaseAdapter:   BaseAdapter{AdapterName: "matrix"},
		homeserverURL: cfg.HomeserverURL,
		userID:        cfg.UserID,
		accessToken:   cfg.AccessToken,
		client:        &http.Client{Timeout: 60 * time.Second},
		logger:        slog.Default(),
	}
}

func (m *MatrixAdapter) Connect(ctx context.Context) error {
	if m.homeserverURL == "" {
		return fmt.Errorf("matrix: homeserver URL is required")
	}

	// Verify connectivity with a whoami request.
	whoamiURL := m.homeserverURL + "/_matrix/client/v3/account/whoami"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, whoamiURL, nil)
	if err != nil {
		return fmt.Errorf("matrix: create whoami request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.accessToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("matrix: connect to %s: %w", m.homeserverURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("matrix: invalid access token")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("matrix: whoami returned status %d", resp.StatusCode)
	}

	m.SetConnected(true)
	m.logger.Info("matrix adapter connected", "homeserver", m.homeserverURL, "user", m.userID)

	syncCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.syncLoop(syncCtx)

	return nil
}

func (m *MatrixAdapter) Disconnect(_ context.Context) error {
	m.SetConnected(false)
	if m.cancel != nil {
		m.cancel()
	}
	m.logger.Info("matrix adapter disconnected")
	return nil
}

func (m *MatrixAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !m.IsConnected() {
		return fmt.Errorf("matrix: not connected")
	}

	txnID := uuid.New().String()
	sendURL := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		m.homeserverURL, chatID, txnID)

	content := map[string]string{
		"msgtype": "m.text",
		"body":    msg.Text,
	}
	if msg.Format == "html" {
		content["format"] = "org.matrix.custom.html"
		content["formatted_body"] = msg.Text
	}

	data, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("matrix: marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("matrix: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.accessToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("matrix: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("matrix: send returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (m *MatrixAdapter) GetChatInfo(ctx context.Context, chatID string) (*gateway.ChatInfo, error) {
	nameURL := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/state/m.room.name", m.homeserverURL, chatID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nameURL, nil)
	if err != nil {
		return nil, fmt.Errorf("matrix: create room name request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.accessToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("matrix: get room name: %w", err)
	}
	defer resp.Body.Close()

	title := "Matrix Room"
	if resp.StatusCode == http.StatusOK {
		var nameResp struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&nameResp); err == nil && nameResp.Name != "" {
			title = nameResp.Name
		}
	}

	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    title,
		Type:     "group",
		Platform: "matrix",
	}, nil
}

func (m *MatrixAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

// matrixSyncResponse is a simplified /sync response.
type matrixSyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Join map[string]struct {
			Timeline struct {
				Events []matrixEvent `json:"events"`
			} `json:"timeline"`
		} `json:"join"`
	} `json:"rooms"`
}

type matrixEvent struct {
	Type     string          `json:"type"`
	Sender   string          `json:"sender"`
	Content  json.RawMessage `json:"content"`
	EventID  string          `json:"event_id"`
	OriginTS int64           `json:"origin_server_ts"`
}

type matrixMessageContent struct {
	MsgType string `json:"msgtype"`
	Body    string `json:"body"`
}

func (m *MatrixAdapter) syncLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m.tokenMu.Lock()
		since := m.syncToken
		m.tokenMu.Unlock()

		syncURL := fmt.Sprintf("%s/_matrix/client/v3/sync?timeout=30000", m.homeserverURL)
		if since != "" {
			syncURL += "&since=" + since
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncURL, nil)
		if err != nil {
			m.logger.Warn("matrix: create sync request failed", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+m.accessToken)

		resp, err := m.client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			m.logger.Warn("matrix: sync request failed", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK || err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var syncResp matrixSyncResponse
		if err := json.Unmarshal(body, &syncResp); err != nil {
			continue
		}

		m.tokenMu.Lock()
		m.syncToken = syncResp.NextBatch
		m.tokenMu.Unlock()

		// Skip initial sync to avoid replaying old messages.
		if since == "" {
			continue
		}

		m.processSync(ctx, syncResp)
	}
}

func (m *MatrixAdapter) processSync(ctx context.Context, syncResp matrixSyncResponse) {
	handler := m.Handler()
	if handler == nil {
		return
	}

	for roomID, room := range syncResp.Rooms.Join {
		for _, event := range room.Timeline.Events {
			if event.Type != "m.room.message" || event.Sender == m.userID {
				continue
			}

			var content matrixMessageContent
			if err := json.Unmarshal(event.Content, &content); err != nil || content.Body == "" {
				continue
			}

			handler(ctx, gateway.IncomingMessage{
				Platform:  "matrix",
				ChatID:    roomID,
				UserID:    event.Sender,
				Username:  event.Sender,
				Text:      content.Body,
				Timestamp: time.UnixMilli(event.OriginTS),
			})
		}
	}
}
