package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// ClipboardTool provides clipboard read and write access.
type ClipboardTool struct{}

type clipboardArgs struct {
	Action string `json:"action"`
	Text   string `json:"text,omitempty"`
}

func (t *ClipboardTool) Name() string { return "clipboard" }

func (t *ClipboardTool) Description() string {
	return "Reads from or writes to the system clipboard."
}

func (t *ClipboardTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["read", "write"],
				"description": "read: get clipboard contents, write: set clipboard contents."
			},
			"text": {
				"type": "string",
				"description": "Text to write to clipboard (required for write action)."
			}
		},
		"required": ["action"]
	}`)
}

func (t *ClipboardTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params clipboardArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	switch params.Action {
	case "read":
		return clipboardRead(ctx)
	case "write":
		if params.Text == "" {
			return &tools.Result{Content: "text is required for write action", IsError: true}, nil
		}
		return clipboardWrite(ctx, params.Text)
	default:
		return &tools.Result{Content: "action must be 'read' or 'write'", IsError: true}, nil
	}
}

func clipboardRead(ctx context.Context) (*tools.Result, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbpaste")
	case "linux":
		cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o")
	default:
		return &tools.Result{Content: fmt.Sprintf("clipboard not supported on %s", runtime.GOOS), IsError: true}, nil
	}

	out, err := cmd.Output()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("clipboard read failed: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: string(out)}, nil
}

func clipboardWrite(ctx context.Context, text string) (*tools.Result, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "linux":
		cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
	default:
		return &tools.Result{Content: fmt.Sprintf("clipboard not supported on %s", runtime.GOOS), IsError: true}, nil
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return &tools.Result{Content: fmt.Sprintf("clipboard write failed: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("Copied %d characters to clipboard", len(text))}, nil
}

// RegisterClipboard registers the clipboard tool with the given registry.
func RegisterClipboard(registry *tools.Registry) {
	registry.Register(&ClipboardTool{})
}
