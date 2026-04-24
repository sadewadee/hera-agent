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
	anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
	anthropicAPIVersion     = "2023-06-01"
)

// AnthropicProvider implements the Provider interface for Anthropic's Claude API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider from configuration.
func NewAnthropicProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = anthropicDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &AnthropicProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// Chat sends a non-streaming chat completion request to Anthropic.
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildRequestBody(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	url := p.baseURL + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}
	p.setHeaders(httpReq)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var antResp anthropicMessageResponse
	if err := json.Unmarshal(respBody, &antResp); err != nil {
		return nil, fmt.Errorf("anthropic: unmarshal response: %w", err)
	}

	return p.convertResponse(&antResp)
}

// ChatStream sends a streaming chat completion request to Anthropic.
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body := p.buildRequestBody(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	url := p.baseURL + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}
	p.setHeaders(httpReq)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("anthropic: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 64)

	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		var eventType string

		// State for accumulating streamed tool_use blocks.
		var currentToolID, currentToolName string
		var argsBuffer strings.Builder

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")

			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(payload), &raw); err != nil {
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("anthropic: parse stream: %w", err)}
				return
			}

			switch eventType {
			case "content_block_start":
				if cb, ok := raw["content_block"]; ok {
					var block struct {
						Type string `json:"type"`
						ID   string `json:"id"`
						Name string `json:"name"`
					}
					json.Unmarshal(cb, &block)
					if block.Type == "tool_use" {
						currentToolID = block.ID
						currentToolName = block.Name
						argsBuffer.Reset()
					}
				}

			case "content_block_delta":
				var delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					PartialJSON string `json:"partial_json"`
				}
				if d, ok := raw["delta"]; ok {
					json.Unmarshal(d, &delta)
				}
				switch delta.Type {
				case "text_delta":
					if delta.Text != "" {
						ch <- StreamEvent{Type: "delta", Delta: delta.Text}
					}
				case "input_json_delta":
					argsBuffer.WriteString(delta.PartialJSON)
				}

			case "content_block_stop":
				if currentToolID != "" {
					ch <- StreamEvent{
						Type: "tool_call",
						ToolCall: &ToolCall{
							ID:   currentToolID,
							Name: currentToolName,
							Args: json.RawMessage(argsBuffer.String()),
						},
					}
					currentToolID = ""
					currentToolName = ""
					argsBuffer.Reset()
				}

			case "message_delta":
				// Contains stop_reason and final usage.
				var delta struct {
					StopReason string `json:"stop_reason"`
				}
				if d, ok := raw["delta"]; ok {
					json.Unmarshal(d, &delta)
				}
				// Usage might be at top level of this event.
				if u, ok := raw["usage"]; ok {
					var usage struct {
						OutputTokens int `json:"output_tokens"`
					}
					json.Unmarshal(u, &usage)
				}

			case "message_stop":
				ch <- StreamEvent{Type: "done"}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
			default:
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("anthropic: stream read error: %w", err)}
			}
		}
	}()

	return ch, nil
}

// CountTokens estimates the token count for messages.
// Uses approximate calculation: 4 characters per token.
func (p *AnthropicProvider) CountTokens(messages []Message) (int, error) {
	total := 0
	for _, msg := range messages {
		total += 4 // Per-message overhead.
		total += len(msg.Content) / 4
		if msg.Name != "" {
			total += len(msg.Name) / 4
		}
		for _, tc := range msg.ToolCalls {
			total += len(tc.Name) / 4
			total += len(tc.Args) / 4
		}
	}
	if total == 0 && len(messages) > 0 {
		total = 1
	}
	return total, nil
}

// ModelInfo returns metadata about the current model.
func (p *AnthropicProvider) ModelInfo() ModelMetadata {
	info := ModelMetadata{
		ID:            p.model,
		Provider:      "anthropic",
		SupportsTools: true,
	}

	switch {
	case strings.Contains(p.model, "opus"):
		info.ContextWindow = 200000
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.015
		info.CostPer1kOut = 0.075
	case strings.Contains(p.model, "sonnet"):
		info.ContextWindow = 200000
		info.MaxOutput = 8192
		info.CostPer1kIn = 0.003
		info.CostPer1kOut = 0.015
	case strings.Contains(p.model, "haiku"):
		info.ContextWindow = 200000
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.00025
		info.CostPer1kOut = 0.00125
	default:
		info.ContextWindow = 200000
		info.MaxOutput = 4096
	}

	return info
}

// RegisterAnthropic registers the Anthropic provider factory with the given registry.
func RegisterAnthropic(reg *Registry) {
	reg.Register("anthropic", func(cfg ProviderConfig) (Provider, error) {
		return NewAnthropicProvider(cfg)
	})
}

// --- Internal helpers ---

func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
}

func (p *AnthropicProvider) buildRequestBody(req ChatRequest, stream bool) map[string]any {
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Anthropic separates system messages from the conversation.
	var systemContent string
	var messages []map[string]any

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			systemContent = msg.Content
			continue
		}

		// Anthropic requires role to be "user" or "assistant".
		role := string(msg.Role)
		if msg.Role == RoleTool {
			role = "user"
		}

		m := map[string]any{
			"role": role,
		}

		// Tool result messages use a different content format.
		if msg.Role == RoleTool && msg.ToolCallID != "" {
			m["content"] = []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     msg.Content,
				},
			}
		} else {
			m["content"] = msg.Content
		}

		messages = append(messages, m)
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}

	if systemContent != "" {
		body["system"] = systemContent
	}
	if stream {
		body["stream"] = true
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if len(req.Stop) > 0 {
		body["stop_sequences"] = req.Stop
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]any{
				"name":         tool.Name,
				"description":  tool.Description,
				"input_schema": json.RawMessage(tool.Parameters),
			}
		}
		body["tools"] = tools

		// Anthropic's tool_choice is an object, not a bare string. Map
		// simple string values from the cross-provider ChatRequest to
		// Anthropic's schema.
		switch req.ToolChoice {
		case "required":
			body["tool_choice"] = map[string]any{"type": "any"}
		case "none":
			// omit tools entirely by not setting tool_choice and leaving tools
			// in place — Anthropic doesn't have a "none" equivalent; fall
			// back to letting the model decide. Callers who truly want
			// none should omit Tools from the request.
		case "", "auto":
			body["tool_choice"] = map[string]any{"type": "auto"}
		default:
			// Treat any other string as a forced tool name.
			body["tool_choice"] = map[string]any{
				"type": "tool",
				"name": req.ToolChoice,
			}
		}
	}

	// Support prompt caching via metadata.
	if req.Metadata != nil {
		if cacheControl, ok := req.Metadata["cache_control"]; ok {
			body["metadata"] = map[string]any{
				"cache_control": cacheControl,
			}
		}
	}

	return body
}

func (p *AnthropicProvider) convertResponse(resp *anthropicMessageResponse) (*ChatResponse, error) {
	msg := Message{
		Role: RoleAssistant,
	}

	var textParts []string
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			argsJSON, err := json.Marshal(block.Input)
			if err != nil {
				return nil, fmt.Errorf("anthropic: marshal tool input: %w", err)
			}
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: json.RawMessage(argsJSON),
			})
		}
	}
	msg.Content = strings.Join(textParts, "")

	finishReason := resp.StopReason
	if finishReason == "end_turn" {
		finishReason = "stop"
	}

	return &ChatResponse{
		Message: msg,
		Model:   resp.Model,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
			CachedTokens:     resp.Usage.CacheReadInputTokens,
		},
		FinishReason: finishReason,
	}, nil
}

// Anthropic API response types.

type anthropicMessageResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Model      string                  `json:"model"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}
