// Package mem0 implements the Mem0 Platform memory provider plugin.
//
// Server-side LLM fact extraction, semantic search with reranking, and
// automatic deduplication via the Mem0 Platform REST API.
//
// Config via environment variables:
//
//	MEM0_API_KEY   - Mem0 Platform API key (required)
//	MEM0_USER_ID   - User identifier (default: hera-user)
//	MEM0_AGENT_ID  - Agent identifier (default: hera)
package mem0

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

const (
	baseURL          = "https://api.mem0.ai/v1"
	breakerThreshold = 5
	breakerCooldown  = 120 * time.Second
)

// Provider implements plugins.MemoryProvider for Mem0 Platform.
type Provider struct {
	apiKey  string
	userID  string
	agentID string
	rerank  bool

	httpClient *http.Client

	prefetchResult string
	prefetchLock   sync.Mutex

	// Circuit breaker
	consecutiveFailures int
	breakerOpenUntil    time.Time
	breakerLock         sync.Mutex
}

// New creates a new Mem0 memory provider.
func New() *Provider {
	return &Provider{
		userID:  "hera-user",
		agentID: "hera",
		rerank:  true,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *Provider) Name() string { return "mem0" }

func (p *Provider) IsAvailable() bool {
	return os.Getenv("MEM0_API_KEY") != ""
}

func (p *Provider) Initialize(sessionID string) error {
	p.apiKey = os.Getenv("MEM0_API_KEY")
	if p.apiKey == "" {
		return fmt.Errorf("MEM0_API_KEY not set")
	}
	if uid := os.Getenv("MEM0_USER_ID"); uid != "" {
		p.userID = uid
	}
	if aid := os.Getenv("MEM0_AGENT_ID"); aid != "" {
		p.agentID = aid
	}
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	return "# Mem0 Memory\n" +
		"Active. User: " + p.userID + ".\n" +
		"Use mem0_search to find memories, mem0_conclude to store facts, " +
		"mem0_profile for a full overview."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	p.prefetchLock.Lock()
	result := p.prefetchResult
	p.prefetchResult = ""
	p.prefetchLock.Unlock()
	if result == "" {
		return ""
	}
	return "## Mem0 Memory\n" + result
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	if p.isBreakerOpen() {
		return
	}

	go func() {
		body := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": userContent},
				{"role": "assistant", "content": assistantContent},
			},
			"user_id":  p.userID,
			"agent_id": p.agentID,
		}
		if _, err := p.apiRequest("POST", "/memories/", body); err != nil {
			p.recordFailure()
			slog.Warn("mem0 sync failed", "error", err)
		} else {
			p.recordSuccess()
		}
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name:        "mem0_profile",
			Description: "Retrieve all stored memories about the user -- preferences, facts, project context.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "mem0_search",
			Description: "Search memories by meaning. Returns relevant facts ranked by similarity.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":  map[string]interface{}{"type": "string", "description": "What to search for."},
					"rerank": map[string]interface{}{"type": "boolean", "description": "Enable reranking for precision (default: false)."},
					"top_k":  map[string]interface{}{"type": "integer", "description": "Max results (default: 10, max: 50)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "mem0_conclude",
			Description: "Store a durable fact about the user. Use for explicit preferences, corrections, or decisions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"conclusion": map[string]interface{}{"type": "string", "description": "The fact to store."},
				},
				"required": []string{"conclusion"},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	if p.isBreakerOpen() {
		return "", fmt.Errorf("mem0 API temporarily unavailable (circuit breaker open)")
	}

	switch toolName {
	case "mem0_profile":
		result, err := p.apiRequest("GET", "/memories/?user_id="+p.userID, nil)
		if err != nil {
			p.recordFailure()
			return "", fmt.Errorf("failed to fetch profile: %w", err)
		}
		p.recordSuccess()
		data, _ := json.Marshal(result)
		return string(data), nil

	case "mem0_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("missing required parameter: query")
		}
		body := map[string]interface{}{
			"query":   query,
			"user_id": p.userID,
			"top_k":   10,
		}
		if topK, ok := args["top_k"].(float64); ok && int(topK) > 0 {
			k := int(topK)
			if k > 50 {
				k = 50
			}
			body["top_k"] = k
		}
		result, err := p.apiRequest("POST", "/memories/search/", body)
		if err != nil {
			p.recordFailure()
			return "", fmt.Errorf("search failed: %w", err)
		}
		p.recordSuccess()
		data, _ := json.Marshal(result)
		return string(data), nil

	case "mem0_conclude":
		conclusion, _ := args["conclusion"].(string)
		if conclusion == "" {
			return "", fmt.Errorf("missing required parameter: conclusion")
		}
		body := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": conclusion},
			},
			"user_id": p.userID,
			"infer":   false,
		}
		_, err := p.apiRequest("POST", "/memories/", body)
		if err != nil {
			p.recordFailure()
			return "", fmt.Errorf("failed to store: %w", err)
		}
		p.recordSuccess()
		data, _ := json.Marshal(map[string]string{"result": "Fact stored."})
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "api_key", Description: "Mem0 Platform API key", Secret: true, EnvVar: "MEM0_API_KEY", URL: "https://app.mem0.ai"},
		{Key: "user_id", Description: "User identifier"},
		{Key: "agent_id", Description: "Agent identifier"},
	}
}

func (p *Provider) Shutdown() {}

// apiRequest performs an HTTP request to the Mem0 API.
func (p *Provider) apiRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mem0 API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *Provider) isBreakerOpen() bool {
	p.breakerLock.Lock()
	defer p.breakerLock.Unlock()
	if p.consecutiveFailures < breakerThreshold {
		return false
	}
	if time.Now().After(p.breakerOpenUntil) {
		p.consecutiveFailures = 0
		return false
	}
	return true
}

func (p *Provider) recordSuccess() {
	p.breakerLock.Lock()
	defer p.breakerLock.Unlock()
	p.consecutiveFailures = 0
}

func (p *Provider) recordFailure() {
	p.breakerLock.Lock()
	defer p.breakerLock.Unlock()
	p.consecutiveFailures++
	if p.consecutiveFailures >= breakerThreshold {
		p.breakerOpenUntil = time.Now().Add(breakerCooldown)
		slog.Warn("mem0 circuit breaker tripped", "failures", p.consecutiveFailures)
	}
}
