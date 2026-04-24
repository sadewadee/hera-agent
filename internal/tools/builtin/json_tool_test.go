package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONTool_Validate(t *testing.T) {
	tool := &JSONTool{}
	ctx := context.Background()

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"valid object", `{"key":"value"}`, false},
		{"valid array", `[1,2,3]`, false},
		{"invalid json", `{bad}`, false}, // validate returns content, not IsError
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(jsonToolArgs{Action: "validate", Data: tt.data})
			result, err := tool.Execute(ctx, args)
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if result == nil {
				t.Fatal("result should not be nil")
			}
		})
	}
}

func TestJSONTool_Format(t *testing.T) {
	tool := &JSONTool{}
	ctx := context.Background()

	args, _ := json.Marshal(jsonToolArgs{Action: "format", Data: `{"a":1,"b":2}`})
	result, _ := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("format error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "\n") {
		t.Error("formatted JSON should contain newlines")
	}
}

func TestJSONTool_Minify(t *testing.T) {
	tool := &JSONTool{}
	ctx := context.Background()

	args, _ := json.Marshal(jsonToolArgs{Action: "minify", Data: `{ "a": 1, "b": 2 }`})
	result, _ := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("minify error: %s", result.Content)
	}
	if strings.Contains(result.Content, " ") {
		t.Error("minified JSON should not contain spaces")
	}
}

func TestJSONTool_Get(t *testing.T) {
	tool := &JSONTool{}
	ctx := context.Background()

	data := `{"user":{"name":"Alice","age":30},"items":[1,2,3]}`

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"nested key", "user.name", "Alice", false},
		{"array index", "items.1", "2", false},
		{"missing key", "user.missing", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(jsonToolArgs{Action: "get", Data: data, Path: tt.path})
			result, _ := tool.Execute(ctx, args)
			if tt.wantErr && !result.IsError {
				t.Error("expected error")
			}
			if !tt.wantErr && result.IsError {
				t.Errorf("unexpected error: %s", result.Content)
			}
			if !tt.wantErr && !strings.Contains(result.Content, tt.want) {
				t.Errorf("got %q, want containing %q", result.Content, tt.want)
			}
		})
	}
}

func TestJSONTool_Keys(t *testing.T) {
	tool := &JSONTool{}
	ctx := context.Background()

	args, _ := json.Marshal(jsonToolArgs{Action: "keys", Data: `{"a":1,"b":2,"c":3}`})
	result, _ := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("keys error: %s", result.Content)
	}
	for _, key := range []string{"a", "b", "c"} {
		if !strings.Contains(result.Content, key) {
			t.Errorf("keys should contain %q", key)
		}
	}
}
