package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ModelsDevEntry represents a model entry from the models.dev API.
type ModelsDevEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextLength int    `json:"context_length"`
	Pricing       struct {
		Prompt     float64 `json:"prompt"`
		Completion float64 `json:"completion"`
	} `json:"pricing"`
}

// ModelsDevClient fetches model data from the models.dev API.
type ModelsDevClient struct {
	BaseURL string
	Client  *http.Client
}

// NewModelsDevClient creates a new client for models.dev.
func NewModelsDevClient() *ModelsDevClient {
	return &ModelsDevClient{
		BaseURL: "https://models.dev/api/v1",
		Client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchModels retrieves the full model catalog from models.dev.
func (c *ModelsDevClient) FetchModels(ctx context.Context) ([]ModelsDevEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/models", nil)
	if err != nil { return nil, fmt.Errorf("create request: %w", err) }
	resp, err := c.Client.Do(req)
	if err != nil { return nil, fmt.Errorf("fetch models: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models.dev returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil { return nil, fmt.Errorf("read body: %w", err) }
	var models []ModelsDevEntry
	if err := json.Unmarshal(body, &models); err != nil { return nil, fmt.Errorf("parse models: %w", err) }
	return models, nil
}

// SearchModels searches for models matching the given query.
func (c *ModelsDevClient) SearchModels(ctx context.Context, query string) ([]ModelsDevEntry, error) {
	all, err := c.FetchModels(ctx)
	if err != nil { return nil, err }
	var results []ModelsDevEntry
	for _, m := range all {
		if contains(m.ID, query) || contains(m.Name, query) || contains(m.Provider, query) {
			results = append(results, m)
		}
	}
	return results, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
