// Package retaindb implements the RetainDB memory provider plugin.
//
// Cross-session memory via RetainDB cloud API with semantic search,
// user profile retrieval, dialectic synthesis, and shared file store.
//
// Config (env vars):
//
//	RETAINDB_API_KEY   - API key (required)
//	RETAINDB_BASE_URL  - API endpoint (default: https://api.retaindb.com)
//	RETAINDB_PROJECT   - Project identifier (optional)
package retaindb

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

const defaultBaseURL = "https://api.retaindb.com"

// Provider implements plugins.MemoryProvider for RetainDB.
type Provider struct {
	apiKey    string
	baseURL   string
	project   string
	userID    string
	sessionID string

	httpClient *http.Client

	contextResult   string
	dialecticResult string
	mu              sync.Mutex
}

// New creates a new RetainDB memory provider.
func New() *Provider {
	return &Provider{
		userID:  "default",
		project: "default",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *Provider) Name() string { return "retaindb" }

func (p *Provider) IsAvailable() bool {
	return os.Getenv("RETAINDB_API_KEY") != ""
}

func (p *Provider) Initialize(sessionID string) error {
	p.apiKey = os.Getenv("RETAINDB_API_KEY")
	if p.apiKey == "" {
		return fmt.Errorf("RETAINDB_API_KEY not set")
	}

	p.baseURL = strings.TrimRight(os.Getenv("RETAINDB_BASE_URL"), "/")
	if p.baseURL == "" {
		p.baseURL = defaultBaseURL
	}

	if proj := os.Getenv("RETAINDB_PROJECT"); proj != "" {
		p.project = proj
	}

	p.sessionID = sessionID
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	return "# RetainDB Memory\n" +
		"Active. Project: " + p.project + ".\n" +
		"Use retaindb_search to find memories, retaindb_remember to store facts, " +
		"retaindb_profile for a user overview, retaindb_context for current-task context."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	p.mu.Lock()
	context := p.contextResult
	dialectic := p.dialecticResult
	p.contextResult = ""
	p.dialecticResult = ""
	p.mu.Unlock()

	var parts []string
	if context != "" {
		parts = append(parts, context)
	}
	if dialectic != "" {
		parts = append(parts, "[RetainDB User Synthesis]\n"+dialectic)
	}
	return strings.Join(parts, "\n\n")
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	if userContent == "" {
		return
	}
	go func() {
		body := map[string]interface{}{
			"project":    p.project,
			"session_id": p.sessionID,
			"user_id":    p.userID,
			"messages": []map[string]string{
				{"role": "user", "content": userContent},
				{"role": "assistant", "content": assistantContent},
			},
			"write_mode": "sync",
		}
		if _, err := p.apiRequest("POST", "/v1/memory/ingest/session", body); err != nil {
			slog.Warn("retaindb sync failed", "error", err)
		}
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {
	if action != "add" || content == "" {
		return
	}
	go func() {
		memType := "factual"
		if target == "user" {
			memType = "preference"
		}
		p.apiRequest("POST", "/v1/memory", map[string]interface{}{
			"project":     p.project,
			"content":     content,
			"memory_type": memType,
			"user_id":     p.userID,
			"session_id":  p.sessionID,
			"importance":  0.7,
		})
	}()
}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name:        "retaindb_profile",
			Description: "Get the user's stable profile from long-term memory.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
		{
			Name:        "retaindb_search",
			Description: "Semantic search across stored memories.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "What to search for."},
					"top_k": map[string]interface{}{"type": "integer", "description": "Max results (default: 8)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "retaindb_context",
			Description: "Synthesized context block for the current task.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Current task or question."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "retaindb_remember",
			Description: "Persist an explicit fact to long-term memory.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":     map[string]interface{}{"type": "string", "description": "The fact to remember."},
					"memory_type": map[string]interface{}{"type": "string", "enum": []string{"factual", "preference", "goal", "instruction"}},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "retaindb_forget",
			Description: "Delete a specific memory by ID.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{"type": "string", "description": "Memory ID to delete."},
				},
				"required": []string{"memory_id"},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "retaindb_profile":
		result, err := p.apiRequest("GET", "/v1/memory/profile/"+p.userID+"?project="+p.project, nil)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "retaindb_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		topK := 8
		if k, ok := args["top_k"].(float64); ok && int(k) > 0 {
			topK = int(k)
			if topK > 20 {
				topK = 20
			}
		}
		result, err := p.apiRequest("POST", "/v1/memory/search", map[string]interface{}{
			"project":    p.project,
			"query":      query,
			"user_id":    p.userID,
			"session_id": p.sessionID,
			"top_k":      topK,
		})
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "retaindb_context":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		result, err := p.apiRequest("POST", "/v1/context/query", map[string]interface{}{
			"project":    p.project,
			"query":      query,
			"user_id":    p.userID,
			"session_id": p.sessionID,
		})
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "retaindb_remember":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required")
		}
		memType, _ := args["memory_type"].(string)
		if memType == "" {
			memType = "factual"
		}
		result, err := p.apiRequest("POST", "/v1/memory", map[string]interface{}{
			"project":     p.project,
			"content":     content,
			"memory_type": memType,
			"user_id":     p.userID,
			"session_id":  p.sessionID,
			"importance":  0.7,
		})
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(result)
		return string(data), nil

	case "retaindb_forget":
		memID, _ := args["memory_id"].(string)
		if memID == "" {
			return "", fmt.Errorf("memory_id is required")
		}
		_, err := p.apiRequest("DELETE", "/v1/memory/"+memID, nil)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]string{"result": "Memory deleted."})
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "api_key", Description: "RetainDB API key", Secret: true, EnvVar: "RETAINDB_API_KEY", URL: "https://retaindb.com"},
		{Key: "base_url", Description: "API endpoint"},
		{Key: "project", Description: "Project identifier"},
	}
}

func (p *Provider) Shutdown() {}

func (p *Provider) apiRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	token := strings.TrimPrefix(p.apiKey, "Bearer ")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("retaindb API error %d: %s", resp.StatusCode, string(respBody))
	}
	var result map[string]interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &result)
	}
	return result, nil
}
