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

	"github.com/sadewadee/hera/internal/gateway"
)

// APIServerAdapter exposes a REST API for programmatic chat access.
// Endpoints:
//   - POST /api/messages     — send a message, get a response
//   - GET  /api/conversations — list active sessions
//   - GET  /api/health       — health check
type APIServerAdapter struct {
	BaseAdapter
	addr   string
	server *http.Server
	mu     sync.Mutex
	// pendingResponses tracks request/response pairs for the message flow.
	pendingResponses map[string]chan string
}

// NewAPIServerAdapter creates an API server adapter listening on the given address.
func NewAPIServerAdapter(addr ...string) *APIServerAdapter {
	listenAddr := ":8080"
	if len(addr) > 0 && addr[0] != "" {
		listenAddr = addr[0]
	}
	return &APIServerAdapter{
		BaseAdapter:      BaseAdapter{AdapterName: "apiserver"},
		addr:             listenAddr,
		pendingResponses: make(map[string]chan string),
	}
}

// apiMessageRequest is the JSON body for POST /api/messages.
type apiMessageRequest struct {
	Text     string `json:"text"`
	UserID   string `json:"user_id,omitempty"`
	ChatID   string `json:"chat_id,omitempty"`
	Platform string `json:"platform,omitempty"`
}

// apiMessageResponse is the JSON response for POST /api/messages.
type apiMessageResponse struct {
	Response string `json:"response"`
	ChatID   string `json:"chat_id"`
	UserID   string `json:"user_id"`
}

func (a *APIServerAdapter) Connect(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/messages", a.handleMessages)
	mux.HandleFunc("/api/conversations", a.handleConversations)
	mux.HandleFunc("/api/health", a.handleHealth)

	a.server = &http.Server{
		Addr:         a.addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	a.SetConnected(true)
	slog.Info("API server starting", "addr", a.addr)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API server error", "error", err)
			a.SetConnected(false)
		}
	}()

	return nil
}

func (a *APIServerAdapter) Disconnect(ctx context.Context) error {
	a.SetConnected(false)
	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (a *APIServerAdapter) Send(_ context.Context, chatID string, msg gateway.OutgoingMessage) error {
	a.mu.Lock()
	ch, ok := a.pendingResponses[chatID]
	a.mu.Unlock()

	if ok {
		select {
		case ch <- msg.Text:
		default:
		}
	}
	return nil
}

func (a *APIServerAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "API Chat " + chatID,
		Type:     "private",
		Platform: "apiserver",
	}, nil
}

func (a *APIServerAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

func (a *APIServerAdapter) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req apiMessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}

	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = "api-user"
	}
	chatID := req.ChatID
	if chatID == "" {
		chatID = fmt.Sprintf("api-%s-%d", userID, time.Now().UnixNano())
	}

	// Create a response channel for this request.
	responseCh := make(chan string, 1)
	a.mu.Lock()
	a.pendingResponses[chatID] = responseCh
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.pendingResponses, chatID)
		a.mu.Unlock()
	}()

	// Dispatch message to the handler (agent).
	handler := a.Handler()
	if handler == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no message handler configured"})
		return
	}

	handler(r.Context(), gateway.IncomingMessage{
		Platform:  "apiserver",
		ChatID:    chatID,
		UserID:    userID,
		Username:  userID,
		Text:      req.Text,
		Timestamp: time.Now(),
	})

	// Wait for response with timeout.
	select {
	case response := <-responseCh:
		writeJSON(w, http.StatusOK, apiMessageResponse{
			Response: response,
			ChatID:   chatID,
			UserID:   userID,
		})
	case <-time.After(120 * time.Second):
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "response timeout"})
	case <-r.Context().Done():
		writeJSON(w, http.StatusRequestTimeout, map[string]string{"error": "request cancelled"})
	}
}

func (a *APIServerAdapter) handleConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	active := len(a.pendingResponses)
	a.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"active_requests": active,
		"platform":        "apiserver",
		"status":          "connected",
	})
}

func (a *APIServerAdapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"platform": "apiserver",
		"uptime":   "running",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
