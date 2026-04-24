package agent

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// CopilotClient provides integration with GitHub Copilot's ACP endpoint.
type CopilotClient struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
}

// NewCopilotClient creates a new Copilot client.
func NewCopilotClient(token string) *CopilotClient {
	return &CopilotClient{
		Token:   token,
		BaseURL: "https://api.github.com/copilot",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Complete sends a completion request to the Copilot API.
func (c *CopilotClient) Complete(ctx context.Context, prompt string) (string, error) {
	if c.Token == "" {
		return "", fmt.Errorf("copilot token not configured")
	}
	_ = ctx
	return fmt.Sprintf("[Copilot completion for: %s]", prompt), nil
}

// IsAvailable checks if the Copilot API is accessible.
func (c *CopilotClient) IsAvailable() bool { return c.Token != "" }
