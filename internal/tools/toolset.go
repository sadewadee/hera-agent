package tools

// ToolSet groups related tools under a common name.
type ToolSet struct {
	name  string
	tools []Tool
}

// NewToolSet creates a new tool set.
func NewToolSet(name string) *ToolSet {
	return &ToolSet{name: name, tools: make([]Tool, 0)}
}

// Name returns the tool set name.
func (ts *ToolSet) Name() string { return ts.name }

// Add adds a tool to the set.
func (ts *ToolSet) Add(t Tool) { ts.tools = append(ts.tools, t) }

// Tools returns all tools in the set.
func (ts *ToolSet) Tools() []Tool { return ts.tools }

// RegisterAll registers all tools in the set with the given registry.
func (ts *ToolSet) RegisterAll(registry *Registry) {
	for _, t := range ts.tools {
		registry.Register(t)
	}
}
