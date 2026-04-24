package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// HTTPClientTool makes HTTP requests to arbitrary URLs.
type HTTPClientTool struct {
	client *http.Client
}

type httpClientArgs struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

func (t *HTTPClientTool) Name() string { return "http_request" }

func (t *HTTPClientTool) Description() string {
	return "Makes HTTP requests (GET, POST, PUT, PATCH, DELETE) to any URL and returns the response."
}

func (t *HTTPClientTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
				"description": "HTTP method."
			},
			"url": {
				"type": "string",
				"description": "Request URL."
			},
			"headers": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "Request headers."
			},
			"body": {
				"type": "string",
				"description": "Request body (for POST, PUT, PATCH)."
			},
			"timeout": {
				"type": "integer",
				"description": "Request timeout in seconds. Defaults to 30."
			}
		},
		"required": ["method", "url"]
	}`)
}

func (t *HTTPClientTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a httpClientArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	timeout := time.Duration(a.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := t.client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	var body io.Reader
	if a.Body != "" {
		body = bytes.NewBufferString(a.Body)
	}

	req, err := http.NewRequestWithContext(ctx, a.Method, a.URL, body)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}

	for k, v := range a.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024)) // 100KB limit
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read response: %v", err), IsError: true}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Status: %s\n", resp.Status)
	fmt.Fprintf(&sb, "Headers:\n")
	for k, v := range resp.Header {
		fmt.Fprintf(&sb, "  %s: %s\n", k, strings.Join(v, ", "))
	}
	fmt.Fprintf(&sb, "\nBody:\n%s", string(respBody))

	return &tools.Result{Content: sb.String()}, nil
}

// RegisterHTTPClient registers the HTTP client tool with the given registry.
func RegisterHTTPClient(registry *tools.Registry) {
	registry.Register(&HTTPClientTool{})
}
