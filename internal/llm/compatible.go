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

// CompatibleProvider implements the Provider interface for any OpenAI-compatible API.
// This works with vLLM, HuggingFace TGI, Nous Portal, LiteLLM, LocalAI, and
// any other server that exposes an OpenAI-compatible /v1/chat/completions endpoint.
type CompatibleProvider struct {
	apiKey       string // fallback single key (still respected if pool is nil)
	pool         *CredentialPool
	baseURL      string
	model        string
	client       *http.Client
	label        string                  // user-friendly name for error messages
	headerSetter func(req *http.Request) // optional custom header setter
}

// NewCompatibleProvider creates a new generic OpenAI-compatible provider.
func NewCompatibleProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("compatible: base_url is required")
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &CompatibleProvider{
		apiKey:  cfg.APIKey,
		pool:    BuildCredentialPool(cfg),
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: timeout},
		label:   "compatible",
	}, nil
}

// Chat sends a non-streaming chat completion request.
func (p *CompatibleProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Try up to 2 attempts so a single bad key in the pool doesn't fail
	// a request that would have succeeded with a healthy key.
	attempts := 1
	if p.pool != nil && p.pool.Size() > 1 {
		attempts = 2
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		body := p.buildRequestBody(req, false)
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%s: marshal request: %w", p.label, err)
		}
		url := p.baseURL + "/chat/completions"
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%s: create request: %w", p.label, err)
		}
		usedKey := p.setHeaders(httpReq)

		httpResp, err := p.client.Do(httpReq)
		if err != nil {
			p.pool.MarkFailure(usedKey, 0)
			lastErr = fmt.Errorf("%s: send request: %w", p.label, err)
			continue
		}

		respBody, readErr := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("%s: read response: %w", p.label, readErr)
		}

		if httpResp.StatusCode != http.StatusOK {
			p.pool.MarkFailure(usedKey, httpResp.StatusCode)
			lastErr = fmt.Errorf("%s: API error (status %d): %s", p.label, httpResp.StatusCode, string(respBody))
			// Only retry on auth/rate errors where rotating helps.
			if httpResp.StatusCode == http.StatusUnauthorized ||
				httpResp.StatusCode == http.StatusForbidden ||
				httpResp.StatusCode == http.StatusTooManyRequests {
				continue
			}
			return nil, lastErr
		}

		p.pool.MarkSuccess(usedKey)

		var oaiResp openaiChatResponse
		if err := json.Unmarshal(respBody, &oaiResp); err != nil {
			return nil, fmt.Errorf("%s: unmarshal response: %w", p.label, err)
		}
		return p.convertResponse(&oaiResp)
	}
	return nil, lastErr
}

// ChatStream sends a streaming chat completion request.
func (p *CompatibleProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body := p.buildRequestBody(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", p.label, err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", p.label, err)
	}
	usedKey := p.setHeaders(httpReq)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		p.pool.MarkFailure(usedKey, 0)
		return nil, fmt.Errorf("%s: send request: %w", p.label, err)
	}

	if httpResp.StatusCode != http.StatusOK {
		p.pool.MarkFailure(usedKey, httpResp.StatusCode)
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("%s: API error (status %d): %s", p.label, httpResp.StatusCode, string(respBody))
	}

	// Stream opened: credential worked. Clear any prior cooldown.
	p.pool.MarkSuccess(usedKey)

	ch := make(chan StreamEvent, 64)

	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		// Track in-flight tool calls (OpenAI streams them incrementally).
		type pendingToolCall struct {
			ID        string
			Name      string
			ArgsBytes []byte
		}
		var pendingTools []pendingToolCall
		var lastUsage *Usage

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")

			if payload == "[DONE]" {
				// Emit any accumulated tool calls before done.
				for _, pt := range pendingTools {
					ch <- StreamEvent{
						Type: "tool_call",
						ToolCall: &ToolCall{
							ID:   pt.ID,
							Name: pt.Name,
							Args: json.RawMessage(pt.ArgsBytes),
						},
					}
				}
				if lastUsage != nil {
					ch <- StreamEvent{Type: "done", Usage: lastUsage}
				} else {
					ch <- StreamEvent{Type: "done"}
				}
				return
			}

			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("%s: parse stream chunk: %w", p.label, err)}
				return
			}

			if len(chunk.Choices) == 0 {
				// Could be a usage-only chunk.
				if chunk.Usage != nil {
					lastUsage = &Usage{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
					}
				}
				continue
			}

			choice := chunk.Choices[0]

			if choice.Delta.Content != "" {
				ch <- StreamEvent{Type: "delta", Delta: choice.Delta.Content}
			}

			// Accumulate tool calls across streaming chunks.
			for _, tc := range choice.Delta.ToolCalls {
				idx := tc.Index
				// Grow pending slice if needed.
				for len(pendingTools) <= idx {
					pendingTools = append(pendingTools, pendingToolCall{})
				}
				if tc.ID != "" {
					pendingTools[idx].ID = tc.ID
				}
				if tc.Function.Name != "" {
					pendingTools[idx].Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					pendingTools[idx].ArgsBytes = append(pendingTools[idx].ArgsBytes, []byte(tc.Function.Arguments)...)
				}
			}

			// Check if response is finished with tool_calls.
			if choice.FinishReason == "tool_calls" || choice.FinishReason == "stop" {
				// Emit accumulated tool calls.
				for _, pt := range pendingTools {
					if pt.Name != "" {
						ch <- StreamEvent{
							Type: "tool_call",
							ToolCall: &ToolCall{
								ID:   pt.ID,
								Name: pt.Name,
								Args: json.RawMessage(pt.ArgsBytes),
							},
						}
					}
				}
				pendingTools = nil
			}

			if chunk.Usage != nil {
				lastUsage = &Usage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}
		}

		// Emit any remaining tool calls (in case stream ended without [DONE]).
		for _, pt := range pendingTools {
			if pt.Name != "" {
				ch <- StreamEvent{
					Type: "tool_call",
					ToolCall: &ToolCall{
						ID:   pt.ID,
						Name: pt.Name,
						Args: json.RawMessage(pt.ArgsBytes),
					},
				}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
			default:
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("%s: stream read error: %w", p.label, err)}
			}
		}
	}()

	return ch, nil
}

// CountTokens estimates the token count for messages.
func (p *CompatibleProvider) CountTokens(messages []Message) (int, error) {
	total := 0
	for _, msg := range messages {
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
	if total == 0 && len(messages) > 0 {
		total = 1
	}
	return total, nil
}

// ModelInfo returns metadata about the current model.
func (p *CompatibleProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      p.label,
		ContextWindow: 128000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// FetchModels queries the /v1/models endpoint of an OpenAI-compatible API
// and returns a list of available model IDs. Works with Ollama, LM Studio,
// vLLM, LocalAI, OpenRouter, and any other OpenAI-compatible endpoint.
func FetchModels(baseURL, apiKey string) ([]string, error) {
	url := strings.TrimRight(baseURL, "/") + "/models"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("models endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
		// Ollama returns {"models": [...]} instead of {"data": [...]}
		Models []struct {
			Name string `json:"name"`
			// Ollama also uses "model" field
			Model string `json:"model"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	var models []string

	// Standard OpenAI format: {"data": [{"id": "model-name"}]}
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}

	// Ollama format: {"models": [{"name": "llama3:latest"}]}
	if len(models) == 0 {
		for _, m := range result.Models {
			name := m.Name
			if name == "" {
				name = m.Model
			}
			if name != "" {
				models = append(models, name)
			}
		}
	}

	return models, nil
}

// RegisterCompatible registers the generic OpenAI-compatible provider factory.
func RegisterCompatible(reg *Registry) {
	reg.Register("compatible", func(cfg ProviderConfig) (Provider, error) {
		return NewCompatibleProvider(cfg)
	})
}

// setHeaders applies Content-Type and Authorization. Returns the API
// key actually used so the caller can feed success/failure back into
// the credential pool. Returns "" when no key is configured (which is
// legal for some local servers).
func (p *CompatibleProvider) setHeaders(req *http.Request) string {
	if p.headerSetter != nil {
		p.headerSetter(req)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	key := p.apiKey
	if p.pool != nil && p.pool.Size() > 0 {
		key = p.pool.Next()
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return key
}

func (p *CompatibleProvider) buildRequestBody(req ChatRequest, stream bool) map[string]any {
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

		// Default to "auto" when the caller didn't specify. Without this
		// key the endpoint's default may bias to text completion even
		// when tools are present, which is exactly how the LLM was
		// hallucinating save-intent without emitting memory_note_save.
		choice := req.ToolChoice
		if choice == "" {
			choice = "auto"
		}
		body["tool_choice"] = choice
	}

	return body
}

func (p *CompatibleProvider) convertResponse(oaiResp *openaiChatResponse) (*ChatResponse, error) {
	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("%s: empty choices in response", p.label)
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
