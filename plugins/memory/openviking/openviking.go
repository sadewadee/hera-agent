// Package openviking implements the OpenViking memory provider plugin.
//
// Context database by Volcengine (ByteDance) that organizes agent knowledge
// into a filesystem hierarchy (viking:// URIs) with tiered context loading,
// automatic memory extraction, and session management.
//
// Config via environment variables:
//
//	OPENVIKING_ENDPOINT  - Server URL (default: http://127.0.0.1:1933)
//	OPENVIKING_API_KEY   - API key (required for authenticated servers)
package openviking

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sadewadee/hera/plugins"
)

const defaultEndpoint = "http://127.0.0.1:1933"

// Provider implements plugins.MemoryProvider for OpenViking.
type Provider struct {
	endpoint  string
	apiKey    string
	sessionID string
	turnCount int

	httpClient     *http.Client
	prefetchResult string
	prefetchLock   sync.Mutex
}

// New creates a new OpenViking memory provider.
func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *Provider) Name() string { return "openviking" }

func (p *Provider) IsAvailable() bool {
	return os.Getenv("OPENVIKING_ENDPOINT") != ""
}

func (p *Provider) Initialize(sessionID string) error {
	p.endpoint = os.Getenv("OPENVIKING_ENDPOINT")
	if p.endpoint == "" {
		p.endpoint = defaultEndpoint
	}
	p.apiKey = os.Getenv("OPENVIKING_API_KEY")
	p.sessionID = sessionID
	p.turnCount = 0

	// Health check
	if !p.healthCheck() {
		slog.Warn("openviking server not reachable", "endpoint", p.endpoint)
	}
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	return "# OpenViking Knowledge Base\n" +
		"Active. Endpoint: " + p.endpoint + "\n" +
		"Use viking_search to find information, viking_read for details " +
		"(abstract/overview/full), viking_browse to explore.\n" +
		"Use viking_remember to store facts, viking_add_resource to index URLs/docs."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	p.prefetchLock.Lock()
	result := p.prefetchResult
	p.prefetchResult = ""
	p.prefetchLock.Unlock()
	if result == "" {
		return ""
	}
	return "## OpenViking Context\n" + result
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	p.turnCount++
	go func() {
		p.apiPost(fmt.Sprintf("/api/v1/sessions/%s/messages", p.sessionID), map[string]interface{}{
			"role":    "user",
			"content": truncate(userContent, 4000),
		})
		p.apiPost(fmt.Sprintf("/api/v1/sessions/%s/messages", p.sessionID), map[string]interface{}{
			"role":    "assistant",
			"content": truncate(assistantContent, 4000),
		})
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {
	if p.turnCount == 0 {
		return
	}
	if _, err := p.apiPost(fmt.Sprintf("/api/v1/sessions/%s/commit", p.sessionID), nil); err != nil {
		slog.Warn("openviking session commit failed", "error", err)
	} else {
		slog.Info("openviking session committed", "session", p.sessionID, "turns", p.turnCount)
	}
}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name:        "viking_search",
			Description: "Semantic search over the OpenViking knowledge base.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query."},
					"mode":  map[string]interface{}{"type": "string", "enum": []string{"auto", "fast", "deep"}, "description": "Search depth."},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results (default: 10)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "viking_read",
			Description: "Read content at a viking:// URI. Levels: abstract, overview, full.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"uri":   map[string]interface{}{"type": "string", "description": "viking:// URI to read."},
					"level": map[string]interface{}{"type": "string", "enum": []string{"abstract", "overview", "full"}, "description": "Detail level."},
				},
				"required": []string{"uri"},
			},
		},
		{
			Name:        "viking_browse",
			Description: "Browse the OpenViking knowledge store like a filesystem.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{"type": "string", "enum": []string{"tree", "list", "stat"}, "description": "Browse action."},
					"path":   map[string]interface{}{"type": "string", "description": "Viking URI path."},
				},
				"required": []string{"action"},
			},
		},
		{
			Name:        "viking_remember",
			Description: "Store a fact or memory in the OpenViking knowledge base.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":  map[string]interface{}{"type": "string", "description": "The information to remember."},
					"category": map[string]interface{}{"type": "string", "enum": []string{"preference", "entity", "event", "case", "pattern"}},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "viking_add_resource",
			Description: "Add a URL or document to the OpenViking knowledge base.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":    map[string]interface{}{"type": "string", "description": "URL or path of the resource."},
					"reason": map[string]interface{}{"type": "string", "description": "Why this resource is relevant."},
				},
				"required": []string{"url"},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "viking_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		result, err := p.apiPost("/api/v1/search/find", map[string]interface{}{"query": query})
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "viking_read":
		uri, _ := args["uri"].(string)
		if uri == "" {
			return "", fmt.Errorf("uri is required")
		}
		level, _ := args["level"].(string)
		if level == "" {
			level = "overview"
		}
		endpoint := "/api/v1/content/overview"
		switch level {
		case "abstract":
			endpoint = "/api/v1/content/abstract"
		case "full":
			endpoint = "/api/v1/content/read"
		}
		result, err := p.apiGet(endpoint + "?uri=" + uri)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "viking_browse":
		action, _ := args["action"].(string)
		path, _ := args["path"].(string)
		if path == "" {
			path = "viking://"
		}
		endpoints := map[string]string{
			"tree": "/api/v1/fs/tree",
			"list": "/api/v1/fs/ls",
			"stat": "/api/v1/fs/stat",
		}
		ep, ok := endpoints[action]
		if !ok {
			ep = "/api/v1/fs/ls"
		}
		result, err := p.apiGet(ep + "?uri=" + path)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "viking_remember":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required")
		}
		category, _ := args["category"].(string)
		text := "[Remember] " + content
		if category != "" {
			text = fmt.Sprintf("[Remember -- %s] %s", category, content)
		}
		_, err := p.apiPost(fmt.Sprintf("/api/v1/sessions/%s/messages", p.sessionID), map[string]interface{}{
			"role":    "user",
			"content": text,
		})
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]string{"status": "stored", "message": "Memory recorded."})
		return string(data), nil

	case "viking_add_resource":
		url, _ := args["url"].(string)
		if url == "" {
			return "", fmt.Errorf("url is required")
		}
		payload := map[string]interface{}{"path": url}
		if reason, _ := args["reason"].(string); reason != "" {
			payload["reason"] = reason
		}
		result, err := p.apiPost("/api/v1/resources", payload)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "endpoint", Description: "OpenViking server URL", EnvVar: "OPENVIKING_ENDPOINT"},
		{Key: "api_key", Description: "OpenViking API key", Secret: true, EnvVar: "OPENVIKING_API_KEY"},
	}
}

func (p *Provider) Shutdown() {}

// HTTP helpers

func (p *Provider) healthCheck() bool {
	resp, err := p.httpClient.Get(p.endpoint + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func (p *Provider) apiGet(path string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", p.endpoint+path, nil)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)
	return p.doRequest(req)
}

func (p *Provider) apiPost(path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest("POST", p.endpoint+path, bodyReader)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)
	return p.doRequest(req)
}

func (p *Provider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}
}

func (p *Provider) doRequest(req *http.Request) (map[string]interface{}, error) {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openviking API error %d: %s", resp.StatusCode, string(body))
	}
	var result map[string]interface{}
	if len(body) > 0 {
		json.Unmarshal(body, &result)
	}
	return result, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
