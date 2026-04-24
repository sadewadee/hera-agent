package platforms

import (
	"context"
	"fmt"
	"sync"

	"github.com/sadewadee/hera/internal/gateway"
)

// MCPBridgeAdapter bridges MCP protocol messages into the gateway message flow.
// Unlike other adapters that connect to external services, this adapter acts
// as an internal bridge that MCP tool handlers use to inject messages.
type MCPBridgeAdapter struct {
	BaseAdapter
	mu      sync.Mutex
	pending map[string]chan string // chatID -> response channel
}

// NewMCPBridgeAdapter creates an MCP bridge adapter.
func NewMCPBridgeAdapter() *MCPBridgeAdapter {
	return &MCPBridgeAdapter{
		BaseAdapter: BaseAdapter{AdapterName: "mcpbridge"},
		pending:     make(map[string]chan string),
	}
}

func (m *MCPBridgeAdapter) Connect(_ context.Context) error {
	m.SetConnected(true)
	return nil
}

func (m *MCPBridgeAdapter) Disconnect(_ context.Context) error {
	m.SetConnected(false)
	return nil
}

func (m *MCPBridgeAdapter) Send(_ context.Context, chatID string, msg gateway.OutgoingMessage) error {
	m.mu.Lock()
	ch, ok := m.pending[chatID]
	m.mu.Unlock()

	if ok {
		select {
		case ch <- msg.Text:
		default:
		}
	}
	return nil
}

// SendToGateway pushes a message from an MCP client into the gateway.
// It returns the agent's response, blocking until the handler calls Send().
func (m *MCPBridgeAdapter) SendToGateway(ctx context.Context, text, userID string) (string, error) {
	handler := m.Handler()
	if handler == nil {
		return "", fmt.Errorf("mcpbridge: no message handler")
	}

	chatID := "mcp:" + userID
	responseCh := make(chan string, 1)

	m.mu.Lock()
	m.pending[chatID] = responseCh
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pending, chatID)
		m.mu.Unlock()
	}()

	handler(ctx, gateway.IncomingMessage{
		Platform: "mcpbridge",
		ChatID:   chatID,
		UserID:   userID,
		Username: userID,
		Text:     text,
	})

	select {
	case response := <-responseCh:
		return response, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (m *MCPBridgeAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	return &gateway.ChatInfo{
		ID:       chatID,
		Title:    "MCP Bridge",
		Type:     "private",
		Platform: "mcpbridge",
	}, nil
}

func (m *MCPBridgeAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{}
}
