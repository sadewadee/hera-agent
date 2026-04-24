package builtin

import (
	"context"
	"encoding/json"
	"testing"
)

func TestScreenshotTool_Interface(t *testing.T) {
	tool := &ScreenshotTool{}
	if tool.Name() != "screenshot" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "screenshot")
	}
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}

	var params json.RawMessage
	params = tool.Parameters()
	if !json.Valid(params) {
		t.Error("Parameters() should return valid JSON")
	}
}

func TestScreenshotTool_Execute_NoArgs(t *testing.T) {
	tool := &ScreenshotTool{}
	ctx := context.Background()

	// Execute with empty args should attempt screenshot (may fail without display)
	result, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	// On CI/headless systems, screenshot will fail - that's expected
	if result == nil {
		t.Fatal("result should not be nil")
	}
}
