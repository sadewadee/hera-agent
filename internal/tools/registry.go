package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sadewadee/hera/internal/llm"
)

// Registry manages available tools and dispatches execution.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Execute runs a tool by name with the given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) (*Result, error) {
	tool, ok := r.Get(name)
	if !ok {
		return &Result{
			Content: fmt.Sprintf("unknown tool: %s", name),
			IsError: true,
		}, nil
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		return &Result{
			Content: fmt.Sprintf("tool %s error: %v", name, err),
			IsError: true,
		}, nil
	}

	return result, nil
}

// Unregister removes a tool from the registry by name.
// Returns an error if the tool is not registered.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[name]; !ok {
		return fmt.Errorf("tool %q is not registered", name)
	}
	delete(r.tools, name)
	return nil
}

// ToolDefs generates LLM-compatible tool definitions for all registered tools.
func (r *Registry) ToolDefs() []llm.ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]llm.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, llm.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}
