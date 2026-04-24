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
	ollamaDefaultBaseURL = "http://localhost:11434"
)

// OllamaProvider implements the Provider interface for local Ollama instances.
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama provider from configuration.
func NewOllamaProvider(cfg ProviderConfig) (Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ollamaDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second // Longer timeout for local models.
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// Chat sends a non-streaming chat completion request to Ollama.
func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildRequestBody(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	url := p.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal response: %w", err)
	}

	return p.convertResponse(&ollamaResp), nil
}

// ChatStream sends a streaming chat completion request to Ollama.
func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body := p.buildRequestBody(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	url := p.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("ollama: API error (status %d): %s", httpResp.StatusCode, string(respBody))
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

			var chunk ollamaChatResponse
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("ollama: parse stream chunk: %w", err)}
				return
			}

			if chunk.Message.Content != "" {
				ch <- StreamEvent{Type: "delta", Delta: chunk.Message.Content}
			}

			if chunk.Done {
				ch <- StreamEvent{
					Type: "done",
					Usage: &Usage{
						PromptTokens:     chunk.PromptEvalCount,
						CompletionTokens: chunk.EvalCount,
						TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
					},
				}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
			default:
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("ollama: stream read error: %w", err)}
			}
		}
	}()

	return ch, nil
}

// CountTokens estimates the token count using approximate calculation.
func (p *OllamaProvider) CountTokens(messages []Message) (int, error) {
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
func (p *OllamaProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "ollama",
		ContextWindow: 8192,
		MaxOutput:     4096,
		SupportsTools: false,
	}
}

// RegisterOllama registers the Ollama provider factory with the given registry.
func RegisterOllama(reg *Registry) {
	reg.Register("ollama", func(cfg ProviderConfig) (Provider, error) {
		return NewOllamaProvider(cfg)
	})
}

// --- Internal helpers ---

func (p *OllamaProvider) buildRequestBody(req ChatRequest, stream bool) map[string]any {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]map[string]any, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		})
	}

	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   stream,
	}

	if req.Temperature != nil {
		if body["options"] == nil {
			body["options"] = map[string]any{}
		}
		body["options"].(map[string]any)["temperature"] = *req.Temperature
	}

	return body
}

func (p *OllamaProvider) convertResponse(resp *ollamaChatResponse) *ChatResponse {
	return &ChatResponse{
		Message: Message{
			Role:    Role(resp.Message.Role),
			Content: resp.Message.Content,
		},
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		FinishReason: "stop",
	}
}

// Ollama API response types.

type ollamaChatResponse struct {
	Model           string        `json:"model"`
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	EvalCount       int           `json:"eval_count"`
	PromptEvalCount int           `json:"prompt_eval_count"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
