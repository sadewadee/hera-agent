package tools

import (
	"context"
	"encoding/json"
)

// Tool is the interface all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema
	Execute(ctx context.Context, args json.RawMessage) (*Result, error)
}

// Result is the output of a tool execution.
type Result struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}
