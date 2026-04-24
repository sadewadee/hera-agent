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

func TestMatrixAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/client/v3/account/whoami" && r.Method == http.MethodGet {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@bot:example.com"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: server.URL,
		UserID:        "@bot:example.com",
		AccessToken:   "test-token",
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

func TestMatrixAdapter_Connect_EmptyURL(t *testing.T) {
	adapter := NewMatrixAdapter(MatrixConfig{})
	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with empty homeserver URL")
	}
}

func TestMatrixAdapter_Connect_BadToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: server.URL,
		UserID:        "@bot:example.com",
		AccessToken:   "bad-token",
	})

	err := adapter.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should fail with invalid token")
	}
	if !strings.Contains(err.Error(), "invalid access token") {
		t.Errorf("error = %q, want to contain 'invalid access token'", err.Error())
	}
}

func TestMatrixAdapter_Send(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotAuth string
	var gotPayload map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/client/v3/account/whoami" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@bot:example.com"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/_matrix/client/v3/rooms/") && strings.Contains(r.URL.Path, "/send/m.room.message/") {
			gotPath = r.URL.Path
			gotMethod = r.Method
			gotAuth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"event_id": "$123"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: server.URL,
		UserID:        "@bot:example.com",
		AccessToken:   "test-token",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "!room:example.com", gateway.OutgoingMessage{Text: "hello matrix"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("Send method = %q, want PUT", gotMethod)
	}
	if !strings.Contains(gotPath, "!room:example.com") {
		t.Errorf("Send path = %q, should contain room ID", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want 'Bearer test-token'", gotAuth)
	}
	if gotPayload["body"] != "hello matrix" {
		t.Errorf("payload body = %q, want 'hello matrix'", gotPayload["body"])
	}
	if gotPayload["msgtype"] != "m.text" {
		t.Errorf("payload msgtype = %q, want 'm.text'", gotPayload["msgtype"])
	}
}

func TestMatrixAdapter_Send_NotConnected(t *testing.T) {
	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: "http://localhost:1",
		AccessToken:   "token",
	})

	err := adapter.Send(context.Background(), "!room:example.com", gateway.OutgoingMessage{Text: "hello"})
	if err == nil {
		t.Fatal("Send() should fail when not connected")
	}
}

func TestMatrixAdapter_Send_HTMLFormat(t *testing.T) {
	var gotPayload map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/client/v3/account/whoami" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@bot:example.com"})
			return
		}
		if strings.Contains(r.URL.Path, "/send/m.room.message/") {
			json.NewDecoder(r.Body).Decode(&gotPayload)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"event_id": "$123"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: server.URL,
		UserID:        "@bot:example.com",
		AccessToken:   "test-token",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer adapter.Disconnect(context.Background())

	err := adapter.Send(context.Background(), "!room:example.com", gateway.OutgoingMessage{
		Text:   "<b>hello</b>",
		Format: "html",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if gotPayload["format"] != "org.matrix.custom.html" {
		t.Errorf("format = %q, want 'org.matrix.custom.html'", gotPayload["format"])
	}
}

func TestMatrixAdapter_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/client/v3/account/whoami" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user_id": "@bot:example.com"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: server.URL,
		UserID:        "@bot:example.com",
		AccessToken:   "test-token",
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if err := adapter.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if adapter.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect")
	}
}

func TestMatrixAdapter_GetChatInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/_matrix/client/v3/rooms/") && strings.HasSuffix(r.URL.Path, "/state/m.room.name") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"name": "Test Room"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewMatrixAdapter(MatrixConfig{
		HomeserverURL: server.URL,
		AccessToken:   "test-token",
	})

	info, err := adapter.GetChatInfo(context.Background(), "!room:example.com")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Title != "Test Room" {
		t.Errorf("Title = %q, want 'Test Room'", info.Title)
	}
	if info.Platform != "matrix" {
		t.Errorf("Platform = %q, want 'matrix'", info.Platform)
	}
}

func TestMatrixAdapter_SupportedMedia(t *testing.T) {
	adapter := NewMatrixAdapter(MatrixConfig{})
	media := adapter.SupportedMedia()
	if len(media) != 2 {
		t.Errorf("SupportedMedia() len = %d, want 2", len(media))
	}
}
