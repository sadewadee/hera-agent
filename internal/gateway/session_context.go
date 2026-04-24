package gateway

import "sync"

// SessionContext holds platform-specific metadata for a gateway session.
type SessionContext struct {
	mu       sync.RWMutex
	metadata map[string]interface{}
}

// NewSessionContext creates a new session context.
func NewSessionContext() *SessionContext {
	return &SessionContext{metadata: make(map[string]interface{})}
}

// Set stores a value in the session context.
func (sc *SessionContext) Set(key string, value interface{}) {
	sc.mu.Lock(); defer sc.mu.Unlock(); sc.metadata[key] = value
}

// Get retrieves a value from the session context.
func (sc *SessionContext) Get(key string) (interface{}, bool) {
	sc.mu.RLock(); defer sc.mu.RUnlock(); v, ok := sc.metadata[key]; return v, ok
}

// All returns a copy of all metadata.
func (sc *SessionContext) All() map[string]interface{} {
	sc.mu.RLock(); defer sc.mu.RUnlock()
	cp := make(map[string]interface{}, len(sc.metadata))
	for k, v := range sc.metadata { cp[k] = v }
	return cp
}
