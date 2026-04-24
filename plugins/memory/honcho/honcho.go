// Package honcho implements the Honcho AI-native memory provider plugin.
//
// Provides cross-session user modeling with dialectic Q&A, semantic search,
// peer cards, and persistent conclusions via the Honcho REST API.
//
// The 4 tools (profile, search, context, conclude) are exposed through
// the MemoryProvider interface.
//
// Config uses the existing Honcho config chain:
//
//	1. $HERA_HOME/honcho.json (profile-scoped)
//	2. ~/.honcho/config.json (legacy global)
//	3. Environment variables
package honcho

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/sadewadee/hera/plugins"
)

// Provider implements plugins.MemoryProvider for Honcho memory.
type Provider struct {
	mu        sync.Mutex
	client    *Client
	session   *SessionManager
	config    *ClientConfig
	sessionID string

	// recallMode controls how memory retrieval works:
	//   "hybrid"  - auto-injected context + tools (default)
	//   "context" - auto-injected context only, tools hidden
	//   "tools"   - tools only, no auto-injection
	recallMode string

	// Prefetch state
	prefetchResult string
	prefetchLock   sync.Mutex

	// First-turn context baking
	firstTurnContext *string
	firstTurnLock    sync.Mutex
}

// New creates a new Honcho memory provider.
func New() *Provider {
	return &Provider{
		recallMode: "hybrid",
	}
}

func (p *Provider) Name() string { return "honcho" }

func (p *Provider) IsAvailable() bool {
	apiKey := os.Getenv("HONCHO_API_KEY")
	baseURL := os.Getenv("HONCHO_BASE_URL")
	return apiKey != "" || baseURL != ""
}

func (p *Provider) Initialize(sessionID string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading honcho config: %w", err)
	}
	if !cfg.Enabled {
		slog.Debug("honcho not configured, plugin inactive")
		return nil
	}

	p.config = cfg
	p.sessionID = sessionID
	p.recallMode = cfg.RecallMode

	client, err := NewClient(cfg)
	if err != nil {
		return fmt.Errorf("creating honcho client: %w", err)
	}
	p.client = client

	p.session = NewSessionManager(client, cfg)

	// Create session eagerly for context/hybrid modes
	if p.recallMode != "tools" {
		_, err := p.session.GetOrCreate(sessionID)
		if err != nil {
			slog.Warn("honcho session creation failed", "error", err)
		}
	}

	return nil
}

func (p *Provider) SystemPromptBlock() string {
	if p.client == nil {
		return ""
	}

	switch p.recallMode {
	case "context":
		return "# Honcho Memory\n" +
			"Active (context-injection mode). Relevant user context is automatically " +
			"injected before each turn. No memory tools are available."
	case "tools":
		return "# Honcho Memory\n" +
			"Active (tools-only mode). Use honcho_profile for a quick factual snapshot, " +
			"honcho_search for raw excerpts, honcho_context for synthesized answers, " +
			"honcho_conclude to save facts about the user."
	default: // hybrid
		return "# Honcho Memory\n" +
			"Active (hybrid mode). Relevant context is auto-injected AND memory tools " +
			"are available. Use honcho_profile, honcho_search, honcho_context, honcho_conclude."
	}
}

func (p *Provider) Prefetch(query, sessionID string) string {
	if p.client == nil || p.recallMode == "tools" {
		return ""
	}

	p.prefetchLock.Lock()
	result := p.prefetchResult
	p.prefetchResult = ""
	p.prefetchLock.Unlock()

	if result == "" {
		return ""
	}
	return "## Honcho Context\n" + result
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	if p.client == nil || p.session == nil {
		return
	}

	go func() {
		sess, err := p.session.GetOrCreate(p.sessionID)
		if err != nil {
			slog.Debug("honcho sync_turn: session error", "error", err)
			return
		}
		sess.AddMessage("user", userContent)
		sess.AddMessage("assistant", assistantContent)
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {
	if action != "add" || target != "user" || content == "" {
		return
	}
	if p.client == nil || p.session == nil {
		return
	}

	go func() {
		if err := p.session.CreateConclusion(p.sessionID, content); err != nil {
			slog.Debug("honcho memory mirror failed", "error", err)
		}
	}()
}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {
	if p.session != nil {
		p.session.FlushAll()
	}
}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	if p.client == nil || p.recallMode == "context" {
		return nil
	}

	return []plugins.ToolSchema{
		{
			Name: "honcho_profile",
			Description: "Retrieve the user's peer card from Honcho -- a curated list of key facts " +
				"about them. Fast, no LLM reasoning, minimal cost.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name: "honcho_search",
			Description: "Semantic search over Honcho's stored context about the user. " +
				"Returns raw excerpts ranked by relevance.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":      map[string]interface{}{"type": "string", "description": "What to search for in Honcho's memory."},
					"max_tokens": map[string]interface{}{"type": "integer", "description": "Token budget for returned context (default 800, max 2000)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name: "honcho_context",
			Description: "Ask Honcho a natural language question and get a synthesized answer. " +
				"Uses Honcho's LLM (dialectic reasoning).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "A natural language question."},
					"peer":  map[string]interface{}{"type": "string", "description": "Which peer to query about: 'user' (default) or 'ai'."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name: "honcho_conclude",
			Description: "Write a conclusion about the user back to Honcho's memory. " +
				"Conclusions are persistent facts that build the user's profile.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"conclusion": map[string]interface{}{"type": "string", "description": "A factual statement about the user to persist."},
				},
				"required": []string{"conclusion"},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	if p.client == nil || p.session == nil {
		return "", fmt.Errorf("honcho is not active for this session")
	}

	switch toolName {
	case "honcho_profile":
		card, err := p.session.GetPeerCard(p.sessionID)
		if err != nil {
			return "", fmt.Errorf("honcho profile failed: %w", err)
		}
		if card == "" {
			card = "No profile facts available yet."
		}
		data, _ := json.Marshal(map[string]string{"result": card})
		return string(data), nil

	case "honcho_search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("missing required parameter: query")
		}
		maxTokens := 800
		if mt, ok := args["max_tokens"].(float64); ok && int(mt) > 0 {
			maxTokens = int(mt)
			if maxTokens > 2000 {
				maxTokens = 2000
			}
		}
		result, err := p.session.SearchContext(p.sessionID, query, maxTokens)
		if err != nil {
			return "", fmt.Errorf("honcho search failed: %w", err)
		}
		if result == "" {
			result = "No relevant context found."
		}
		data, _ := json.Marshal(map[string]string{"result": result})
		return string(data), nil

	case "honcho_context":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("missing required parameter: query")
		}
		peer, _ := args["peer"].(string)
		if peer == "" {
			peer = "user"
		}
		result, err := p.session.DialecticQuery(p.sessionID, query, peer)
		if err != nil {
			return "", fmt.Errorf("honcho context failed: %w", err)
		}
		if result == "" {
			result = "No result from Honcho."
		}
		data, _ := json.Marshal(map[string]string{"result": result})
		return string(data), nil

	case "honcho_conclude":
		conclusion, _ := args["conclusion"].(string)
		if conclusion == "" {
			return "", fmt.Errorf("missing required parameter: conclusion")
		}
		if err := p.session.CreateConclusion(p.sessionID, conclusion); err != nil {
			return "", fmt.Errorf("honcho conclude failed: %w", err)
		}
		data, _ := json.Marshal(map[string]string{"result": "Conclusion saved: " + conclusion})
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "api_key", Description: "Honcho API key", Secret: true, EnvVar: "HONCHO_API_KEY", URL: "https://app.honcho.dev"},
		{Key: "base_url", Description: "Honcho base URL (for self-hosted)"},
	}
}

func (p *Provider) Shutdown() {
	if p.session != nil {
		p.session.FlushAll()
	}
}

// getStringArg is a helper to extract string args from tool call maps.
func getStringArg(args map[string]interface{}, key string) string {
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}
