// Package acp provides the Agent Client Protocol implementation.
//
// session.go implements the ACP session manager that maps ACP sessions to
// Hera agent instances. Sessions are held in-memory for fast access and
// can be listed, forked, and cleaned up. The session manager provides
// the core session lifecycle for the ACP server.
package acp

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// SessionState tracks per-session state for an ACP-managed agent.
type SessionState struct {
	SessionID string         `json:"session_id"`
	CWD       string         `json:"cwd"`
	Model     string         `json:"model"`
	History   []HistoryEntry `json:"history,omitempty"`
}

// HistoryEntry represents a single message in the conversation history.
type HistoryEntry struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// SessionManager manages ACP sessions. Thread-safe.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]*SessionState
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionState),
	}
}

// CreateSession creates a new session with a unique ID.
func (m *SessionManager) CreateSession(cwd, model string) *SessionState {
	if cwd == "" {
		cwd = "."
	}

	state := &SessionState{
		SessionID: uuid.New().String(),
		CWD:       cwd,
		Model:     model,
	}

	m.mu.Lock()
	m.sessions[state.SessionID] = state
	m.mu.Unlock()

	slog.Info("created ACP session",
		"session_id", state.SessionID,
		"cwd", cwd,
		"model", model,
	)
	return state
}

// GetSession returns the session for the given ID, or nil if not found.
func (m *SessionManager) GetSession(sessionID string) *SessionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[sessionID]
}

// RemoveSession removes a session. Returns true if it existed.
func (m *SessionManager) RemoveSession(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, existed := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	if existed {
		slog.Info("removed ACP session", "session_id", sessionID)
	}
	return existed
}

// ForkSession creates a deep copy of a session's history into a new session.
func (m *SessionManager) ForkSession(sessionID, cwd string) *SessionState {
	m.mu.Lock()
	original, ok := m.sessions[sessionID]
	m.mu.Unlock()

	if !ok {
		return nil
	}

	if cwd == "" {
		cwd = original.CWD
	}

	// Deep-copy history.
	history := make([]HistoryEntry, len(original.History))
	copy(history, original.History)

	newState := &SessionState{
		SessionID: uuid.New().String(),
		CWD:       cwd,
		Model:     original.Model,
		History:   history,
	}

	m.mu.Lock()
	m.sessions[newState.SessionID] = newState
	m.mu.Unlock()

	slog.Info("forked ACP session",
		"from", sessionID,
		"to", newState.SessionID,
	)
	return newState
}

// ListSessions returns lightweight info dicts for all sessions.
func (m *SessionManager) ListSessions() []SessionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, SessionInfo{
			SessionID:  s.SessionID,
			CWD:        s.CWD,
			Model:      s.Model,
			HistoryLen: len(s.History),
		})
	}
	return result
}

// SessionInfo holds lightweight session metadata.
type SessionInfo struct {
	SessionID  string `json:"session_id"`
	CWD        string `json:"cwd"`
	Model      string `json:"model"`
	HistoryLen int    `json:"history_len"`
}

// UpdateCWD updates the working directory for a session.
func (m *SessionManager) UpdateCWD(sessionID, cwd string) *SessionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	state.CWD = cwd
	return state
}

// AppendHistory adds a history entry to a session.
func (m *SessionManager) AppendHistory(sessionID string, entry HistoryEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.sessions[sessionID]
	if !ok {
		return
	}
	state.History = append(state.History, entry)
}

// GetHistory returns the conversation history for a session as JSON.
func (m *SessionManager) GetHistory(sessionID string) ([]byte, error) {
	m.mu.Lock()
	state, ok := m.sessions[sessionID]
	m.mu.Unlock()

	if !ok {
		return json.Marshal([]HistoryEntry{})
	}
	return json.Marshal(state.History)
}

// Cleanup removes all sessions.
func (m *SessionManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := len(m.sessions)
	m.sessions = make(map[string]*SessionState)
	if count > 0 {
		slog.Info("cleaned up ACP sessions", "count", count)
	}
}
