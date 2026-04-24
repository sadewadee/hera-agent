package honcho

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Session represents a conversation session backed by Honcho.
type Session struct {
	Key              string
	HonchoSessionID  string
	Messages         []SessionMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SessionMessage is a single message in a Honcho session.
type SessionMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// AddMessage appends a message to the session's local cache.
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, SessionMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// SessionManager manages conversation sessions using Honcho.
type SessionManager struct {
	mu     sync.Mutex
	client *Client
	config *ClientConfig
	cache  map[string]*Session
}

// NewSessionManager creates a new session manager.
func NewSessionManager(client *Client, config *ClientConfig) *SessionManager {
	return &SessionManager{
		client: client,
		config: config,
		cache:  make(map[string]*Session),
	}
}

// GetOrCreate returns an existing session or creates a new one.
func (sm *SessionManager) GetOrCreate(key string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sess, ok := sm.cache[key]; ok {
		return sess, nil
	}

	sess := &Session{
		Key:       key,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	sm.cache[key] = sess

	slog.Debug("honcho session created", "key", key)
	return sess, nil
}

// GetPeerCard retrieves the user's peer card for the given session.
func (sm *SessionManager) GetPeerCard(sessionKey string) (string, error) {
	if sm.client == nil {
		return "", fmt.Errorf("honcho client not initialized")
	}

	result, err := sm.client.request("GET", "/v1/peer/card", nil)
	if err != nil {
		return "", err
	}

	card, _ := result["card"].(string)
	return card, nil
}

// SearchContext performs a semantic search over Honcho's stored context.
func (sm *SessionManager) SearchContext(sessionKey, query string, maxTokens int) (string, error) {
	if sm.client == nil {
		return "", fmt.Errorf("honcho client not initialized")
	}

	result, err := sm.client.request("POST", "/v1/context/search", map[string]interface{}{
		"query":      query,
		"max_tokens": maxTokens,
	})
	if err != nil {
		return "", err
	}

	context, _ := result["context"].(string)
	return context, nil
}

// DialecticQuery asks Honcho a natural language question.
func (sm *SessionManager) DialecticQuery(sessionKey, query, peer string) (string, error) {
	if sm.client == nil {
		return "", fmt.Errorf("honcho client not initialized")
	}

	result, err := sm.client.request("POST", "/v1/dialectic/query", map[string]interface{}{
		"query": query,
		"peer":  peer,
	})
	if err != nil {
		return "", err
	}

	answer, _ := result["answer"].(string)
	return answer, nil
}

// CreateConclusion writes a persistent conclusion about the user.
func (sm *SessionManager) CreateConclusion(sessionKey, conclusion string) error {
	if sm.client == nil {
		return fmt.Errorf("honcho client not initialized")
	}

	_, err := sm.client.request("POST", "/v1/conclusions", map[string]interface{}{
		"content": conclusion,
	})
	return err
}

// FlushAll flushes all pending messages for all sessions.
func (sm *SessionManager) FlushAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for key, sess := range sm.cache {
		if len(sess.Messages) > 0 {
			slog.Debug("honcho flushing session", "key", key, "messages", len(sess.Messages))
		}
	}
}
