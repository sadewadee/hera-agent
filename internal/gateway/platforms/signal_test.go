package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestSignalAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/about" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"versions": []string{"0.1.0"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+1234567890",
	})

	err := adapter.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	if !adapter.IsConnected() {
		t.Error("IsConnected() should be true after Connect")
	}
}

func TestSignalAdapter_Connect_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+1234567890",
	})

	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail on non-200 status")
	}
}

func TestSignalAdapter_Send(t *testing.T) {
	var gotPath string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/about" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"versions": []string{"0.1.0"}})
			return
		}
		if r.URL.Path == "/v2/send" && r.Method == http.MethodPost {
			gotPath = r.URL.Path
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+1234567890",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "+0987654321", gateway.OutgoingMessage{Text: "hello signal"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotPath != "/v2/send" {
		t.Errorf("Send path = %q, want /v2/send", gotPath)
	}
	if gotPayload["message"] != "hello signal" {
		t.Errorf("payload message = %v, want 'hello signal'", gotPayload["message"])
	}
	if gotPayload["number"] != "+1234567890" {
		t.Errorf("payload number = %v, want '+1234567890'", gotPayload["number"])
	}
}

func TestSignalAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      "http://localhost:1",
		PhoneNumber: "+1234567890",
	})

	err := adapter.Send(context.Background(), "+0987654321", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestSignalAdapter_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/about" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"versions": []string{"0.1.0"}})
			return
		}
		if r.URL.Path == "/v2/send" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"bad request"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+1234567890",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "+0987654321", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail on 400 status")
	}
}

func TestSignalAdapter_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/about" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"versions": []string{"0.1.0"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+1234567890",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err := adapter.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if adapter.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect")
	}
}

func TestSignalAdapter_GetChatInfo(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      "http://localhost:1",
		PhoneNumber: "+1234567890",
	})

	info, err := adapter.GetChatInfo(context.Background(), "+0987654321")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.ID != "+0987654321" {
		t.Errorf("ID = %q, want '+0987654321'", info.ID)
	}
	if info.Platform != "signal" {
		t.Errorf("Platform = %q, want 'signal'", info.Platform)
	}
}

func TestSignalAdapter_SupportedMedia(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}

func TestSignalAdapter_FetchMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/about" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"versions": []string{"0.1.0"}})
			return
		}
		if r.URL.Path == "/v1/receive/+1234567890" {
			w.WriteHeader(http.StatusOK)
			msgs := []signalReceived{
				{
					Envelope: struct {
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
					}{
						Source:     "+5555555555",
						SourceName: "Test User",
						Timestamp:  time.Now().UnixMilli(),
						DataMessage: &struct {
							Message   string `json:"message"`
							Timestamp int64  `json:"timestamp"`
							GroupInfo *struct {
								GroupID string `json:"groupId"`
							} `json:"groupInfo"`
						}{
							Message:   "hello from signal",
							Timestamp: time.Now().UnixMilli(),
						},
					},
				},
			}
			json.NewEncoder(w).Encode(msgs)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewSignalAdapter(SignalConfig{
		APIURL:      server.URL,
		PhoneNumber: "+1234567890",
	})

	var received gateway.IncomingMessage
	var mu sync.Mutex
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		mu.Lock()
		received = msg
		mu.Unlock()
	})

	// Call fetchMessages directly to test the polling logic.
	adapter.SetConnected(true)
	adapter.fetchMessages(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received.Text != "hello from signal" {
		t.Errorf("received text = %q, want 'hello from signal'", received.Text)
	}
	if received.Platform != "signal" {
		t.Errorf("received platform = %q, want 'signal'", received.Platform)
	}
	if received.UserID != "+5555555555" {
		t.Errorf("received UserID = %q, want '+5555555555'", received.UserID)
	}
}

func TestSignalAdapter_DefaultURL(t *testing.T) {
	adapter := NewSignalAdapter(SignalConfig{PhoneNumber: "+1234567890"})
	if adapter.apiURL != "http://localhost:8080" {
		t.Errorf("default apiURL = %q, want 'http://localhost:8080'", adapter.apiURL)
	}
}
