package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// BrowserAutomation provides browser automation through a Playwright MCP server
// or any CDP-compatible endpoint. Supports navigation, clicking, typing,
// screenshots, and JavaScript evaluation.
type BrowserAutomation struct {
	client  *http.Client
	baseURL string // Playwright server URL (default: http://localhost:3000)
}

type browserAutoArgs struct {
	Action   string `json:"action"` // navigate, click, type, screenshot, evaluate, snapshot
	URL      string `json:"url,omitempty"`
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text,omitempty"`
	Script   string `json:"script,omitempty"`
}

func (b *BrowserAutomation) Name() string { return "browser_auto" }

func (b *BrowserAutomation) Description() string {
	return "Automates browser interactions: navigate to URLs, click elements, type text, take screenshots, and run JavaScript. Use for web research, form filling, testing, and data extraction."
}

func (b *BrowserAutomation) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["navigate", "click", "type", "screenshot", "evaluate", "snapshot"],
				"description": "Browser action to perform"
			},
			"url": {
				"type": "string",
				"description": "URL to navigate to (for 'navigate' action)"
			},
			"selector": {
				"type": "string",
				"description": "CSS selector for click/type actions"
			},
			"text": {
				"type": "string",
				"description": "Text to type (for 'type' action)"
			},
			"script": {
				"type": "string",
				"description": "JavaScript to evaluate (for 'evaluate' action)"
			}
		},
		"required": ["action"]
	}`)
}

func (b *BrowserAutomation) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params browserAutoArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	baseURL := b.baseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	switch params.Action {
	case "navigate":
		if params.URL == "" {
			return &tools.Result{Content: "url is required for navigate action", IsError: true}, nil
		}
		return b.callPlaywright(ctx, baseURL, "browser_navigate", map[string]any{"url": params.URL})

	case "click":
		if params.Selector == "" {
			return &tools.Result{Content: "selector is required for click action", IsError: true}, nil
		}
		return b.callPlaywright(ctx, baseURL, "browser_click", map[string]any{"selector": params.Selector})

	case "type":
		if params.Selector == "" || params.Text == "" {
			return &tools.Result{Content: "selector and text are required for type action", IsError: true}, nil
		}
		return b.callPlaywright(ctx, baseURL, "browser_type", map[string]any{
			"selector": params.Selector,
			"text":     params.Text,
		})

	case "screenshot":
		return b.callPlaywright(ctx, baseURL, "browser_take_screenshot", map[string]any{})

	case "evaluate":
		if params.Script == "" {
			return &tools.Result{Content: "script is required for evaluate action", IsError: true}, nil
		}
		return b.callPlaywright(ctx, baseURL, "browser_evaluate", map[string]any{"script": params.Script})

	case "snapshot":
		return b.callPlaywright(ctx, baseURL, "browser_snapshot", map[string]any{})

	default:
		return &tools.Result{Content: fmt.Sprintf("unknown action: %s (use: navigate, click, type, screenshot, evaluate, snapshot)", params.Action), IsError: true}, nil
	}
}

func (b *BrowserAutomation) callPlaywright(ctx context.Context, baseURL, method string, params map[string]any) (*tools.Result, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      method,
			"arguments": params,
		},
	}

	data, _ := json.Marshal(reqBody)

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, baseURL, bytes.NewReader(data))
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return &tools.Result{
			Content: "Browser automation server not available. Start a Playwright MCP server or configure mcp_servers in config.yaml.",
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return &tools.Result{Content: string(body)}, nil
	}

	if result.Error != nil {
		return &tools.Result{Content: result.Error.Message, IsError: true}, nil
	}

	var texts []string
	for _, c := range result.Result.Content {
		if c.Text != "" {
			texts = append(texts, c.Text)
		}
	}

	if len(texts) == 0 {
		return &tools.Result{Content: "(no content returned)"}, nil
	}

	output := texts[0]
	for _, t := range texts[1:] {
		output += "\n" + t
	}

	if len(output) > 50000 {
		output = output[:50000] + "\n... (truncated)"
	}

	return &tools.Result{Content: output}, nil
}

// RegisterBrowserAutomation registers the browser automation tool.
func RegisterBrowserAutomation(registry *tools.Registry) {
	registry.Register(&BrowserAutomation{
		client: &http.Client{Timeout: 30 * time.Second},
	})
}
