package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// GatewaySession links a platform user to an active agent conversation.
type GatewaySession struct {
	ID           string    `json:"id"`
	Platform     string    `json:"platform"`
	UserID       string    `json:"user_id"`
	ChatID       string    `json:"chat_id"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}

// SessionManager tracks active gateway sessions and handles expiry.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*GatewaySession // key -> session
	keyIndex map[string]string          // sessionID -> key (for Remove by ID)
	timeout  time.Duration
}

// NewSessionManager creates a new session manager with the given idle timeout.
func NewSessionManager(timeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*GatewaySession),
		keyIndex: make(map[string]string),
		timeout:  timeout,
	}
}

// sessionKey builds a unique key from platform and userID.
func sessionKey(platform, userID string) string {
	return platform + ":" + userID
}

// GetOrCreate returns an existing session or creates a new one for the
// given platform and user combination. LastActivity is always updated.
func (sm *SessionManager) GetOrCreate(platform, userID string) *GatewaySession {
	key := sessionKey(platform, userID)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sess, ok := sm.sessions[key]; ok {
		sess.LastActivity = time.Now()
		return sess
	}

	sess := &GatewaySession{
		ID:           generateSessionID(),
		Platform:     platform,
		UserID:       userID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	sm.sessions[key] = sess
	sm.keyIndex[sess.ID] = key
	return sess
}

// Remove deletes a session by its ID.
func (sm *SessionManager) Remove(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if key, ok := sm.keyIndex[sessionID]; ok {
		delete(sm.sessions, key)
		delete(sm.keyIndex, sessionID)
	}
}

// ActiveCount returns the number of currently tracked sessions.
func (sm *SessionManager) ActiveCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// CleanExpired removes sessions whose last activity exceeds the timeout.
// Returns the number of sessions removed.
func (sm *SessionManager) CleanExpired() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, sess := range sm.sessions {
		if now.Sub(sess.LastActivity) > sm.timeout {
			delete(sm.sessions, key)
			delete(sm.keyIndex, sess.ID)
			removed++
		}
	}
	return removed
}

// StartExpiryWatcher runs a goroutine that periodically cleans expired
// sessions. It stops when the context is cancelled.
func (sm *SessionManager) StartExpiryWatcher(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sm.CleanExpired()
			}
		}
	}()
}

// generateSessionID produces a random hex session identifier.
func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
