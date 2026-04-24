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

func TestMattermostAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/users/me" && r.Method == http.MethodGet {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer mm-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"id":       "bot-user-id",
				"username": "hera-bot",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
		Channels:  []string{"ch1"},
	})

	err := adapter.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	if !adapter.IsConnected() {
		t.Error("IsConnected() should be true after Connect")
	}
	if adapter.botUserID != "bot-user-id" {
		t.Errorf("botUserID = %q, want 'bot-user-id'", adapter.botUserID)
	}
}

func TestMattermostAdapter_Connect_MissingCredentials(t *testing.T) {
	adapter := NewMattermostAdapter(MattermostConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with missing credentials")
	}
}

func TestMattermostAdapter_Connect_AuthFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "bad-token",
	})

	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with bad token")
	}
}

func TestMattermostAdapter_Send(t *testing.T) {
	var gotPayload map[string]string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/users/me" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"id": "bot-id", "username": "bot"})
			return
		}
		if r.URL.Path == "/api/v4/posts" && r.Method == http.MethodPost {
			gotAuth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"id": "post123"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "ch-123", gateway.OutgoingMessage{Text: "hello mattermost"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAuth != "Bearer mm-token" {
		t.Errorf("Authorization = %q, want 'Bearer mm-token'", gotAuth)
	}
	if gotPayload["channel_id"] != "ch-123" {
		t.Errorf("channel_id = %q, want 'ch-123'", gotPayload["channel_id"])
	}
	if gotPayload["message"] != "hello mattermost" {
		t.Errorf("message = %q, want 'hello mattermost'", gotPayload["message"])
	}
}

func TestMattermostAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: "http://localhost:1",
		Token:     "token",
	})

	err := adapter.Send(context.Background(), "ch1", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestMattermostAdapter_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/users/me" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"id": "bot-id", "username": "bot"})
			return
		}
		if r.URL.Path == "/api/v4/posts" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message":"forbidden"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "ch1", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail on 403 status")
	}
}

func TestMattermostAdapter_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/users/me" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"id": "bot-id", "username": "bot"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
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

func TestMattermostAdapter_GetChatInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v4/channels/") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"display_name": "Town Square",
				"type":         "O",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
	})

	info, err := adapter.GetChatInfo(context.Background(), "ch-123")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Title != "Town Square" {
		t.Errorf("Title = %q, want 'Town Square'", info.Title)
	}
	if info.Platform != "mattermost" {
		t.Errorf("Platform = %q, want 'mattermost'", info.Platform)
	}
}

func TestMattermostAdapter_GetChatInfo_DirectMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"display_name": "DM",
			"type":         "D",
		})
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
	})

	info, err := adapter.GetChatInfo(context.Background(), "dm-123")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Type != "private" {
		t.Errorf("Type = %q, want 'private'", info.Type)
	}
}

func TestMattermostAdapter_SupportedMedia(t *testing.T) {
	adapter := NewMattermostAdapter(MattermostConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}

func TestMattermostAdapter_FetchChannelPosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/users/me" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"id": "bot-id", "username": "bot"})
			return
		}
		if strings.Contains(r.URL.Path, "/api/v4/channels/ch1/posts") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"order": []string{"post1"},
				"posts": map[string]any{
					"post1": map[string]any{
						"id":         "post1",
						"channel_id": "ch1",
						"user_id":    "other-user",
						"message":    "hello from mattermost",
						"create_at":  1700000000001,
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMattermostAdapter(MattermostConfig{
		ServerURL: server.URL,
		Token:     "mm-token",
		Channels:  []string{"ch1"},
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	var receivedText string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
	})

	adapter.fetchChannelPosts(context.Background(), "ch1")

	if receivedText != "hello from mattermost" {
		t.Errorf("received text = %q, want 'hello from mattermost'", receivedText)
	}
}
