package platforms

import (
	"sync"

	"github.com/sadewadee/hera/internal/gateway"
)

// BaseAdapter provides common fields and methods shared across all
// platform adapters. Embed this struct in concrete adapters.
type BaseAdapter struct {
	AdapterName string
	mu          sync.Mutex
	connected   bool
	handler     gateway.MessageHandler
}

// Name returns the adapter's platform name.
func (b *BaseAdapter) Name() string {
	return b.AdapterName
}

// IsConnected reports whether the adapter is currently connected.
func (b *BaseAdapter) IsConnected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connected
}

// SetConnected updates the connection status. This should be called by
// concrete adapters when their connection state changes.
func (b *BaseAdapter) SetConnected(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connected = v
}

// OnMessage registers the handler that receives incoming messages.
func (b *BaseAdapter) OnMessage(handler gateway.MessageHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handler = handler
}

// Handler returns the currently registered message handler.
func (b *BaseAdapter) Handler() gateway.MessageHandler {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.handler
}
