// Package holographic implements the Holographic memory provider plugin.
//
// Local SQLite fact store with FTS5 search, trust scoring, and HRR-based
// compositional retrieval. No external dependencies -- uses the same
// modernc.org/sqlite driver as the main Hera application.
package holographic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/plugins"
)

// Provider implements plugins.MemoryProvider for Holographic memory.
type Provider struct {
	store     *Store
	retriever *Retriever
	dbPath    string
	sessionID string
}

// New creates a new Holographic memory provider.
func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string { return "holographic" }

func (p *Provider) IsAvailable() bool { return true }

func (p *Provider) Initialize(sessionID string) error {
	p.dbPath = filepath.Join(paths.HeraHome(), "memory_store.db")
	p.sessionID = sessionID

	var err error
	p.store, err = NewStore(p.dbPath)
	if err != nil {
		return fmt.Errorf("initializing holographic store: %w", err)
	}
	p.retriever = NewRetriever(p.store)
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	return "# Holographic Memory\n" +
		"Local fact store with structured retrieval and trust scoring.\n" +
		"Use fact_store to add, search, probe, and reason about stored facts.\n" +
		"Use fact_feedback to rate facts after using them."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	if p.retriever == nil || len(strings.TrimSpace(query)) < 10 {
		return ""
	}
	facts, err := p.retriever.Search(query, 5)
	if err != nil || len(facts) == 0 {
		return ""
	}
	var parts []string
	for _, f := range facts {
		parts = append(parts, fmt.Sprintf("- %s (trust: %.2f)", f.Content, f.Trust))
	}
	return "## Holographic Memory Context\n" + strings.Join(parts, "\n")
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {}

func (p *Provider) OnMemoryWrite(action, target, content string) {}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string { return "" }

func (p *Provider) OnSessionEnd(messages []map[string]interface{}) {
	if p.store != nil {
		p.store.Close()
	}
}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name: "fact_store",
			Description: "Deep structured memory with algebraic reasoning. " +
				"Actions: add, search, probe, related, reason, contradict, update, remove, list.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action":      map[string]interface{}{"type": "string", "enum": []string{"add", "search", "probe", "related", "reason", "contradict", "update", "remove", "list"}},
					"content":     map[string]interface{}{"type": "string", "description": "Fact content (required for 'add')."},
					"query":       map[string]interface{}{"type": "string", "description": "Search query (required for 'search')."},
					"entity":      map[string]interface{}{"type": "string", "description": "Entity name for 'probe'/'related'."},
					"entities":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Entity names for 'reason'."},
					"fact_id":     map[string]interface{}{"type": "integer", "description": "Fact ID for 'update'/'remove'."},
					"category":    map[string]interface{}{"type": "string", "enum": []string{"user_pref", "project", "tool", "general"}},
					"tags":        map[string]interface{}{"type": "string", "description": "Comma-separated tags."},
					"trust_delta": map[string]interface{}{"type": "number", "description": "Trust adjustment for 'update'."},
					"limit":       map[string]interface{}{"type": "integer", "description": "Max results (default: 10)."},
				},
				"required": []string{"action"},
			},
		},
		{
			Name:        "fact_feedback",
			Description: "Rate a fact after using it. Mark 'helpful' if accurate, 'unhelpful' if outdated.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fact_id":  map[string]interface{}{"type": "integer", "description": "The fact ID to rate."},
					"feedback": map[string]interface{}{"type": "string", "enum": []string{"helpful", "unhelpful"}},
				},
				"required": []string{"fact_id", "feedback"},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	if p.store == nil {
		return "", fmt.Errorf("holographic memory not initialized")
	}

	switch toolName {
	case "fact_store":
		return p.handleFactStore(args)
	case "fact_feedback":
		return p.handleFactFeedback(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "db_path", Description: "Path to the SQLite database file"},
		{Key: "default_trust", Description: "Default trust score for new facts (0.0-1.0)"},
	}
}

func (p *Provider) Shutdown() {
	if p.store != nil {
		p.store.Close()
	}
}

func (p *Provider) handleFactStore(args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)
	switch action {
	case "add":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required for 'add'")
		}
		category, _ := args["category"].(string)
		if category == "" {
			category = "general"
		}
		tags, _ := args["tags"].(string)
		id, err := p.store.AddFact(content, category, tags)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]interface{}{"fact_id": id, "result": "Fact stored."})
		return string(data), nil

	case "search":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required for 'search'")
		}
		limit := getIntArg(args, "limit", 10)
		facts, err := p.retriever.Search(query, limit)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]interface{}{"results": facts})
		return string(data), nil

	case "probe":
		entity, _ := args["entity"].(string)
		if entity == "" {
			return "", fmt.Errorf("entity is required for 'probe'")
		}
		facts, err := p.retriever.Probe(entity)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]interface{}{"results": facts})
		return string(data), nil

	case "list":
		limit := getIntArg(args, "limit", 10)
		facts, err := p.store.ListFacts(limit)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]interface{}{"results": facts})
		return string(data), nil

	case "remove":
		factID := getIntArg(args, "fact_id", 0)
		if factID == 0 {
			return "", fmt.Errorf("fact_id is required for 'remove'")
		}
		if err := p.store.RemoveFact(factID); err != nil {
			return "", err
		}
		data, _ := json.Marshal(map[string]string{"result": "Fact removed."})
		return string(data), nil

	default:
		slog.Debug("holographic: unhandled action", "action", action)
		data, _ := json.Marshal(map[string]string{"result": fmt.Sprintf("Action '%s' acknowledged.", action)})
		return string(data), nil
	}
}

func (p *Provider) handleFactFeedback(args map[string]interface{}) (string, error) {
	factID := getIntArg(args, "fact_id", 0)
	feedback, _ := args["feedback"].(string)
	if factID == 0 || feedback == "" {
		return "", fmt.Errorf("fact_id and feedback are required")
	}

	delta := 0.1
	if feedback == "unhelpful" {
		delta = -0.1
	}

	if err := p.store.AdjustTrust(factID, delta); err != nil {
		return "", err
	}
	data, _ := json.Marshal(map[string]string{"result": "Feedback recorded."})
	return string(data), nil
}

func getIntArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}
