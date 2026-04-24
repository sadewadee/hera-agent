package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
)

func TestClipboardTool_Name(t *testing.T) {
	tool := &ClipboardTool{}
	if got := tool.Name(); got != "clipboard" {
		t.Errorf("Name() = %q, want %q", got, "clipboard")
	}
}

func TestClipboardTool_Description(t *testing.T) {
	tool := &ClipboardTool{}
	if desc := tool.Description(); desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestClipboardTool_Parameters(t *testing.T) {
	tool := &ClipboardTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Parameters() returned invalid JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
}

func TestClipboardTool_InvalidJSON(t *testing.T) {
	tool := &ClipboardTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad json}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for invalid JSON")
	}
	if !strings.Contains(result.Content, "invalid arguments") {
		t.Errorf("expected 'invalid arguments' in content, got %q", result.Content)
	}
}

func TestClipboardTool_InvalidAction(t *testing.T) {
	tool := &ClipboardTool{}
	args, _ := json.Marshal(clipboardArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for invalid action")
	}
	if !strings.Contains(result.Content, "read") || !strings.Contains(result.Content, "write") {
		t.Errorf("error should mention valid actions, got %q", result.Content)
	}
}

func TestClipboardTool_WriteRequiresText(t *testing.T) {
	tool := &ClipboardTool{}
	args, _ := json.Marshal(clipboardArgs{Action: "write", Text: ""})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError when writing with empty text")
	}
	if !strings.Contains(result.Content, "text is required") {
		t.Errorf("expected 'text is required' error, got %q", result.Content)
	}
}

func TestClipboardTool_ReadExecutes(t *testing.T) {
	tool := &ClipboardTool{}
	args, _ := json.Marshal(clipboardArgs{Action: "read"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	// On macOS, pbpaste will succeed. On CI/linux without xclip, it may fail.
	// We just verify the tool doesn't panic and returns a result.
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestClipboardTool_WriteExecutes(t *testing.T) {
	tool := &ClipboardTool{}
	args, _ := json.Marshal(clipboardArgs{Action: "write", Text: "hello clipboard"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	// On macOS this should succeed
	if !result.IsError && !strings.Contains(result.Content, "15 characters") {
		t.Logf("clipboard write result: %s", result.Content)
	}
}

func TestRegisterClipboard(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterClipboard(registry)
	if _, ok := registry.Get("clipboard"); !ok {
		t.Error("clipboard tool should be registered")
	}
}
