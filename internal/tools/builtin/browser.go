package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

type BrowserTool struct{ client *http.Client }

type browserArgs struct {
	URL       string `json:"url"`
	Selector  string `json:"selector,omitempty"`
	MaxLength int    `json:"max_length,omitempty"`
}

func (t *BrowserTool) Name() string        { return "browser" }
func (t *BrowserTool) Description() string  { return "Opens a URL and extracts text content from the page." }
func (t *BrowserTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"URL to fetch"},"selector":{"type":"string","description":"CSS selector to extract (optional)"},"max_length":{"type":"integer","description":"Max chars to return"}},"required":["url"]}`)
}

func (t *BrowserTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a browserArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	if t.client == nil {
		t.client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", a.URL, nil)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("request error: %v", err), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "Hera/1.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("fetch error: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	text := stripHTMLTags(string(body))
	maxLen := a.MaxLength
	if maxLen <= 0 { maxLen = 10000 }
	if len(text) > maxLen { text = text[:maxLen] + "\n... [truncated]" }
	return &tools.Result{Content: text}, nil
}

func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<': inTag = true
		case r == '>': inTag = false
		case !inTag: b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func RegisterBrowser(registry *tools.Registry) { registry.Register(&BrowserTool{}) }
