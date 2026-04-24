package platforms

import (
	"context"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestMCPBridgeAdapter_Connect(t *testing.T) {
	adapter := NewMCPBridgeAdapter()

	err := adapter.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if !adapter.IsConnected() {
		t.Error("IsConnected() should be true after Connect")
	}
}

func TestMCPBridgeAdapter_Disconnect(t *testing.T) {
	adapter := NewMCPBridgeAdapter()

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	err := adapter.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	if adapter.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect")
	}
}

func TestMCPBridgeAdapter_Send(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	adapter.SetConnected(true)

	// Create a pending channel to receive the message.
	chatID := "mcp:testuser"
	responseCh := make(chan string, 1)

	adapter.mu.Lock()
	adapter.pending[chatID] = responseCh
	adapter.mu.Unlock()

	err := adapter.Send(context.Background(), chatID, gateway.OutgoingMessage{Text: "response text"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case got := <-responseCh:
		if got != "response text" {
			t.Errorf("response = %q, want 'response text'", got)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for response on pending channel")
	}
}

func TestMCPBridgeAdapter_Send_NoPending(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	adapter.SetConnected(true)

	// Send without a pending channel should not error.
	err := adapter.Send(context.Background(), "mcp:nobody", gateway.OutgoingMessage{Text: "lost"})
	if err != nil {
		t.Fatalf("Send() error = %v (should silently drop)", err)
	}
}

func TestMCPBridgeAdapter_SendToGateway(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	adapter.SetConnected(true)

	var receivedText string
	var receivedChatID string
	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		receivedText = msg.Text
		receivedChatID = msg.ChatID
		// Simulate the agent responding by calling Send.
		adapter.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "agent response"})
	})

	response, err := adapter.SendToGateway(context.Background(), "user question", "user1")
	if err != nil {
		t.Fatalf("SendToGateway() error = %v", err)
	}

	if receivedText != "user question" {
		t.Errorf("received text = %q, want 'user question'", receivedText)
	}
	if receivedChatID != "mcp:user1" {
		t.Errorf("received chatID = %q, want 'mcp:user1'", receivedChatID)
	}
	if response != "agent response" {
		t.Errorf("response = %q, want 'agent response'", response)
	}
}

func TestMCPBridgeAdapter_SendToGateway_NoHandler(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	adapter.SetConnected(true)

	_, err := adapter.SendToGateway(context.Background(), "hello", "user1")
	if err == nil {
		t.Fatal("SendToGateway() should fail with no handler")
	}
}

func TestMCPBridgeAdapter_SendToGateway_ContextCancelled(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	adapter.SetConnected(true)

	// Handler that does NOT call Send (simulates no response).
	adapter.OnMessage(func(_ context.Context, _ gateway.IncomingMessage) {
		// Do nothing -- agent never responds.
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := adapter.SendToGateway(ctx, "hello", "user1")
	if err == nil {
		t.Fatal("SendToGateway() should fail when context is cancelled")
	}
}

func TestMCPBridgeAdapter_GetChatInfo(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	info, err := adapter.GetChatInfo(context.Background(), "mcp:user1")
	if err != nil {
		t.Fatalf("GetChatInfo() error = %v", err)
	}
	if info.Platform != "mcpbridge" {
		t.Errorf("Platform = %q, want 'mcpbridge'", info.Platform)
	}
	if info.Title != "MCP Bridge" {
		t.Errorf("Title = %q, want 'MCP Bridge'", info.Title)
	}
}

func TestMCPBridgeAdapter_SupportedMedia(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	media := adapter.SupportedMedia()
	if len(media) != 0 {
		t.Errorf("SupportedMedia() len = %d, want 0", len(media))
	}
}

func TestMCPBridgeAdapter_PendingCleanup(t *testing.T) {
	adapter := NewMCPBridgeAdapter()
	adapter.SetConnected(true)

	adapter.OnMessage(func(_ context.Context, msg gateway.IncomingMessage) {
		adapter.Send(context.Background(), msg.ChatID, gateway.OutgoingMessage{Text: "reply"})
	})

	// Call SendToGateway which creates and cleans up the pending channel.
	_, err := adapter.SendToGateway(context.Background(), "hello", "user1")
	if err != nil {
		t.Fatalf("SendToGateway() error = %v", err)
	}

	// Verify pending map is cleaned up.
	adapter.mu.Lock()
	_, exists := adapter.pending["mcp:user1"]
	adapter.mu.Unlock()

	if exists {
		t.Error("pending channel should be cleaned up after SendToGateway returns")
	}
}
