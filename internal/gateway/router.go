package gateway

import (
	"context"
	"sync"
)

// RouteHandler is called when a message is routed to a session.
type RouteHandler func(ctx context.Context, sess *GatewaySession, msg IncomingMessage)

// Router maps incoming messages to the appropriate gateway session and
// dispatches them to the configured handler.
type Router struct {
	mu      sync.RWMutex
	sm      *SessionManager
	handler RouteHandler
}

// NewRouter creates a router backed by the given session manager.
func NewRouter(sm *SessionManager) *Router {
	return &Router{sm: sm}
}

// SetHandler configures the function that processes routed messages.
func (r *Router) SetHandler(h RouteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handler = h
}

// Route finds or creates a session for the incoming message's platform
// and user, then dispatches to the handler.
func (r *Router) Route(ctx context.Context, msg IncomingMessage) {
	sess := r.sm.GetOrCreate(msg.Platform, msg.UserID)

	// Update ChatID on the session if provided.
	if msg.ChatID != "" {
		sess.ChatID = msg.ChatID
	}

	r.mu.RLock()
	h := r.handler
	r.mu.RUnlock()

	if h != nil {
		h(ctx, sess, msg)
	}
}
