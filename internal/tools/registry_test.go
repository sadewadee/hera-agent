package tools

import (
	"context"
	"encoding/json"
	"testing"
)

// mockTool is a simple Tool implementation for testing.
type mockTool struct {
	name        string
	description string
	params      json.RawMessage
	executeFn   func(ctx context.Context, args json.RawMessage) (*Result, error)
}

func (m *mockTool) Name() string                { return m.name }
func (m *mockTool) Description() string         { return m.description }
func (m *mockTool) Parameters() json.RawMessage { return m.params }
func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, args)
	}
	return &Result{Content: "mock result"}, nil
}

func newMockTool(name, description string) *mockTool {
	return &mockTool{
		name:        name,
		description: description,
		params:      json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}}}`),
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	tool := newMockTool("greet", "Says hello")

	reg.Register(tool)

	got, ok := reg.Get("greet")
	if !ok {
		t.Fatal("Get returned false for registered tool")
	}
	if got.Name() != "greet" {
		t.Errorf("Name() = %q, want %q", got.Name(), "greet")
	}
	if got.Description() != "Says hello" {
		t.Errorf("Description() = %q, want %q", got.Description(), "Says hello")
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Get returned true for unknown tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newMockTool("tool_a", "Tool A"))
	reg.Register(newMockTool("tool_b", "Tool B"))
	reg.Register(newMockTool("tool_c", "Tool C"))

	list := reg.List()
	if len(list) != 3 {
		t.Fatalf("List() returned %d tools, want 3", len(list))
	}

	names := make(map[string]bool)
	for _, tool := range list {
		names[tool.Name()] = true
	}
	for _, want := range []string{"tool_a", "tool_b", "tool_c"} {
		if !names[want] {
			t.Errorf("List() missing tool %q", want)
		}
	}
}

func TestRegistry_Execute(t *testing.T) {
	reg := NewRegistry()
	tool := &mockTool{
		name:        "echo",
		description: "Echoes input",
		params:      json.RawMessage(`{}`),
		executeFn: func(ctx context.Context, args json.RawMessage) (*Result, error) {
			var input struct {
				Text string `json:"text"`
			}
			json.Unmarshal(args, &input)
			return &Result{Content: "echo: " + input.Text}, nil
		},
	}
	reg.Register(tool)

	result, err := reg.Execute(context.Background(), "echo", json.RawMessage(`{"text":"hello"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Content != "echo: hello" {
		t.Errorf("Execute() content = %q, want %q", result.Content, "echo: hello")
	}
	if result.IsError {
		t.Error("Execute() result marked as error, want non-error")
	}
}

func TestRegistry_ExecuteUnknown(t *testing.T) {
	reg := NewRegistry()

	result, err := reg.Execute(context.Background(), "missing_tool", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("Execute() for unknown tool should return error result")
	}
	if result.Content == "" {
		t.Error("Execute() for unknown tool should have a content message")
	}
}

func TestRegistry_ToolDefs(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newMockTool("search", "Search the web"))
	reg.Register(newMockTool("calculate", "Do math"))

	defs := reg.ToolDefs()
	if len(defs) != 2 {
		t.Fatalf("ToolDefs() returned %d definitions, want 2", len(defs))
	}

	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
		if d.Description == "" {
			t.Errorf("ToolDef %q has empty description", d.Name)
		}
		if len(d.Parameters) == 0 {
			t.Errorf("ToolDef %q has empty parameters", d.Name)
		}
	}
	if !names["search"] {
		t.Error("ToolDefs() missing 'search'")
	}
	if !names["calculate"] {
		t.Error("ToolDefs() missing 'calculate'")
	}
}
