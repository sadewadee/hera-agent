package platforms

import (
	"context"
	"testing"

	"github.com/sadewadee/hera/internal/gateway"
)

func TestBaseAdapter_Name(t *testing.T) {
	b := &BaseAdapter{AdapterName: "test"}
	if b.Name() != "test" {
		t.Errorf("Name() = %q, want %q", b.Name(), "test")
	}
}

func TestBaseAdapter_IsConnected_DefaultFalse(t *testing.T) {
	b := &BaseAdapter{AdapterName: "test"}
	if b.IsConnected() {
		t.Error("IsConnected() should be false by default")
	}
}

func TestBaseAdapter_SetConnected(t *testing.T) {
	b := &BaseAdapter{AdapterName: "test"}
	b.SetConnected(true)
	if !b.IsConnected() {
		t.Error("IsConnected() should be true after SetConnected(true)")
	}
	b.SetConnected(false)
	if b.IsConnected() {
		t.Error("IsConnected() should be false after SetConnected(false)")
	}
}

func TestBaseAdapter_OnMessage(t *testing.T) {
	b := &BaseAdapter{AdapterName: "test"}

	b.OnMessage(func(_ context.Context, _ gateway.IncomingMessage) {
		// handler set
	})

	h := b.Handler()
	if h == nil {
		t.Fatal("Handler() should not be nil after OnMessage")
	}
}

func TestBaseAdapter_Handler_NilByDefault(t *testing.T) {
	b := &BaseAdapter{AdapterName: "test"}
	if b.Handler() != nil {
		t.Error("Handler() should be nil by default")
	}
}
