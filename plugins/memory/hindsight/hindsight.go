// Package hindsight implements the Hindsight memory provider plugin.
//
// Long-term memory with knowledge graph, entity resolution, and multi-strategy
// retrieval. Supports cloud (API key) and local modes.
//
// Config via environment variables:
//
//	HINDSIGHT_API_KEY   - API key for Hindsight Cloud
//	HINDSIGHT_BANK_ID   - memory bank identifier (default: hera)
//	HINDSIGHT_BUDGET    - recall budget: low/mid/high (default: mid)
//	HINDSIGHT_API_URL   - API endpoint
//	HINDSIGHT_MODE      - cloud or local (default: cloud)
package hindsight

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
	defaultAPIURL   = "https://api.hindsight.vectorize.io"
	defaultLocalURL = "http://localhost:8888"
	defaultBudget   = "mid"
	defaultBankID   = "hera"
	requestTimeout  = 30 * time.Second
)

// Provider implements plugins.MemoryProvider for Hindsight.
type Provider struct {
	apiURL    string
	apiKey    string
	bankID    string
	budget    string
	mode      string
	sessionID string
	client    *http.Client
	mu        sync.Mutex
}

// New creates a new Hindsight memory provider.
func New() *Provider {
	return &Provider{
		client: &http.Client{Timeout: requestTimeout},
	}
}

func (p *Provider) Name() string { return "hindsight" }

func (p *Provider) IsAvailable() bool {
	apiKey := os.Getenv("HINDSIGHT_API_KEY")
	mode := os.Getenv("HINDSIGHT_MODE")
	return apiKey != "" || mode == "local"
}

func (p *Provider) Initialize(sessionID string) error {
	p.sessionID = sessionID
	p.apiKey = os.Getenv("HINDSIGHT_API_KEY")
	p.bankID = getEnvDefault("HINDSIGHT_BANK_ID", defaultBankID)
	p.budget = getEnvDefault("HINDSIGHT_BUDGET", defaultBudget)
	p.mode = getEnvDefault("HINDSIGHT_MODE", "cloud")

	if p.mode == "local" {
		p.apiURL = getEnvDefault("HINDSIGHT_API_URL", defaultLocalURL)
	} else {
		p.apiURL = getEnvDefault("HINDSIGHT_API_URL", defaultAPIURL)
	}
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	return "# Hindsight Memory\n" +
		"Long-term memory with knowledge graph and entity resolution.\n" +
		"Use hindsight_recall to retrieve past knowledge, hindsight_memorize to store facts."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	if len(strings.TrimSpace(query)) < 10 {
		return ""
	}
	result, err := p.recall(query, 5)
	if err != nil {
		slog.Debug("hindsight prefetch failed", "error", err)
		return ""
	}
	if result == "" {
		return ""
	}
	return "## Hindsight Context\n" + result
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	if len(strings.TrimSpace(userContent)) < 10 {
		return
	}
	go func() {
		content := fmt.Sprintf("User: %.2000s\nAssistant: %.2000s", userContent, assistantContent)
		if err := p.memorize(content); err != nil {
			slog.Debug("hindsight sync failed", "error", err)
		}
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {
	if action != "add" && action != "replace" {
		return
	}
	go func() {
		p.memorize(content)
	}()
}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {
	if len(messages) == 0 {
		return
	}
	// Extract key facts from the session before it ends.
	var parts []string
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		if content != "" && (role == "user" || role == "assistant") {
			text := content
			if len(text) > 500 {
				text = text[:500]
			}
			parts = append(parts, fmt.Sprintf("%s: %s", role, text))
		}
	}
	if len(parts) > 0 {
		combined := strings.Join(parts, "\n")
		if len(combined) > 5000 {
			combined = combined[:5000]
		}
		p.memorize("[Session summary]\n" + combined)
	}
}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name:        "hindsight_recall",
			Description: "Search Hindsight's knowledge graph for relevant context from past sessions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "What to recall."},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results (default: 5)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "hindsight_memorize",
			Description: "Store important information in Hindsight's knowledge graph for future recall.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string", "description": "The information to remember."},
				},
				"required": []string{"content"},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "hindsight_recall":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		limit := 5
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		result, err := p.recall(query, limit)
		if err != nil {
			return "", err
		}
		if result == "" {
			result = "No relevant memories found."
		}
		data, _ := json.Marshal(map[string]string{"result": result})
		return string(data), nil

	case "hindsight_memorize":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required")
		}
		if err := p.memorize(content); err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]string{"result": "Memorized successfully."})
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "api_key", Description: "Hindsight API key", Secret: true, EnvVar: "HINDSIGHT_API_KEY"},
		{Key: "bank_id", Description: "Memory bank identifier", EnvVar: "HINDSIGHT_BANK_ID"},
		{Key: "budget", Description: "Recall budget: low/mid/high", EnvVar: "HINDSIGHT_BUDGET"},
	}
}

func (p *Provider) Shutdown() {}

// --- Internal API methods ---

func (p *Provider) recall(query string, limit int) (string, error) {
	body := map[string]interface{}{
		"query":   query,
		"bank_id": p.bankID,
		"budget":  p.budget,
		"limit":   limit,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", p.apiURL+"/v1/recall", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("hindsight recall: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hindsight recall: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}

	if memories, ok := result["memories"]; ok {
		data, _ := json.MarshalIndent(memories, "", "  ")
		return string(data), nil
	}
	return string(respBody), nil
}

func (p *Provider) memorize(content string) error {
	body := map[string]interface{}{
		"content": content,
		"bank_id": p.bankID,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", p.apiURL+"/v1/memorize", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("hindsight memorize: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hindsight memorize: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func getEnvDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
