package acp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestACPServer creates an ACP Server with nil agent/memory (sufficient for
// testing auth, sessions, health, and tools endpoints).
func newTestACPServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	cfg := ServerConfig{
		Addr:      ":0",
		JWTSecret: "test-secret",
	}
	srv := NewServer(cfg, nil, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/acp/auth/token", srv.handleAuthToken)
	mux.HandleFunc("/acp/sessions", srv.handleSessions)
	mux.HandleFunc("/acp/sessions/", srv.handleSessionByID)
	mux.HandleFunc("/acp/messages", srv.handleMessages)
	mux.HandleFunc("/acp/tools", srv.handleTools)
	mux.HandleFunc("/acp/events", srv.handleEvents)
	mux.HandleFunc("/acp/health", srv.handleHealth)

	handler := srv.authMiddleware(mux)
	return srv, handler
}

// getToken obtains a valid auth token from the token endpoint.
func getToken(t *testing.T, handler http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/acp/auth/token", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /acp/auth/token status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatal("token endpoint did not return a valid token")
	}
	return token
}

func TestHealth(t *testing.T) {
	_, handler := newTestACPServer(t)

	req := httptest.NewRequest(http.MethodGet, "/acp/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /acp/health status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("health status = %v, want %q", body["status"], "ok")
	}
}

func TestAuthRequired(t *testing.T) {
	_, handler := newTestACPServer(t)

	// Request without Authorization header.
	req := httptest.NewRequest(http.MethodGet, "/acp/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /acp/sessions without auth status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	// Request with invalid token.
	req = httptest.NewRequest(http.MethodGet, "/acp/sessions", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /acp/sessions with invalid token status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthFlow(t *testing.T) {
	_, handler := newTestACPServer(t)

	// Step 1: Get a token.
	token := getToken(t, handler)

	// Step 2: Use the token for an authenticated request.
	req := httptest.NewRequest(http.MethodGet, "/acp/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /acp/sessions with valid token status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	sessions, ok := body["sessions"].([]any)
	if !ok {
		t.Fatalf("sessions is not an array: %T", body["sessions"])
	}
	// Initially empty.
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestSessions_Create(t *testing.T) {
	_, handler := newTestACPServer(t)
	token := getToken(t, handler)

	// Create a session.
	req := httptest.NewRequest(http.MethodPost, "/acp/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /acp/sessions status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var session map[string]any
	json.NewDecoder(rec.Body).Decode(&session)
	id, ok := session["id"].(string)
	if !ok || id == "" {
		t.Fatal("created session has no ID")
	}

	// Verify the session is listed.
	req = httptest.NewRequest(http.MethodGet, "/acp/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var listBody map[string]any
	json.NewDecoder(rec.Body).Decode(&listBody)
	sessions, ok := listBody["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Errorf("expected 1 session after create, got %v", sessions)
	}
}

func TestMessages_Send(t *testing.T) {
	_, handler := newTestACPServer(t)
	token := getToken(t, handler)

	// Send a message. Since agent is nil, we expect a 503 error.
	body := `{"text": "hello", "session_id": "test-session"}`
	req := httptest.NewRequest(http.MethodPost, "/acp/messages", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /acp/messages (nil agent) status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var respBody map[string]any
	json.NewDecoder(rec.Body).Decode(&respBody)
	errMsg, ok := respBody["error"].(string)
	if !ok || errMsg == "" {
		t.Error("expected error message in response")
	}
}
