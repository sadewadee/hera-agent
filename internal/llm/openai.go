package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	openaiDefaultBaseURL = "https://api.openai.com/v1"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	orgID   string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider from configuration.
func NewOpenAIProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openaiDefaultBaseURL
	}
	// Strip trailing slash.
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &OpenAIProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Model,
		orgID:   cfg.OrgID,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// Chat sends a non-streaming chat completion request.
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildRequestBody(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	p.setHeaders(httpReq)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var oaiResp openaiChatResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("openai: unmarshal response: %w", err)
	}

	return p.convertResponse(&oaiResp)
}

// ChatStream sends a streaming chat completion request.
func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body := p.buildRequestBody(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	p.setHeaders(httpReq)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("openai: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 64)

	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")

			if payload == "[DONE]" {
				ch <- StreamEvent{Type: "done"}
				return
			}

			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("openai: parse stream chunk: %w", err)}
				return
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]

			// Emit text delta.
			if choice.Delta.Content != "" {
				ch <- StreamEvent{Type: "delta", Delta: choice.Delta.Content}
			}

			// Emit tool calls.
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Function.Name != "" {
					ch <- StreamEvent{
						Type: "tool_call",
						ToolCall: &ToolCall{
							ID:   tc.ID,
							Name: tc.Function.Name,
							Args: json.RawMessage(tc.Function.Arguments),
						},
					}
				}
			}

			// Emit usage if present.
			if chunk.Usage != nil {
				ch <- StreamEvent{
					Type: "done",
					Usage: &Usage{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
					},
				}
				return
			}

			// Check for finish reason.
			if choice.FinishReason != "" && chunk.Usage == nil {
				// Will get done from [DONE] or usage chunk.
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
			default:
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("openai: stream read error: %w", err)}
			}
		}
	}()

	return ch, nil
}

// CountTokens estimates the token count for messages.
// Uses a simple approximation of 4 characters per token.
func (p *OpenAIProvider) CountTokens(messages []Message) (int, error) {
	total := 0
	for _, msg := range messages {
		// Each message has overhead of ~4 tokens for role/formatting.
		total += 4
		total += len(msg.Content) / 4
		if msg.Name != "" {
			total += len(msg.Name) / 4
		}
		for _, tc := range msg.ToolCalls {
			total += len(tc.Name) / 4
			total += len(tc.Args) / 4
		}
	}
	// Ensure at least 1 token per non-empty message set.
	if total == 0 && len(messages) > 0 {
		total = 1
	}
	return total, nil
}

// ModelInfo returns metadata about the current model.
func (p *OpenAIProvider) ModelInfo() ModelMetadata {
	info := ModelMetadata{
		ID:            p.model,
		Provider:      "openai",
		SupportsTools: true,
	}

	// Set known context windows for common models.
	switch {
	case strings.HasPrefix(p.model, "gpt-4o"):
		info.ContextWindow = 128000
		info.MaxOutput = 16384
		info.SupportsVision = true
		info.CostPer1kIn = 0.0025
		info.CostPer1kOut = 0.01
	case strings.HasPrefix(p.model, "gpt-4-turbo"):
		info.ContextWindow = 128000
		info.MaxOutput = 4096
		info.SupportsVision = true
		info.CostPer1kIn = 0.01
		info.CostPer1kOut = 0.03
	case strings.HasPrefix(p.model, "gpt-4"):
		info.ContextWindow = 8192
		info.MaxOutput = 8192
		info.CostPer1kIn = 0.03
		info.CostPer1kOut = 0.06
	case strings.HasPrefix(p.model, "gpt-3.5"):
		info.ContextWindow = 16385
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.0005
		info.CostPer1kOut = 0.0015
	default:
		info.ContextWindow = 128000
		info.MaxOutput = 4096
	}

	return info
}

// RegisterOpenAI registers the OpenAI provider factory with the given registry.
func RegisterOpenAI(reg *Registry) {
	reg.Register("openai", func(cfg ProviderConfig) (Provider, error) {
		return NewOpenAIProvider(cfg)
	})
}

// --- Internal request/response types ---

func (p *OpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.orgID != "" {
		req.Header.Set("OpenAI-Organization", p.orgID)
	}
}

func (p *OpenAIProvider) buildRequestBody(req ChatRequest, stream bool) map[string]any {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]map[string]any, 0, len(req.Messages))
	for _, msg := range req.Messages {
		m := map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
		if msg.Name != "" {
			m["name"] = msg.Name
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			tcs := make([]map[string]any, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				tcs[i] = map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": string(tc.Args),
					},
				}
			}
			m["tool_calls"] = tcs
		}
		messages = append(messages, m)
	}

	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   stream,
	}

	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if len(req.Stop) > 0 {
		body["stop"] = req.Stop
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  json.RawMessage(tool.Parameters),
				},
			}
		}
		body["tools"] = tools
		choice := req.ToolChoice
		if choice == "" {
			choice = "auto"
		}
		body["tool_choice"] = choice
	}

	return body
}

func (p *OpenAIProvider) convertResponse(oaiResp *openaiChatResponse) (*ChatResponse, error) {
	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty choices in response")
	}

	choice := oaiResp.Choices[0]
	msg := Message{
		Role:    Role(choice.Message.Role),
		Content: choice.Message.Content,
	}

	for _, tc := range choice.Message.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: json.RawMessage(tc.Function.Arguments),
		})
	}

	return &ChatResponse{
		Message: msg,
		Model:   oaiResp.Model,
		Usage: Usage{
			PromptTokens:     oaiResp.Usage.PromptTokens,
			CompletionTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:      oaiResp.Usage.TotalTokens,
		},
		FinishReason: choice.FinishReason,
	}, nil
}

// OpenAI API response types.

type openaiChatResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiStreamChunk struct {
	ID      string               `json:"id"`
	Choices []openaiStreamChoice `json:"choices"`
	Usage   *openaiUsage         `json:"usage,omitempty"`
}

type openaiStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openaiStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type openaiStreamDelta struct {
	Role      string                 `json:"role,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ToolCalls []openaiStreamToolCall `json:"tool_calls,omitempty"`
}

type openaiStreamToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openaiToolFunction `json:"function"`
}
