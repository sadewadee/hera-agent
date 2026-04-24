package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRegexTool_Test(t *testing.T) {
	tool := &RegexTool{}
	ctx := context.Background()

	tests := []struct {
		name    string
		pattern string
		text    string
		match   bool
	}{
		{"simple match", `\d+`, "abc123def", true},
		{"no match", `\d+`, "abcdef", false},
		{"email", `[\w.]+@[\w.]+`, "user@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(regexToolArgs{Action: "test", Pattern: tt.pattern, Text: tt.text})
			result, _ := tool.Execute(ctx, args)
			if tt.match && strings.Contains(result.Content, "No match") {
				t.Error("expected match")
			}
			if !tt.match && !strings.Contains(result.Content, "No match") {
				t.Error("expected no match")
			}
		})
	}
}

func TestRegexTool_FindAll(t *testing.T) {
	tool := &RegexTool{}
	ctx := context.Background()

	args, _ := json.Marshal(regexToolArgs{Action: "find_all", Pattern: `\d+`, Text: "a1 b22 c333"})
	result, _ := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("find_all error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "3 matches") {
		t.Errorf("expected 3 matches, got: %s", result.Content)
	}
}

func TestRegexTool_Replace(t *testing.T) {
	tool := &RegexTool{}
	ctx := context.Background()

	args, _ := json.Marshal(regexToolArgs{Action: "replace", Pattern: `\d+`, Text: "a1 b2 c3", Replace: "X"})
	result, _ := tool.Execute(ctx, args)
	if result.Content != "aX bX cX" {
		t.Errorf("replace = %q, want %q", result.Content, "aX bX cX")
	}
}

func TestRegexTool_Split(t *testing.T) {
	tool := &RegexTool{}
	ctx := context.Background()

	args, _ := json.Marshal(regexToolArgs{Action: "split", Pattern: `[,;]`, Text: "a,b;c,d"})
	result, _ := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("split error: %s", result.Content)
	}
	for _, expected := range []string{"a", "b", "c", "d"} {
		if !strings.Contains(result.Content, expected) {
			t.Errorf("split should contain %q, got: %s", expected, result.Content)
		}
	}
}

func TestRegexTool_InvalidPattern(t *testing.T) {
	tool := &RegexTool{}
	ctx := context.Background()

	args, _ := json.Marshal(regexToolArgs{Action: "test", Pattern: `[invalid`, Text: "test"})
	result, _ := tool.Execute(ctx, args)
	if !result.IsError {
		t.Error("invalid regex should return error")
	}
}
