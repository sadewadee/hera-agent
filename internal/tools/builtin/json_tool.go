package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// JSONTool provides JSON manipulation operations.
type JSONTool struct{}

type jsonToolArgs struct {
	Action string `json:"action"`
	Data   string `json:"data"`
	Path   string `json:"path,omitempty"`
	Value  string `json:"value,omitempty"`
}

func (t *JSONTool) Name() string { return "json" }

func (t *JSONTool) Description() string {
	return "JSON manipulation: validate, format, minify, extract values by path, and query."
}

func (t *JSONTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["validate", "format", "minify", "get", "keys", "type"],
				"description": "JSON action: validate, format (pretty-print), minify, get (extract path), keys (list keys), type (show type)."
			},
			"data": {
				"type": "string",
				"description": "JSON data as string."
			},
			"path": {
				"type": "string",
				"description": "Dot-separated path for get action (e.g. 'user.name', 'items.0.id')."
			}
		},
		"required": ["action", "data"]
	}`)
}

func (t *JSONTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a jsonToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	switch a.Action {
	case "validate":
		var v any
		if err := json.Unmarshal([]byte(a.Data), &v); err != nil {
			return &tools.Result{Content: fmt.Sprintf("Invalid JSON: %v", err)}, nil
		}
		return &tools.Result{Content: "Valid JSON"}, nil

	case "format":
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(a.Data), "", "  "); err != nil {
			return &tools.Result{Content: fmt.Sprintf("format error: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: buf.String()}, nil

	case "minify":
		var buf bytes.Buffer
		if err := json.Compact(&buf, []byte(a.Data)); err != nil {
			return &tools.Result{Content: fmt.Sprintf("minify error: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: buf.String()}, nil

	case "get":
		return jsonGet(a.Data, a.Path)

	case "keys":
		var obj map[string]any
		if err := json.Unmarshal([]byte(a.Data), &obj); err != nil {
			return &tools.Result{Content: fmt.Sprintf("not a JSON object: %v", err), IsError: true}, nil
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		out, _ := json.Marshal(keys)
		return &tools.Result{Content: string(out)}, nil

	case "type":
		var v any
		if err := json.Unmarshal([]byte(a.Data), &v); err != nil {
			return &tools.Result{Content: fmt.Sprintf("parse error: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("%T", v)}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

func jsonGet(data, path string) (*tools.Result, error) {
	if path == "" {
		return &tools.Result{Content: "path is required for get action", IsError: true}, nil
	}

	var current any
	if err := json.Unmarshal([]byte(data), &current); err != nil {
		return &tools.Result{Content: fmt.Sprintf("parse error: %v", err), IsError: true}, nil
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return &tools.Result{Content: fmt.Sprintf("key %q not found", part), IsError: true}, nil
			}
			current = val
		case []any:
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				return &tools.Result{Content: fmt.Sprintf("invalid array index %q", part), IsError: true}, nil
			}
			if idx < 0 || idx >= len(v) {
				return &tools.Result{Content: fmt.Sprintf("index %d out of range (length %d)", idx, len(v)), IsError: true}, nil
			}
			current = v[idx]
		default:
			return &tools.Result{Content: fmt.Sprintf("cannot traverse %T at %q", current, part), IsError: true}, nil
		}
	}

	out, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("%v", current)}, nil
	}
	return &tools.Result{Content: string(out)}, nil
}

// RegisterJSON registers the JSON tool with the given registry.
func RegisterJSON(registry *tools.Registry) {
	registry.Register(&JSONTool{})
}
