package hcore

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// AppState manages global application state and shutdown coordination.
type AppState struct {
	mu       sync.RWMutex
	running  bool
	shutdown chan struct{}
}

// NewAppState creates a new app state.
func NewAppState() *AppState {
	return &AppState{running: true, shutdown: make(chan struct{})}
}

// IsRunning returns whether the app is still running.
func (s *AppState) IsRunning() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.running }

// Shutdown signals the app to stop.
func (s *AppState) Shutdown() {
	s.mu.Lock(); defer s.mu.Unlock()
	if s.running { s.running = false; close(s.shutdown) }
}

// Done returns a channel that closes when shutdown is requested.
func (s *AppState) Done() <-chan struct{} { return s.shutdown }

// WaitForSignal blocks until SIGINT or SIGTERM, then calls cancel.
func WaitForSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
}
