package builtin_hooks

import (
	"context"
	"os"

	"github.com/sadewadee/hera/internal/gateway"
)

// BootMDHook loads a boot.md file and injects its content into the first message of a session.
type BootMDHook struct {
	FilePath string
	content  string
	loaded   bool
}

// NewBootMDHook creates a hook that loads boot.md from the given path.
func NewBootMDHook(filePath string) *BootMDHook {
	return &BootMDHook{FilePath: filePath}
}

func (h *BootMDHook) Name() string { return "boot_md" }

func (h *BootMDHook) BeforeMessage(_ context.Context, msg *gateway.IncomingMessage) (*gateway.IncomingMessage, error) {
	if !h.loaded {
		data, err := os.ReadFile(h.FilePath)
		if err == nil { h.content = string(data) }
		h.loaded = true
	}
	if h.content != "" {
		modified := *msg
		modified.Text = h.content + "\n\n" + msg.Text
		return &modified, nil
	}
	return msg, nil
}

func (h *BootMDHook) AfterMessage(_ context.Context, _ *gateway.IncomingMessage, _ string) error { return nil }
