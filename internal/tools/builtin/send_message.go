package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// SendMessageTool dispatches an outgoing message (optionally with file
// attachments) to a connected gateway adapter. Requires a non-nil
// *gateway.Gateway injected via ToolDeps — without one, the tool fails
// loudly instead of pretending to succeed.
type SendMessageTool struct {
	gw *gateway.Gateway
}

type sendMessageArgs struct {
	Platform    string   `json:"platform"`
	ChatID      string   `json:"chat_id"`
	Text        string   `json:"text"`
	Attachments []string `json:"attachments,omitempty"`
	Caption     string   `json:"caption,omitempty"`
	Format      string   `json:"format,omitempty"` // markdown|plain|html
}

func (t *SendMessageTool) Name() string { return "send_message" }

func (t *SendMessageTool) Description() string {
	return "Sends a message (with optional file attachments) to a specific platform chat via the live gateway. Supports telegram, discord, slack, whatsapp, etc. Attachment paths accept ~, $HERA_HOME, and absolute paths. Fails if the gateway or adapter is not available."
}

func (t *SendMessageTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"platform": {"type": "string", "description": "Target platform name (e.g. telegram, discord, slack). Must match a connected adapter."},
			"chat_id": {"type": "string", "description": "Chat/channel/user ID on that platform."},
			"text": {"type": "string", "description": "Message text. May be empty when sending only attachments."},
			"attachments": {"type": "array", "items": {"type": "string"}, "description": "Optional file paths to upload. MIME auto-detected from extension. Paths support ~, $HERA_HOME, absolute."},
			"caption": {"type": "string", "description": "Optional caption applied to each attachment."},
			"format": {"type": "string", "enum": ["markdown", "plain", "html"], "description": "Text formatting hint for adapters that distinguish."}
		},
		"required": ["platform", "chat_id"]
	}`)
}

func (t *SendMessageTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a sendMessageArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	if a.Platform == "" {
		return &tools.Result{Content: "platform is required", IsError: true}, nil
	}
	if a.ChatID == "" {
		return &tools.Result{Content: "chat_id is required", IsError: true}, nil
	}
	if a.Text == "" && len(a.Attachments) == 0 {
		return &tools.Result{Content: "either text or attachments must be provided", IsError: true}, nil
	}

	if t.gw == nil {
		return &tools.Result{
			Content: "send_message is not wired: no gateway available in this binary. Use hera-agent (daemon) or configure the gateway in this entrypoint.",
			IsError: true,
		}, nil
	}

	adapter := t.gw.FindAdapter(a.Platform)
	if adapter == nil {
		return &tools.Result{
			Content: fmt.Sprintf("platform %q has no registered adapter (check config.gateway.platforms.%s.enabled)", a.Platform, a.Platform),
			IsError: true,
		}, nil
	}
	if !adapter.IsConnected() {
		return &tools.Result{
			Content: fmt.Sprintf("platform %q adapter is registered but not connected — check credentials, network, and adapter logs", a.Platform),
			IsError: true,
		}, nil
	}

	media := make([]gateway.Media, 0, len(a.Attachments))
	for _, raw := range a.Attachments {
		p := paths.Normalize(raw)
		if p == "" {
			return &tools.Result{
				Content: fmt.Sprintf("attachment path %q resolved to empty (traversal refused or empty input)", raw),
				IsError: true,
			}, nil
		}
		info, err := os.Stat(p)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("attachment %q: %v", raw, err), IsError: true}, nil
		}
		if info.IsDir() {
			return &tools.Result{Content: fmt.Sprintf("attachment %q is a directory; send individual files", raw), IsError: true}, nil
		}
		media = append(media, gateway.Media{
			Type:    detectMediaType(p),
			URL:     p, // local path; adapter decides upload vs URL
			Caption: a.Caption,
		})
	}

	msg := gateway.OutgoingMessage{
		Text:   a.Text,
		Media:  media,
		Format: a.Format,
	}
	if err := adapter.Send(ctx, a.ChatID, msg); err != nil {
		return &tools.Result{
			Content: fmt.Sprintf("adapter.Send failed for %s:%s: %v", a.Platform, a.ChatID, err),
			IsError: true,
		}, nil
	}

	parts := []string{fmt.Sprintf("Sent to %s:%s", a.Platform, a.ChatID)}
	if a.Text != "" {
		parts = append(parts, fmt.Sprintf("text=%d chars", len(a.Text)))
	}
	if n := len(media); n > 0 {
		parts = append(parts, fmt.Sprintf("attachments=%d", n))
	}
	return &tools.Result{Content: strings.Join(parts, " ")}, nil
}

// detectMediaType picks a gateway.MediaType from a filename extension.
// Unknown extensions default to MediaFile (document upload on Telegram),
// which is safe for .md / .txt / .pdf / .zip / arbitrary data.
func detectMediaType(path string) gateway.MediaType {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return gateway.MediaPhoto
	case ".mp3", ".wav", ".flac", ".m4a", ".aac":
		return gateway.MediaAudio
	case ".ogg", ".oga", ".opus":
		// ogg is commonly used for voice on Telegram; treat as voice so the
		// adapter renders a push-to-talk bubble instead of an audio attachment.
		return gateway.MediaVoice
	case ".mp4", ".mov", ".webm", ".mkv", ".avi":
		return gateway.MediaVideo
	default:
		return gateway.MediaFile
	}
}

// RegisterSendMessage wires the send_message tool. Pass the live gateway
// so the tool can actually reach adapters; pass nil for entrypoints that
// don't host a gateway (the tool will refuse with a clear error instead
// of pretending to succeed — this was the v0.14.1 hallucination bug).
func RegisterSendMessage(registry *tools.Registry, gw *gateway.Gateway) {
	registry.Register(&SendMessageTool{gw: gw})
}
