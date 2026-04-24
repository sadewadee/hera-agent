package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// ScreenshotTool captures screenshots of the desktop or specific windows.
type ScreenshotTool struct{}

type screenshotArgs struct {
	Output  string `json:"output,omitempty"`
	Display int    `json:"display,omitempty"`
	Delay   int    `json:"delay,omitempty"`
}

func (t *ScreenshotTool) Name() string { return "screenshot" }

func (t *ScreenshotTool) Description() string {
	return "Captures a screenshot of the desktop. Supports delay and custom output path."
}

func (t *ScreenshotTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"output": {
				"type": "string",
				"description": "Output file path. Defaults to a timestamped PNG in the current directory."
			},
			"display": {
				"type": "integer",
				"description": "Display number to capture (0-based). Defaults to primary display."
			},
			"delay": {
				"type": "integer",
				"description": "Delay in seconds before capturing. Defaults to 0."
			}
		}
	}`)
}

func (t *ScreenshotTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params screenshotArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
		}
	}

	if params.Delay > 0 {
		select {
		case <-time.After(time.Duration(params.Delay) * time.Second):
		case <-ctx.Done():
			return &tools.Result{Content: "cancelled during delay", IsError: true}, nil
		}
	}

	output := params.Output
	if output == "" {
		output = fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405"))
	}
	output, _ = filepath.Abs(output)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "screencapture", "-x", output)
	case "linux":
		cmd = exec.CommandContext(ctx, "import", "-window", "root", output)
	default:
		return &tools.Result{Content: fmt.Sprintf("screenshot not supported on %s", runtime.GOOS), IsError: true}, nil
	}

	if err := cmd.Run(); err != nil {
		return &tools.Result{Content: fmt.Sprintf("screenshot failed: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("Screenshot saved to %s", output)}, nil
}

// RegisterScreenshot registers the screenshot tool with the given registry.
func RegisterScreenshot(registry *tools.Registry) {
	registry.Register(&ScreenshotTool{})
}
