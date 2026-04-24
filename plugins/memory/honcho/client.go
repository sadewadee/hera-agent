package honcho

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sadewadee/hera/internal/paths"
)

// ClientConfig holds configuration for the Honcho client.
type ClientConfig struct {
	Host        string `json:"host"`
	WorkspaceID string `json:"workspace"`
	APIKey      string `json:"apiKey"`
	Environment string `json:"environment"`
	BaseURL     string `json:"baseUrl"`
	PeerName    string `json:"peerName"`
	AIPeer      string `json:"aiPeer"`
	Enabled     bool   `json:"enabled"`

	RecallMode              string `json:"recallMode"`
	ContextTokens           int    `json:"contextTokens"`
	DialecticReasoningLevel string `json:"dialecticReasoningLevel"`
	DialecticMaxChars       int    `json:"dialecticMaxChars"`
	MessageMaxChars         int    `json:"messageMaxChars"`
}

// LoadConfig loads the Honcho configuration from the config chain:
// 1. $HERA_HOME/honcho.json
// 2. ~/.honcho/config.json
// 3. Environment variables
func LoadConfig() (*ClientConfig, error) {
	cfg := &ClientConfig{
		Host:                    "hera",
		WorkspaceID:             "hera",
		AIPeer:                  "hera",
		Environment:             "production",
		RecallMode:              "hybrid",
		DialecticReasoningLevel: "low",
		DialecticMaxChars:       600,
		MessageMaxChars:         25000,
	}

	// Try $HERA_HOME/honcho.json first (paths.HeraHome honours HERA_HOME env).
	localPath := filepath.Join(paths.HeraHome(), "honcho.json")
	globalPath := filepath.Join(os.Getenv("HOME"), ".honcho", "config.json")

	var configPath string
	if _, err := os.Stat(localPath); err == nil {
		configPath = localPath
	} else if _, err := os.Stat(globalPath); err == nil {
		configPath = globalPath
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err == nil {
				if v, ok := raw["apiKey"].(string); ok && v != "" {
					cfg.APIKey = v
				}
				if v, ok := raw["baseUrl"].(string); ok && v != "" {
					cfg.BaseURL = v
				}
				if v, ok := raw["workspace"].(string); ok && v != "" {
					cfg.WorkspaceID = v
				}
				if v, ok := raw["aiPeer"].(string); ok && v != "" {
					cfg.AIPeer = v
				}
				if v, ok := raw["peerName"].(string); ok && v != "" {
					cfg.PeerName = v
				}
				if v, ok := raw["environment"].(string); ok && v != "" {
					cfg.Environment = v
				}
				if v, ok := raw["recallMode"].(string); ok && v != "" {
					cfg.RecallMode = normalizeRecallMode(v)
				}
				if v, ok := raw["enabled"].(bool); ok {
					cfg.Enabled = v
				}
			}
		}
	}

	// Environment variable overrides
	if envKey := os.Getenv("HONCHO_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}
	if envURL := os.Getenv("HONCHO_BASE_URL"); envURL != "" {
		cfg.BaseURL = envURL
	}

	// Auto-enable if credentials present
	if cfg.APIKey != "" || cfg.BaseURL != "" {
		cfg.Enabled = true
	}

	return cfg, nil
}

func normalizeRecallMode(mode string) string {
	switch mode {
	case "context", "tools", "hybrid":
		return mode
	case "auto":
		return "hybrid"
	default:
		return "hybrid"
	}
}

// Client is a thin HTTP client for the Honcho REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a Honcho HTTP client from the given config.
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg.APIKey == "" && cfg.BaseURL == "" {
		return nil, fmt.Errorf("honcho API key or base URL required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.honcho.dev"
	}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = "local"
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// request performs an HTTP request to the Honcho API.
func (c *Client) request(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("honcho API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		slog.Debug("honcho API error", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("honcho API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
	}

	return result, nil
}
