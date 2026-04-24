package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sadewadee/hera/internal/llm"
)

// Session represents an active conversation session.
// The mu field protects Messages and TurnCount from concurrent access.
type Session struct {
	mu        sync.Mutex
	ID        string        `json:"id"`
	Platform  string        `json:"platform"`
	UserID    string        `json:"user_id"`
	Messages  []llm.Message `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	TurnCount int           `json:"turn_count"`
	Title     string        `json:"title,omitempty"`
}

// Lock acquires the session mutex for direct field manipulation.
func (s *Session) Lock() { s.mu.Lock() }

// Unlock releases the session mutex.
func (s *Session) Unlock() { s.mu.Unlock() }

// AppendMessage safely appends a message to the session and increments TurnCount for user messages.
func (s *Session) AppendMessage(msg llm.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msg)
	if msg.Role == llm.RoleUser {
		s.TurnCount++
	}
}

// GetMessages returns a copy of the session's messages for safe concurrent use.
func (s *Session) GetMessages() []llm.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]llm.Message, len(s.Messages))
	copy(cp, s.Messages)
	return cp
}

// NewSession creates a new session with the format "{platform}:{user_id}:{uuid}".
func NewSession(platform, userID string) *Session {
	now := time.Now()
	return &Session{
		ID:        fmt.Sprintf("%s:%s:%s", platform, userID, uuid.New().String()),
		Platform:  platform,
		UserID:    userID,
		Messages:  []llm.Message{},
		CreatedAt: now,
		UpdatedAt: now,
		TurnCount: 0,
	}
}

// SessionLifecycle holds optional callbacks fired on session transitions.
// OnStart fires when a brand-new Session is added to the manager.
// OnEnd fires just before a Session is removed (expired, cleaned, or deleted).
// Both callbacks run after the manager's lock is released; they may call
// session.GetMessages() safely.
type SessionLifecycle struct {
	OnStart func(*Session)
	OnEnd   func(*Session)
}

// SessionManager manages session lifecycle with in-memory caching.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	// index by platform:userID for quick lookups
	byPlatformUser map[string]string // "platform:userID" -> sessionID
	timeout        time.Duration
	lifecycle      *SessionLifecycle
}

// NewSessionManager creates a new session manager with the given session timeout.
func NewSessionManager(timeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions:       make(map[string]*Session),
		byPlatformUser: make(map[string]string),
		timeout:        timeout,
	}
}

// SetLifecycle installs (or replaces) the lifecycle callbacks. Pass nil to
// disable. Safe to call at any time, though typically wired once at startup.
func (sm *SessionManager) SetLifecycle(lc *SessionLifecycle) {
	sm.mu.Lock()
	sm.lifecycle = lc
	sm.mu.Unlock()
}

// Create creates a new session and adds it to the manager.
func (sm *SessionManager) Create(platform, userID string) *Session {
	s := NewSession(platform, userID)

	sm.mu.Lock()
	sm.sessions[s.ID] = s
	key := platformUserKey(platform, userID)
	sm.byPlatformUser[key] = s.ID
	onStart := lifecycleOnStart(sm.lifecycle)
	sm.mu.Unlock()

	if onStart != nil {
		onStart(s)
	}
	return s
}

// Get returns a session by its ID.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[id]
	return s, ok
}

// GetOrCreate returns an existing session for the platform/user pair,
// or creates a new one if none exists or the existing one has expired.
func (sm *SessionManager) GetOrCreate(platform, userID string) *Session {
	sm.mu.Lock()

	var expired *Session
	key := platformUserKey(platform, userID)
	if id, ok := sm.byPlatformUser[key]; ok {
		if s, exists := sm.sessions[id]; exists {
			if time.Since(s.UpdatedAt) < sm.timeout {
				sm.mu.Unlock()
				return s
			}
			// Session expired; remove it, remember for OnEnd.
			expired = s
			delete(sm.sessions, id)
			delete(sm.byPlatformUser, key)
		}
	}

	// Create new session.
	s := NewSession(platform, userID)
	sm.sessions[s.ID] = s
	sm.byPlatformUser[key] = s.ID
	onStart := lifecycleOnStart(sm.lifecycle)
	onEnd := lifecycleOnEnd(sm.lifecycle)
	sm.mu.Unlock()

	if expired != nil && onEnd != nil {
		onEnd(expired)
	}
	if onStart != nil {
		onStart(s)
	}
	return s
}

// List returns all active sessions.
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		result = append(result, s)
	}
	return result
}

// Delete removes a session by ID.
func (sm *SessionManager) Delete(id string) {
	sm.mu.Lock()
	var ended *Session
	if s, ok := sm.sessions[id]; ok {
		key := platformUserKey(s.Platform, s.UserID)
		delete(sm.byPlatformUser, key)
		delete(sm.sessions, id)
		ended = s
	}
	onEnd := lifecycleOnEnd(sm.lifecycle)
	sm.mu.Unlock()

	if ended != nil && onEnd != nil {
		onEnd(ended)
	}
}

// CleanExpired removes all sessions that have exceeded the timeout.
func (sm *SessionManager) CleanExpired() {
	sm.mu.Lock()
	var ended []*Session
	for id, s := range sm.sessions {
		if time.Since(s.UpdatedAt) >= sm.timeout {
			key := platformUserKey(s.Platform, s.UserID)
			delete(sm.byPlatformUser, key)
			delete(sm.sessions, id)
			ended = append(ended, s)
		}
	}
	onEnd := lifecycleOnEnd(sm.lifecycle)
	sm.mu.Unlock()

	if onEnd != nil {
		for _, s := range ended {
			onEnd(s)
		}
	}
}

// Touch updates the session's UpdatedAt timestamp.
func (sm *SessionManager) Touch(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[id]; ok {
		s.UpdatedAt = time.Now()
	}
}

// Branch creates a copy of a session at its current state, including all messages.
// The new session gets a new ID but shares the same platform and user.
// Messages are copied, not shared -- changes to one session do not affect the other.
func (sm *SessionManager) Branch(sessionID string) (*Session, error) {
	sm.mu.Lock()

	orig, ok := sm.sessions[sessionID]
	if !ok {
		sm.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	branched := NewSession(orig.Platform, orig.UserID)

	// Copy messages from the original session.
	orig.mu.Lock()
	branched.Messages = make([]llm.Message, len(orig.Messages))
	copy(branched.Messages, orig.Messages)
	branched.TurnCount = orig.TurnCount
	orig.mu.Unlock()

	sm.sessions[branched.ID] = branched
	// Note: byPlatformUser still points to the original; the branch is accessible by ID only.

	onStart := lifecycleOnStart(sm.lifecycle)
	sm.mu.Unlock()

	if onStart != nil {
		onStart(branched)
	}
	return branched, nil
}

// Fork creates a copy of a session up to a specific message index.
// Messages from index 0 through messageIndex (inclusive) are copied.
// This allows "going back in time" to a particular point in the conversation.
func (sm *SessionManager) Fork(sessionID string, messageIndex int) (*Session, error) {
	sm.mu.Lock()

	orig, ok := sm.sessions[sessionID]
	if !ok {
		sm.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	forked := NewSession(orig.Platform, orig.UserID)

	orig.mu.Lock()
	if messageIndex < 0 || messageIndex >= len(orig.Messages) {
		orig.mu.Unlock()
		sm.mu.Unlock()
		return nil, fmt.Errorf("message index %d out of range [0, %d)", messageIndex, len(orig.Messages))
	}

	// Copy messages up to and including messageIndex.
	forked.Messages = make([]llm.Message, messageIndex+1)
	copy(forked.Messages, orig.Messages[:messageIndex+1])

	// Recount user turns in the forked range.
	turnCount := 0
	for _, m := range forked.Messages {
		if m.Role == llm.RoleUser {
			turnCount++
		}
	}
	forked.TurnCount = turnCount
	orig.mu.Unlock()

	sm.sessions[forked.ID] = forked

	onStart := lifecycleOnStart(sm.lifecycle)
	sm.mu.Unlock()

	if onStart != nil {
		onStart(forked)
	}
	return forked, nil
}

func platformUserKey(platform, userID string) string {
	return platform + ":" + userID
}

func lifecycleOnStart(lc *SessionLifecycle) func(*Session) {
	if lc == nil {
		return nil
	}
	return lc.OnStart
}

func lifecycleOnEnd(lc *SessionLifecycle) func(*Session) {
	if lc == nil {
		return nil
	}
	return lc.OnEnd
}
