package builtin

import (
	"context"
	"encoding/json"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/plugins"
)

// engineToolAdapter bridges a single tool schema exposed by a
// plugins.ContextEngine to the tools.Tool interface so it can be
// registered in the tool registry and invoked by the agent.
type engineToolAdapter struct {
	name        string
	description string
	parameters  json.RawMessage
	engine      plugins.ContextEngine
}

func (a *engineToolAdapter) Name() string                { return a.name }
func (a *engineToolAdapter) Description() string         { return a.description }
func (a *engineToolAdapter) Parameters() json.RawMessage { return a.parameters }

func (a *engineToolAdapter) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	result, err := a.engine.HandleToolCall(ctx, a.name, args)
	if err != nil {
		return &tools.Result{Content: err.Error(), IsError: true}, nil
	}
	return &tools.Result{Content: result}, nil
}

// RegisterEngineTools harvests tool schemas from a ContextEngine and
// registers each as a tool.Tool in the given registry.
func RegisterEngineTools(reg *tools.Registry, engine plugins.ContextEngine) {
	if engine == nil {
		return
	}
	schemas := engine.GetToolSchemas()
	for _, s := range schemas {
		reg.Register(&engineToolAdapter{
			name:        s.Name,
			description: s.Description,
			parameters:  s.Parameters,
			engine:      engine,
		})
	}
}
