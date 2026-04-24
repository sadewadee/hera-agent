package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// MemorySaveTool saves a fact to long-term memory.
type MemorySaveTool struct {
	manager *memory.Manager
}

type memorySaveArgs struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	UserID string `json:"user_id,omitempty"`
}

func (m *MemorySaveTool) Name() string {
	return "memory_save"
}

func (m *MemorySaveTool) Description() string {
	return "Saves a fact to long-term memory. Use this to remember important information about the user or conversation."
}

func (m *MemorySaveTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"key": {
				"type": "string",
				"description": "A short label for the fact (e.g. 'name', 'language', 'preference')."
			},
			"value": {
				"type": "string",
				"description": "The fact value to remember."
			},
			"user_id": {
				"type": "string",
				"description": "User identifier. Defaults to 'default' if not provided."
			}
		},
		"required": ["key", "value"]
	}`)
}

func (m *MemorySaveTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params memorySaveArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Key == "" || params.Value == "" {
		return &tools.Result{Content: "both key and value are required", IsError: true}, nil
	}

	userID := resolveNoteUserID(ctx, params.UserID)

	if err := m.manager.SaveFact(ctx, userID, params.Key, params.Value); err != nil {
		return &tools.Result{Content: fmt.Sprintf("save fact: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("remembered: %s = %s", params.Key, params.Value)}, nil
}

// MemorySearchTool searches long-term memory.
type MemorySearchTool struct {
	manager *memory.Manager
}

type memorySearchArgs struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

func (m *MemorySearchTool) Name() string {
	return "memory_search"
}

func (m *MemorySearchTool) Description() string {
	return "Searches long-term memory for relevant facts and past conversations."
}

func (m *MemorySearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query to find relevant memories."
			},
			"limit": {
				"type": "integer",
				"description": "Maximum number of results to return. Defaults to 5."
			},
			"user_id": {
				"type": "string",
				"description": "User identifier. Defaults to 'default' if not provided."
			}
		},
		"required": ["query"]
	}`)
}

func (m *MemorySearchTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params memorySearchArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Query == "" {
		return &tools.Result{Content: "query is required", IsError: true}, nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 5
	}

	userID := resolveNoteUserID(ctx, params.UserID)

	results, err := m.manager.Search(ctx, params.Query, memory.SearchOpts{
		Limit:  limit,
		UserID: userID,
	})
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("search memory: %v", err), IsError: true}, nil
	}

	if len(results) == 0 {
		return &tools.Result{Content: "no memories found matching the query"}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d memories:\n\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (score: %.2f)\n", i+1, r.Source, r.Content, r.Score))
	}

	return &tools.Result{Content: sb.String()}, nil
}

// RegisterMemory registers memory_save and memory_search tools with the given registry.
func RegisterMemory(registry *tools.Registry, mgr *memory.Manager) {
	registry.Register(&MemorySaveTool{manager: mgr})
	registry.Register(&MemorySearchTool{manager: mgr})
}
