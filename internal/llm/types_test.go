package llm

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, Role("system"), RoleSystem)
	assert.Equal(t, Role("user"), RoleUser)
	assert.Equal(t, Role("assistant"), RoleAssistant)
	assert.Equal(t, Role("tool"), RoleTool)
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "hello world",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, msg.Role, decoded.Role)
	assert.Equal(t, msg.Content, decoded.Content)
}

func TestMessage_WithToolCalls(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ToolCalls: []ToolCall{
			{
				ID:   "call-1",
				Name: "read_file",
				Args: json.RawMessage(`{"path": "/tmp/test.go"}`),
			},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded.ToolCalls, 1)
	assert.Equal(t, "call-1", decoded.ToolCalls[0].ID)
}

func TestToolCall_JSON(t *testing.T) {
	tc := ToolCall{
		ID:   "tc-123",
		Name: "terminal",
		Args: json.RawMessage(`{"command": "ls"}`),
	}

	data, err := json.Marshal(tc)
	require.NoError(t, err)

	var decoded ToolCall
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "tc-123", decoded.ID)
	assert.Equal(t, "terminal", decoded.Name)
}

func TestToolResult_JSON(t *testing.T) {
	tr := ToolResult{
		CallID:  "call-1",
		Content: "file contents here",
		IsError: false,
	}

	data, err := json.Marshal(tr)
	require.NoError(t, err)

	var decoded ToolResult
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, tr.CallID, decoded.CallID)
	assert.Equal(t, tr.Content, decoded.Content)
}

func TestToolResult_Error(t *testing.T) {
	tr := ToolResult{
		CallID:  "call-2",
		Content: "permission denied",
		IsError: true,
	}
	assert.True(t, tr.IsError)
}

func TestChatRequest(t *testing.T) {
	temp := 0.7
	req := ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
		MaxTokens:   1000,
		Temperature: &temp,
		Stream:      true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded ChatRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "gpt-4o", decoded.Model)
	assert.Len(t, decoded.Messages, 1)
	assert.Equal(t, 1000, decoded.MaxTokens)
	assert.NotNil(t, decoded.Temperature)
	assert.InDelta(t, 0.7, *decoded.Temperature, 0.001)
}

func TestToolDef(t *testing.T) {
	td := ToolDef{
		Name:        "read_file",
		Description: "Reads a file",
		Parameters:  json.RawMessage(`{"type":"object"}`),
	}
	assert.Equal(t, "read_file", td.Name)
	assert.Equal(t, "Reads a file", td.Description)
}

func TestChatResponse(t *testing.T) {
	resp := ChatResponse{
		Message:      Message{Role: RoleAssistant, Content: "hello"},
		Usage:        Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		Model:        "gpt-4o",
		FinishReason: "stop",
	}

	assert.Equal(t, "gpt-4o", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestUsage(t *testing.T) {
	u := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		CachedTokens:     20,
	}
	assert.Equal(t, 100, u.PromptTokens)
	assert.Equal(t, 50, u.CompletionTokens)
	assert.Equal(t, 150, u.TotalTokens)
	assert.Equal(t, 20, u.CachedTokens)
}

func TestStreamEvent(t *testing.T) {
	ev := StreamEvent{
		Type:  "delta",
		Delta: "hello",
	}
	assert.Equal(t, "delta", ev.Type)
	assert.Equal(t, "hello", ev.Delta)
}

func TestStreamEvent_Error(t *testing.T) {
	ev := StreamEvent{
		Type:  "error",
		Error: assert.AnError,
	}
	assert.Equal(t, "error", ev.Type)
	assert.Error(t, ev.Error)
}

func TestModelMetadata(t *testing.T) {
	m := ModelMetadata{
		ID:             "gpt-4o",
		Provider:       "openai",
		ContextWindow:  128000,
		MaxOutput:      4096,
		SupportsTools:  true,
		SupportsVision: true,
		CostPer1kIn:    0.005,
		CostPer1kOut:   0.015,
	}
	assert.Equal(t, "gpt-4o", m.ID)
	assert.Equal(t, "openai", m.Provider)
	assert.True(t, m.SupportsTools)
	assert.True(t, m.SupportsVision)
}

func TestNewHTTPClient(t *testing.T) {
	client := newHTTPClient(30 * time.Second)
	assert.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)
}
