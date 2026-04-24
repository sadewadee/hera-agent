package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/sadewadee/hera/internal/tools"
)

// NotificationsTool sends system desktop notifications.
type NotificationsTool struct{}

type notificationsArgs struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Sound   bool   `json:"sound,omitempty"`
}

func (t *NotificationsTool) Name() string { return "notifications" }

func (t *NotificationsTool) Description() string {
	return "Sends a desktop notification with a title and message."
}

func (t *NotificationsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"title": {
				"type": "string",
				"description": "Notification title."
			},
			"message": {
				"type": "string",
				"description": "Notification body text."
			},
			"sound": {
				"type": "boolean",
				"description": "Play a sound with the notification. Defaults to false."
			}
		},
		"required": ["title", "message"]
	}`)
}

func (t *NotificationsTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params notificationsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, params.Message, params.Title)
		if params.Sound {
			script += ` sound name "Glass"`
		}
		cmd = exec.CommandContext(ctx, "osascript", "-e", script)
	case "linux":
		cmd = exec.CommandContext(ctx, "notify-send", params.Title, params.Message)
	default:
		return &tools.Result{Content: fmt.Sprintf("notifications not supported on %s", runtime.GOOS), IsError: true}, nil
	}

	if err := cmd.Run(); err != nil {
		return &tools.Result{Content: fmt.Sprintf("notification failed: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("Notification sent: %s", params.Title)}, nil
}

// RegisterNotifications registers the notifications tool with the given registry.
func RegisterNotifications(registry *tools.Registry) {
	registry.Register(&NotificationsTool{})
}
