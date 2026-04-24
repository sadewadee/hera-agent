package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// SignalAdapter connects to Signal via the signal-cli REST API.
// See: https://github.com/bbernhard/signal-cli-rest-api
type SignalAdapter struct {
	BaseAdapter
	apiURL      string
	phoneNumber string
	client      *http.Client
	cancel      context.CancelFunc
	logger      *slog.Logger
}

// SignalConfig holds configuration for the Signal adapter.
type SignalConfig struct {
	APIURL      string `json:"api_url" yaml:"api_url"`
	PhoneNumber string `json:"phone_number" yaml:"phone_number"`
}

// NewSignalAdapter creates a Signal adapter using the signal-cli REST API.
func NewSignalAdapter(cfg SignalConfig) *SignalAdapter {
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}
	return &SignalAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "signal"},
		apiURL:      cfg.APIURL,
		phoneNumber: cfg.PhoneNumber,
		client:      &http.Client{Timeout: 30 * time.Second},
		logger:      slog.Default(),
	}
}

// signalReceived represents a received Signal message from the REST API.
type signalReceived struct {
	Envelope struct {
		Source      string `json:"source"`
		SourceName  string `json:"sourceName"`
		Timestamp   int64  `json:"timestamp"`
		DataMessage *struct {
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
			GroupInfo *struct {
				GroupID string `json:"groupId"`
			} `json:"groupInfo"`
		} `json:"dataMessage"`
	} `json:"envelope"`
}

func (s *SignalAdapter) Connect(ctx context.Context) error {
	// Verify connectivity by calling /v1/about.
	aboutURL := s.apiURL + "/v1/about"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, aboutURL, nil)
	if err != nil {
		return fmt.Errorf("signal: create about request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("signal: connect to %s: %w", s.apiURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("signal: API returned status %d", resp.StatusCode)
	}

	s.SetConnected(true)
	s.logger.Info("signal adapter connected", "api_url", s.apiURL, "number", s.phoneNumber)

	pollCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.pollMessages(pollCtx)

	return nil
}

func (s *SignalAdapter) Disconnect(_ context.Context) error {
	s.SetConnected(false)
	if s.cancel != nil {
		s.cancel()
	}
	s.logger.Info("signal adapter disconnected")
	return nil
}

func (s *SignalAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !s.IsConnected() {
		return fmt.Errorf("signal: not connected")
	}

	sendURL := s.apiURL + "/v2/send"
	payload := map[string]any{
		"message":    msg.Text,
		"number":     s.phoneNumber,
		"recipients": []string{chatID},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("signal: marshal send payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, byteReadCloser(data))
	if err != nil {
		return fmt.Errorf("signal: create send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("signal: send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("signal: send returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *SignalAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "Signal Chat " + chatID,
		Type:     "private",
		Platform: "signal",
	}, nil
}

func (s *SignalAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

// pollMessages polls the signal-cli REST API for incoming messages.
func (s *SignalAdapter) pollMessages(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.fetchMessages(ctx)
		}
	}
}

func (s *SignalAdapter) fetchMessages(ctx context.Context) {
	receiveURL := s.apiURL + "/v1/receive/" + url.PathEscape(s.phoneNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, receiveURL, nil)
	if err != nil {
		s.logger.Warn("signal: create receive request failed", "error", err)
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("signal: receive request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return
	}

	var messages []signalReceived
	if err := json.Unmarshal(body, &messages); err != nil {
		return
	}

	handler := s.Handler()
	if handler == nil {
		return
	}

	for _, m := range messages {
		if m.Envelope.DataMessage == nil || m.Envelope.DataMessage.Message == "" {
			continue
		}

		chatID := m.Envelope.Source
		if m.Envelope.DataMessage.GroupInfo != nil {
			chatID = m.Envelope.DataMessage.GroupInfo.GroupID
		}

		handler(ctx, gateway.IncomingMessage{
			Platform:  "signal",
			ChatID:    chatID,
			UserID:    m.Envelope.Source,
			Username:  m.Envelope.SourceName,
			Text:      m.Envelope.DataMessage.Message,
			Timestamp: time.UnixMilli(m.Envelope.Timestamp),
		})
	}
}

// byteReadCloser wraps a byte slice as an io.ReadCloser.
func byteReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(&bytesBuffer{data: data})
}

type bytesBuffer struct {
	data []byte
	pos  int
}

func (b *bytesBuffer) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}
