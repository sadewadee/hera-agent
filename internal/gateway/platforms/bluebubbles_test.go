package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestBlueBubblesAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/server" && r.URL.Query().Get("password") == "test-pass" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
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

func TestBlueBubblesAdapter_Connect_MissingPassword(t *testing.T) {
	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with missing password")
	}
}

func TestBlueBubblesAdapter_Connect_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
	})

	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail on server error")
	}
}

func TestBlueBubblesAdapter_Send(t *testing.T) {
	var gotPath string
	var gotPayload map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/server" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		if r.URL.Path == "/api/v1/message/text" && r.Method == http.MethodPost {
			gotPath = r.URL.Path
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "iMessage;-;+1234567890", gateway.OutgoingMessage{Text: "hello imessage"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotPath != "/api/v1/message/text" {
		t.Errorf("Send path = %q, want '/api/v1/message/text'", gotPath)
	}
	if gotPayload["chatGuid"] != "iMessage;-;+1234567890" {
		t.Errorf("chatGuid = %q, want 'iMessage;-;+1234567890'", gotPayload["chatGuid"])
	}
	if gotPayload["message"] != "hello imessage" {
		t.Errorf("message = %q, want 'hello imessage'", gotPayload["message"])
	}
}

func TestBlueBubblesAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   "http://localhost:1",
		Password: "pass",
	})

	err := adapter.Send(context.Background(), "chat1", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestBlueBubblesAdapter_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/server" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
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

func TestBlueBubblesAdapter_GetChatInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/v1/chat/") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{
					"displayName": "John Doe",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
	})

	info, err := adapter.GetChatInfo(context.Background(), "iMessage;-;+1234567890")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Title != "John Doe" {
		t.Errorf("Title = %q, want 'John Doe'", info.Title)
	}
	if info.Platform != "bluebubbles" {
		t.Errorf("Platform = %q, want 'bluebubbles'", info.Platform)
	}
}

func TestBlueBubblesAdapter_GetChatInfo_GroupChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{
				"groupName": "Family Group",
			},
		})
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
	})

	info, err := adapter.GetChatInfo(context.Background(), "chat-guid")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Title != "Family Group" {
		t.Errorf("Title = %q, want 'Family Group'", info.Title)
	}
	if info.Type != "group" {
		t.Errorf("Type = %q, want 'group'", info.Type)
	}
}

func TestBlueBubblesAdapter_SupportedMedia(t *testing.T) {
	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}

func TestBlueBubblesAdapter_FetchMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/server" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		if r.URL.Path == "/api/v1/message" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"guid":        "msg1",
						"text":        "hello from imessage",
						"isFromMe":    false,
						"dateCreated": 1700000000001,
						"handle": map[string]string{
							"address": "+5555555555",
						},
						"chats": []map[string]string{
							{"guid": "iMessage;-;+5555555555"},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{
		APIURL:   server.URL,
		Password: "test-pass",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	var receivedText string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
	})

	adapter.fetchMessages(context.Background())

	if receivedText != "hello from imessage" {
		t.Errorf("received text = %q, want 'hello from imessage'", receivedText)
	}
}

func TestBlueBubblesAdapter_DefaultURL(t *testing.T) {
	adapter := NewBlueBubblesAdapter(BlueBubblesConfig{Password: "pass"})
	if adapter.apiURL != "http://localhost:1234" {
		t.Errorf("default apiURL = %q, want 'http://localhost:1234'", adapter.apiURL)
	}
}
