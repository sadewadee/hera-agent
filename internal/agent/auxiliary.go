package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/llm"
)

// AuxiliaryClient provides vision and web-extract helpers that use an LLM
// provider for analysis tasks.
type AuxiliaryClient struct {
	provider   llm.Provider
	httpClient *http.Client
}

// NewAuxiliaryClient creates a new AuxiliaryClient.
func NewAuxiliaryClient(provider llm.Provider) *AuxiliaryClient {
	return &AuxiliaryClient{
		provider: provider,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DescribeImage sends an image URL to a vision-capable model and returns a text
// description of the image contents.
func (ac *AuxiliaryClient) DescribeImage(ctx context.Context, imageURL string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("image URL is required")
	}

	// Check that the provider supports vision.
	info := ac.provider.ModelInfo()
	if !info.SupportsVision {
		return "", fmt.Errorf("current model %s does not support vision", info.ID)
	}

	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are an image analysis assistant. Describe the image in detail.",
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Describe this image in detail: %s", imageURL),
		},
	}

	resp, err := ac.provider.Chat(ctx, llm.ChatRequest{
		Messages:  messages,
		MaxTokens: 1024,
	})
	if err != nil {
		return "", fmt.Errorf("describe image: %w", err)
	}

	return strings.TrimSpace(resp.Message.Content), nil
}

// ExtractWebContent fetches a URL, extracts the text content, and optionally
// summarizes it using the LLM provider.
func (ac *AuxiliaryClient) ExtractWebContent(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("URL is required")
	}

	// Fetch the URL content.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Hera/1.0 (Web Content Extractor)")
	req.Header.Set("Accept", "text/html,text/plain,application/json")

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	// Read body with a 512KB limit to avoid excessive memory use.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	rawText := string(body)

	// Strip HTML tags for a rough text extraction.
	text := stripHTMLTags(rawText)
	text = collapseWhitespace(text)

	// If the text is short enough, return it directly.
	if len(text) < 2000 {
		return text, nil
	}

	// Summarize long content with the LLM.
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "Summarize the following web page content concisely, preserving key information.",
		},
		{
			Role:    llm.RoleUser,
			Content: truncateText(text, 8000),
		},
	}

	llmResp, err := ac.provider.Chat(ctx, llm.ChatRequest{
		Messages:  messages,
		MaxTokens: 1024,
	})
	if err != nil {
		// If summarization fails, return truncated raw text.
		return truncateText(text, 4000), nil
	}

	return strings.TrimSpace(llmResp.Message.Content), nil
}

// stripHTMLTags removes HTML tags from text. This is a simple implementation
// that handles the common case without requiring an HTML parser dependency.
func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteRune(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// collapseWhitespace replaces runs of whitespace with a single space.
func collapseWhitespace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// truncateText truncates text to maxLen characters, adding an ellipsis if truncated.
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
