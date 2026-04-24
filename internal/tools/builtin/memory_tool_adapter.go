package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/plugins"
)

// memoryToolAdapter bridges a single tool schema exposed by a
// plugins.MemoryProvider (hindsight_recall, mem0_search, etc.) to the
// tools.Tool interface so it can be registered and invoked by the agent.
type memoryToolAdapter struct {
	name        string
	description string
	parameters  json.RawMessage
	provider    plugins.MemoryProvider
}

func (a *memoryToolAdapter) Name() string                { return a.name }
func (a *memoryToolAdapter) Description() string         { return a.description }
func (a *memoryToolAdapter) Parameters() json.RawMessage { return a.parameters }

func (a *memoryToolAdapter) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var argMap map[string]interface{}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &argMap); err != nil {
			return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
		}
	}
	result, err := a.provider.HandleToolCall(a.name, argMap)
	if err != nil {
		return &tools.Result{Content: err.Error(), IsError: true}, nil
	}
	return &tools.Result{Content: result}, nil
}

// RegisterMemoryProviderTools harvests tool schemas from a MemoryProvider
// and registers each as a tools.Tool in the given registry. Plugin providers
// like hindsight expose real tools (hindsight_recall, hindsight_memorize)
// that would otherwise be unreachable from LLM tool calls.
//
// Nil provider or empty schema list is a no-op.
func RegisterMemoryProviderTools(reg *tools.Registry, provider plugins.MemoryProvider) {
	if provider == nil {
		return
	}
	for _, s := range provider.GetToolSchemas() {
		params, err := json.Marshal(s.Parameters)
		if err != nil {
			// Skip this tool; log elsewhere would be noisy. Parameters is
			// developer-authored static data, so a Marshal error here means
			// the plugin is broken — fail-soft rather than crash startup.
			continue
		}
		reg.Register(&memoryToolAdapter{
			name:        s.Name,
			description: s.Description,
			parameters:  params,
			provider:    provider,
		})
	}
}
