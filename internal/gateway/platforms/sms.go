package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

// SMSAdapter sends and receives SMS via the Twilio REST API.
type SMSAdapter struct {
	BaseAdapter
	accountSID  string
	authToken   string
	fromNumber  string
	webhookAddr string
	client      *http.Client
	server      *http.Server
	logger      *slog.Logger
}

// SMSConfig holds configuration for the Twilio SMS adapter.
type SMSConfig struct {
	AccountSID  string `json:"account_sid" yaml:"account_sid"`
	AuthToken   string `json:"auth_token" yaml:"auth_token"`
	FromNumber  string `json:"from_number" yaml:"from_number"`
	WebhookAddr string `json:"webhook_addr" yaml:"webhook_addr"`
}

// NewSMSAdapter creates an SMS adapter using the Twilio API.
func NewSMSAdapter(cfg SMSConfig) *SMSAdapter {
	if cfg.WebhookAddr == "" {
		cfg.WebhookAddr = ":8086"
	}
	return &SMSAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "sms"},
		accountSID:  cfg.AccountSID,
		authToken:   cfg.AuthToken,
		fromNumber:  cfg.FromNumber,
		webhookAddr: cfg.WebhookAddr,
		client:      &http.Client{Timeout: 30 * time.Second},
		logger:      slog.Default(),
	}
}

func (s *SMSAdapter) Connect(_ context.Context) error {
	if s.accountSID == "" || s.authToken == "" {
		return fmt.Errorf("sms: Twilio account SID and auth token are required")
	}

	s.SetConnected(true)
	s.logger.Info("sms adapter connected", "from", s.fromNumber)

	mux := http.NewServeMux()
	mux.HandleFunc("/sms/incoming", s.handleIncoming)
	mux.HandleFunc("/sms/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "platform": "sms"})
	})

	s.server = &http.Server{
		Addr:         s.webhookAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("sms webhook server error", "error", err)
		}
	}()

	return nil
}

func (s *SMSAdapter) Disconnect(ctx context.Context) error {
	s.SetConnected(false)
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (s *SMSAdapter) Send(ctx context.Context, chatID string, msg gateway.OutgoingMessage) error {
	if !s.IsConnected() {
		return fmt.Errorf("sms: not connected")
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.accountSID)

	form := url.Values{}
	form.Set("To", chatID)
	form.Set("From", s.fromNumber)
	form.Set("Body", msg.Text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("sms: create request: %w", err)
	}
	req.SetBasicAuth(s.accountSID, s.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sms: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("sms: Twilio returned %d: %s", resp.StatusCode, string(body))
	}

	s.logger.Info("sms sent", "to", chatID)
	return nil
}

func (s *SMSAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "SMS: " + chatID,
		Type:     "private",
		Platform: "sms",
	}, nil
}

func (s *SMSAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{}
}

// handleIncoming processes Twilio webhook callbacks for incoming SMS.
func (s *SMSAdapter) handleIncoming(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, `{"error":"parse form failed"}`, http.StatusBadRequest)
		return
	}

	from := r.FormValue("From")
	body := r.FormValue("Body")
	to := r.FormValue("To")

	if body == "" {
		http.Error(w, `{"error":"empty message"}`, http.StatusBadRequest)
		return
	}

	handler := s.Handler()
	if handler != nil {
		handler(r.Context(), gateway.IncomingMessage{
			Platform:  "sms",
			ChatID:    from,
			UserID:    from,
			Username:  from,
			Text:      body,
			Timestamp: time.Now(),
		})
	}

	_ = to // acknowledge
	// Twilio expects a TwiML response; empty 200 is acceptable.
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Response></Response>")
}
