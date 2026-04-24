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
	geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiProvider implements the Provider interface for Google's Gemini API.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewGeminiProvider creates a new Gemini provider from configuration.
func NewGeminiProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &GeminiProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// Chat sends a non-streaming chat completion request to Gemini.
func (p *GeminiProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildRequestBody(req)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	model := req.Model
	if model == "" {
		model = p.model
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var gemResp geminiGenerateResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return nil, fmt.Errorf("gemini: unmarshal response: %w", err)
	}

	return p.convertResponse(&gemResp)
}

// ChatStream sends a streaming chat completion request to Gemini.
func (p *GeminiProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body := p.buildRequestBody(req)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	model := req.Model
	if model == "" {
		model = p.model
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", p.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("gemini: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 64)

	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Handle both SSE format and newline-delimited JSON.
			payload := line
			if strings.HasPrefix(line, "data: ") {
				payload = strings.TrimPrefix(line, "data: ")
			}

			var chunk geminiGenerateResponse
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				// Skip non-JSON lines (like SSE event type lines).
				continue
			}

			if len(chunk.Candidates) > 0 {
				candidate := chunk.Candidates[0]
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						ch <- StreamEvent{Type: "delta", Delta: part.Text}
					}
					if part.FunctionCall != nil {
						argsJSON, _ := json.Marshal(part.FunctionCall.Args)
						ch <- StreamEvent{
							Type: "tool_call",
							ToolCall: &ToolCall{
								ID:   fmt.Sprintf("call_%s", part.FunctionCall.Name),
								Name: part.FunctionCall.Name,
								Args: json.RawMessage(argsJSON),
							},
						}
					}
				}

				if candidate.FinishReason == "STOP" || candidate.FinishReason != "" {
					// Check for usage metadata.
					if chunk.UsageMetadata != nil {
						ch <- StreamEvent{
							Type: "done",
							Usage: &Usage{
								PromptTokens:     chunk.UsageMetadata.PromptTokenCount,
								CompletionTokens: chunk.UsageMetadata.CandidatesTokenCount,
								TotalTokens:      chunk.UsageMetadata.TotalTokenCount,
							},
						}
						return
					}
				}
			}
		}

		// If we exit the loop without sending done, send it now.
		ch <- StreamEvent{Type: "done"}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
			default:
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("gemini: stream read error: %w", err)}
			}
		}
	}()

	return ch, nil
}

// CountTokens estimates the token count using approximate calculation.
func (p *GeminiProvider) CountTokens(messages []Message) (int, error) {
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
func (p *GeminiProvider) ModelInfo() ModelMetadata {
	info := ModelMetadata{
		ID:            p.model,
		Provider:      "gemini",
		SupportsTools: true,
	}

	switch {
	case strings.Contains(p.model, "1.5-pro"):
		info.ContextWindow = 2097152
		info.MaxOutput = 8192
		info.SupportsVision = true
		info.CostPer1kIn = 0.00125
		info.CostPer1kOut = 0.005
	case strings.Contains(p.model, "1.5-flash"):
		info.ContextWindow = 1048576
		info.MaxOutput = 8192
		info.SupportsVision = true
		info.CostPer1kIn = 0.000075
		info.CostPer1kOut = 0.0003
	case strings.Contains(p.model, "2.0-flash"):
		info.ContextWindow = 1048576
		info.MaxOutput = 8192
		info.SupportsVision = true
		info.CostPer1kIn = 0.0001
		info.CostPer1kOut = 0.0004
	default:
		info.ContextWindow = 1048576
		info.MaxOutput = 8192
	}

	return info
}

// RegisterGemini registers the Gemini provider factory with the given registry.
func RegisterGemini(reg *Registry) {
	reg.Register("gemini", func(cfg ProviderConfig) (Provider, error) {
		return NewGeminiProvider(cfg)
	})
}

// --- Internal helpers ---

func (p *GeminiProvider) buildRequestBody(req ChatRequest) map[string]any {
	// Translate Hera messages to Gemini format.
	var contents []map[string]any
	var systemInstruction *map[string]any

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			si := map[string]any{
				"parts": []map[string]any{
					{"text": msg.Content},
				},
			}
			systemInstruction = &si
			continue
		}

		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}

		parts := []map[string]any{}

		if msg.Content != "" {
			parts = append(parts, map[string]any{"text": msg.Content})
		}

		// Handle tool responses.
		if msg.Role == RoleTool && msg.ToolCallID != "" {
			parts = []map[string]any{
				{
					"functionResponse": map[string]any{
						"name":     msg.Name,
						"response": map[string]any{"content": msg.Content},
					},
				},
			}
		}

		// Handle tool calls from assistant.
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			json.Unmarshal(tc.Args, &args)
			parts = append(parts, map[string]any{
				"functionCall": map[string]any{
					"name": tc.Name,
					"args": args,
				},
			})
		}

		contents = append(contents, map[string]any{
			"role":  role,
			"parts": parts,
		})
	}

	body := map[string]any{
		"contents": contents,
	}

	if systemInstruction != nil {
		body["systemInstruction"] = *systemInstruction
	}

	// Generation config.
	genConfig := map[string]any{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		genConfig["temperature"] = *req.Temperature
	}
	if len(req.Stop) > 0 {
		genConfig["stopSequences"] = req.Stop
	}
	if len(genConfig) > 0 {
		body["generationConfig"] = genConfig
	}

	// Tools.
	if len(req.Tools) > 0 {
		funcDecls := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			funcDecls[i] = map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  json.RawMessage(tool.Parameters),
			}
		}
		body["tools"] = []map[string]any{
			{"functionDeclarations": funcDecls},
		}

		// Gemini uses toolConfig.functionCallingConfig.mode with values
		// AUTO | ANY | NONE. Map the cross-provider string here.
		mode := "AUTO"
		switch req.ToolChoice {
		case "required":
			mode = "ANY"
		case "none":
			mode = "NONE"
		case "", "auto":
			mode = "AUTO"
		}
		body["toolConfig"] = map[string]any{
			"functionCallingConfig": map[string]any{"mode": mode},
		}
	}

	return body
}

func (p *GeminiProvider) convertResponse(resp *geminiGenerateResponse) (*ChatResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini: empty candidates in response")
	}

	candidate := resp.Candidates[0]
	msg := Message{
		Role: RoleAssistant,
	}

	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:   fmt.Sprintf("call_%s", part.FunctionCall.Name),
				Name: part.FunctionCall.Name,
				Args: json.RawMessage(argsJSON),
			})
		}
	}
	msg.Content = strings.Join(textParts, "")

	finishReason := strings.ToLower(candidate.FinishReason)
	if finishReason == "stop" {
		finishReason = "stop"
	}

	var usage Usage
	if resp.UsageMetadata != nil {
		usage = Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &ChatResponse{
		Message:      msg,
		Model:        p.model,
		Usage:        usage,
		FinishReason: finishReason,
	}, nil
}

// Gemini API response types.

type geminiGenerateResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage      `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
