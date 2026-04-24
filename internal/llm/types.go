package llm

import (
	"encoding/json"
	"net/http"
	"time"
)

// Role represents a message role in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Timestamp  time.Time  `json:"timestamp,omitempty"`
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	CallID  string `json:"call_id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

// ChatRequest is the input for a chat completion.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
	// ToolChoice controls tool-invocation policy. Empty string lets the
	// provider default it — compatible-style providers default to "auto"
	// when Tools is non-empty so the LLM reliably emits tool calls even
	// in persona-heavy contexts. Explicit values: "auto", "required",
	// "none", or a function-name object.
	ToolChoice  string         `json:"tool_choice,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Stop        []string       `json:"stop,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ToolDef defines a tool available for the model to call.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatResponse is the output from a chat completion.
type ChatResponse struct {
	Message      Message `json:"message"`
	Usage        Usage   `json:"usage"`
	Model        string  `json:"model"`
	FinishReason string  `json:"finish_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CachedTokens     int `json:"cached_tokens,omitempty"`
}

// StreamEvent represents a single event in a streaming response.
type StreamEvent struct {
	Type     string    `json:"type"` // "delta", "tool_call", "done", "error"
	Delta    string    `json:"delta,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Usage    *Usage    `json:"usage,omitempty"`
	Error    error     `json:"-"`
}

// newHTTPClient creates an http.Client with the given timeout.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// ModelMetadata describes a model's capabilities and pricing.
type ModelMetadata struct {
	ID             string  `json:"id"`
	Provider       string  `json:"provider"`
	ContextWindow  int     `json:"context_window"`
	MaxOutput      int     `json:"max_output"`
	SupportsTools  bool    `json:"supports_tools"`
	SupportsVision bool    `json:"supports_vision"`
	CostPer1kIn    float64 `json:"cost_per_1k_in"`
	CostPer1kOut   float64 `json:"cost_per_1k_out"`
}
