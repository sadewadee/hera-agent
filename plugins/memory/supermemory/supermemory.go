// Package supermemory implements the Supermemory memory provider plugin.
//
// Provides semantic long-term memory with profile recall, semantic search,
// explicit memory tools, turn capture, and session-end conversation ingest
// via the Supermemory REST API.
//
// Config via environment variables:
//
//	SUPERMEMORY_API_KEY       - Supermemory API key (required)
//	SUPERMEMORY_CONTAINER_TAG - Container tag (default: hera)
package supermemory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sadewadee/hera/plugins"
)

const (
	apiBaseURL          = "https://api.supermemory.ai/v3"
	conversationsURL    = "https://api.supermemory.ai/v4/conversations"
	defaultContainerTag = "hera"
)

// Provider implements plugins.MemoryProvider for Supermemory.
type Provider struct {
	apiKey       string
	containerTag string
	sessionID    string
	turnCount    int
	active       bool

	httpClient   *http.Client
	syncThread   sync.Mutex
	prefetchLock sync.Mutex
}

// New creates a new Supermemory memory provider.
func New() *Provider {
	return &Provider{
		containerTag: defaultContainerTag,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Provider) Name() string { return "supermemory" }

func (p *Provider) IsAvailable() bool {
	return os.Getenv("SUPERMEMORY_API_KEY") != ""
}

func (p *Provider) Initialize(sessionID string) error {
	p.apiKey = os.Getenv("SUPERMEMORY_API_KEY")
	if p.apiKey == "" {
		return fmt.Errorf("SUPERMEMORY_API_KEY not set")
	}

	if tag := os.Getenv("SUPERMEMORY_CONTAINER_TAG"); tag != "" {
		p.containerTag = tag
	}

	p.sessionID = sessionID
	p.turnCount = 0
	p.active = true
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	if !p.active {
		return ""
	}
	return "# Supermemory\n" +
		"Active. Container: " + p.containerTag + ".\n" +
		"Use supermemory_search, supermemory_store, supermemory_forget, " +
		"and supermemory_profile for explicit memory operations."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	if !p.active || query == "" {
		return ""
	}

	// Synchronous profile+search for first turn, search-only otherwise
	result, err := p.apiPost("/search/memories", map[string]interface{}{
		"q":             query,
		"container_tag": p.containerTag,
		"limit":         5,
	})
	if err != nil {
		slog.Debug("supermemory prefetch failed", "error", err)
		return ""
	}

	results, _ := result["results"].([]interface{})
	if len(results) == 0 {
		return ""
	}

	var lines []string
	for _, item := range results {
		if m, ok := item.(map[string]interface{}); ok {
			if mem, _ := m["memory"].(string); mem != "" {
				lines = append(lines, "- "+mem)
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "<supermemory-context>\n## Relevant Memories\n" + strings.Join(lines, "\n") + "\n</supermemory-context>"
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	if !p.active || userContent == "" || assistantContent == "" {
		return
	}
	p.turnCount++

	content := fmt.Sprintf("[role: user]\n%s\n[user:end]\n\n[role: assistant]\n%s\n[assistant:end]",
		userContent, assistantContent)

	go func() {
		p.apiPost("/documents/add", map[string]interface{}{
			"content":        content,
			"container_tags": []string{p.containerTag},
			"metadata":       map[string]string{"source": "hera", "type": "conversation_turn"},
		})
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {
	if !p.active || action != "add" || content == "" {
		return
	}
	go func() {
		p.apiPost("/documents/add", map[string]interface{}{
			"content":        strings.TrimSpace(content),
			"container_tags": []string{p.containerTag},
			"metadata":       map[string]string{"source": "hera_memory", "target": target, "type": "explicit_memory"},
		})
	}()
}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {
	if !p.active || p.sessionID == "" || len(messages) == 0 {
		return
	}

	var cleaned []map[string]string
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		if role == "user" || role == "assistant" {
			if content = strings.TrimSpace(content); content != "" {
				cleaned = append(cleaned, map[string]string{"role": role, "content": content})
			}
		}
	}
	if len(cleaned) == 0 {
		return
	}

	body := map[string]interface{}{
		"conversationId": p.sessionID,
		"messages":       cleaned,
		"containerTags":  []string{p.containerTag},
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", conversationsURL, bytes.NewReader(data))
	if err != nil {
		slog.Warn("supermemory session ingest failed", "error", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		slog.Warn("supermemory session ingest failed", "error", err)
		return
	}
	resp.Body.Close()
}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name:        "supermemory_store",
			Description: "Store an explicit memory for future recall.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string", "description": "The memory content to store."},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "supermemory_search",
			Description: "Search long-term memory by semantic similarity.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "What to search for."},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results (1-20)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "supermemory_forget",
			Description: "Forget a memory by exact id or by best-match query.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":    map[string]interface{}{"type": "string", "description": "Exact memory id to delete."},
					"query": map[string]interface{}{"type": "string", "description": "Query to find the memory to forget."},
				},
			},
		},
		{
			Name:        "supermemory_profile",
			Description: "Retrieve persistent profile facts and recent memory context.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Optional focus query."},
				},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	if !p.active {
		return "", fmt.Errorf("supermemory is not configured")
	}

	switch toolName {
	case "supermemory_store":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required")
		}
		result, err := p.apiPost("/documents/add", map[string]interface{}{
			"content":        strings.TrimSpace(content),
			"container_tags": []string{p.containerTag},
		})
		if err != nil {
			return "", fmt.Errorf("failed to store: %w", err)
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "supermemory_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		limit := 5
		if l, ok := args["limit"].(float64); ok && int(l) > 0 {
			limit = int(l)
			if limit > 20 {
				limit = 20
			}
		}
		result, err := p.apiPost("/search/memories", map[string]interface{}{
			"q":             query,
			"container_tag": p.containerTag,
			"limit":         limit,
		})
		if err != nil {
			return "", fmt.Errorf("search failed: %w", err)
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "supermemory_forget":
		id, _ := args["id"].(string)
		query, _ := args["query"].(string)
		if id == "" && query == "" {
			return "", fmt.Errorf("provide either id or query")
		}
		if id != "" {
			_, err := p.apiPost("/memories/forget", map[string]interface{}{
				"container_tag": p.containerTag,
				"id":            id,
			})
			if err != nil {
				return "", err
			}
			data, _ := json.Marshal(map[string]interface{}{"forgotten": true, "id": id})
			return string(data), nil
		}
		return "", fmt.Errorf("query-based forget not supported; provide 'id' parameter instead")

	case "supermemory_profile":
		query, _ := args["query"].(string)
		payload := map[string]interface{}{"container_tag": p.containerTag}
		if query != "" {
			payload["q"] = query
		}
		result, err := p.apiPost("/profile", payload)
		if err != nil {
			return "", fmt.Errorf("profile failed: %w", err)
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "api_key", Description: "Supermemory API key", Secret: true, EnvVar: "SUPERMEMORY_API_KEY", URL: "https://supermemory.ai"},
	}
}

func (p *Provider) Shutdown() {}

func (p *Provider) apiPost(path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiBaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("supermemory API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &result)
	}
	return result, nil
}
