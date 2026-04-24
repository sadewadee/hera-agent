package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/mcp"
	"github.com/sadewadee/hera/internal/memory"
)

// Server implements the Agent Client Protocol (ACP) for IDE integration.
// It provides an HTTP API for editor extensions to interact with the Hera agent.
type Server struct {
	addr     string
	agent    *agent.Agent
	memory   *memory.Manager
	sessions *SessionStore
	auth     *AuthManager
	eventBus *mcp.EventBus
	httpSrv  *http.Server
	logger   *slog.Logger
}

// ServerConfig configures the ACP server.
type ServerConfig struct {
	Addr      string `json:"addr" yaml:"addr"`
	JWTSecret string `json:"jwt_secret" yaml:"jwt_secret"`
}

// NewServer creates a new ACP server.
func NewServer(cfg ServerConfig, ag *agent.Agent, mem *memory.Manager, eventBus ...*mcp.EventBus) *Server {
	if cfg.Addr == "" {
		cfg.Addr = ":9090"
	}
	srv := &Server{
		addr:     cfg.Addr,
		agent:    ag,
		memory:   mem,
		sessions: NewSessionStore(),
		auth:     NewAuthManager(cfg.JWTSecret),
		logger:   slog.Default(),
	}
	if len(eventBus) > 0 {
		srv.eventBus = eventBus[0]
	}
	return srv
}

// Start begins serving the ACP HTTP API.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Auth endpoints.
	mux.HandleFunc("/acp/auth/token", s.handleAuthToken)

	// Session endpoints.
	mux.HandleFunc("/acp/sessions", s.handleSessions)
	mux.HandleFunc("/acp/sessions/", s.handleSessionByID)

	// Message endpoints.
	mux.HandleFunc("/acp/messages", s.handleMessages)

	// Tool endpoints.
	mux.HandleFunc("/acp/tools", s.handleTools)

	// Event endpoints.
	mux.HandleFunc("/acp/events", s.handleEvents)

	// Health.
	mux.HandleFunc("/acp/health", s.handleHealth)

	s.httpSrv = &http.Server{
		Addr:         s.addr,
		Handler:      s.authMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	s.logger.Info("ACP server starting", "addr", s.addr)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("ACP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the ACP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for token endpoint and health.
		if r.URL.Path == "/acp/auth/token" || r.URL.Path == "/acp/health" {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		if token == "" {
			writeACPJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		clientID, valid := s.auth.ValidateToken(token)
		if !valid {
			writeACPJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		// Propagate the authenticated client ID via request header for
		// downstream handlers that may need it.
		r.Header.Set("X-ACP-Client-ID", clientID)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeACPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	token, err := s.auth.GenerateToken("acp-client")
	if err != nil {
		writeACPJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeACPJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_in": 3600,
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessions := s.sessions.List()
		writeACPJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
	case http.MethodPost:
		sess := s.sessions.Create()
		writeACPJSON(w, http.StatusCreated, sess)
	default:
		writeACPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/acp/sessions/"):]
	if id == "" {
		writeACPJSON(w, http.StatusBadRequest, map[string]string{"error": "session ID required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		sess, ok := s.sessions.Get(id)
		if !ok {
			writeACPJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		writeACPJSON(w, http.StatusOK, sess)
	case http.MethodDelete:
		s.sessions.Delete(id)
		writeACPJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeACPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeACPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		Text      string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeACPJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if req.Text == "" {
		writeACPJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}

	if s.agent == nil {
		writeACPJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent not initialized"})
		return
	}

	response, err := s.agent.HandleMessage(r.Context(), "acp", "acp-chat", "acp-user", req.Text)
	if err != nil {
		writeACPJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("agent: %v", err)})
		return
	}

	writeACPJSON(w, http.StatusOK, map[string]any{
		"response":   response,
		"session_id": req.SessionID,
	})
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeACPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Return available tools list.
	writeACPJSON(w, http.StatusOK, map[string]any{
		"tools": []string{"web_search", "web_scrape", "file_read", "file_write",
			"memory_save", "memory_search", "run_command", "skill_create",
			"datetime", "image_generate", "text_to_speech", "claude_code"},
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeACPJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// SSE endpoint for real-time events.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeACPJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	// Send initial heartbeat.
	fmt.Fprintf(w, "data: {\"type\":\"heartbeat\"}\n\n")
	flusher.Flush()

	// If no EventBus is wired, send heartbeat and close (backward compatible).
	if s.eventBus == nil {
		return
	}

	ch := s.eventBus.Subscribe()
	defer s.eventBus.Unsubscribe(ch)

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(map[string]any{
				"type": evt.Type,
				"data": evt.Data,
			})
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-heartbeat.C:
			fmt.Fprintf(w, "data: {\"type\":\"heartbeat\"}\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeACPJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "hera-acp",
	})
}

func writeACPJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// SessionStore manages ACP sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*ACPSession
}

// ACPSession represents an ACP session.
type ACPSession struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*ACPSession),
	}
}

// Create creates a new ACP session.
func (ss *SessionStore) Create() *ACPSession {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	id := fmt.Sprintf("acp-%d", time.Now().UnixNano())
	sess := &ACPSession{
		ID:        id,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ss.sessions[id] = sess
	return sess
}

// Get retrieves a session by ID.
func (ss *SessionStore) Get(id string) (*ACPSession, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	s, ok := ss.sessions[id]
	return s, ok
}

// List returns all sessions.
func (ss *SessionStore) List() []*ACPSession {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	result := make([]*ACPSession, 0, len(ss.sessions))
	for _, s := range ss.sessions {
		result = append(result, s)
	}
	return result
}

// Delete removes a session.
func (ss *SessionStore) Delete(id string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, id)
}
