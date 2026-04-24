// Package cli provides the Hera CLI application.
//
// runtime_provider.go implements shared runtime provider resolution for
// CLI, gateway, cron, and helpers. Resolves credentials and base URLs
// for the target LLM provider.
package cli

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// RuntimeProvider holds resolved provider credentials and configuration.
type RuntimeProvider struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	APIMode string `json:"api_mode"`
}

// ResolveRuntimeProvider resolves credentials for the requested provider.
func ResolveRuntimeProvider(requested string) (*RuntimeProvider, error) {
	provider := strings.TrimSpace(strings.ToLower(requested))
	if provider == "" {
		return nil, fmt.Errorf("no provider specified")
	}

	slog.Debug("resolving runtime provider", "provider", provider)

	result := &RuntimeProvider{}

	// Try environment variables for API key.
	envVarNames := providerEnvVars(provider)
	for _, envVar := range envVarNames {
		if key := os.Getenv(envVar); key != "" {
			result.APIKey = key
			break
		}
	}

	// Resolve base URL.
	result.BaseURL = providerBaseURL(provider)

	// Detect API mode.
	result.APIMode = DetectAPIModeForURL(result.BaseURL)

	if result.APIKey == "" && !isLocalProvider(provider, result.BaseURL) {
		return nil, fmt.Errorf("no API key found for provider '%s' (checked: %s)",
			provider, strings.Join(envVarNames, ", "))
	}

	return result, nil
}

// DetectAPIModeForURL auto-detects api_mode from the resolved base URL.
func DetectAPIModeForURL(baseURL string) string {
	normalized := strings.TrimSpace(strings.ToLower(strings.TrimRight(baseURL, "/")))
	if strings.Contains(normalized, "api.openai.com") && !strings.Contains(normalized, "openrouter") {
		return "codex_responses"
	}
	return ""
}

// AutoDetectLocalModel queries a local server for its model name.
func AutoDetectLocalModel(baseURL string) string {
	if baseURL == "" {
		return ""
	}

	url := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(url, "/v1") {
		url += "/v1"
	}
	url += "/models"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Simplified: just return empty for now. Full JSON decoding
	// would extract the model ID from the response.
	return ""
}

func providerEnvVars(provider string) []string {
	envMap := map[string][]string{
		"openrouter":  {"OPENROUTER_API_KEY"},
		"anthropic":   {"ANTHROPIC_API_KEY"},
		"openai":      {"OPENAI_API_KEY"},
		"gemini":      {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		"deepseek":    {"DEEPSEEK_API_KEY"},
		"nous":        {"NOUS_API_KEY"},
		"mistral":     {"MISTRAL_API_KEY"},
		"kimi":        {"KIMI_API_KEY", "MOONSHOT_API_KEY"},
		"minimax":     {"MINIMAX_API_KEY"},
		"glm":         {"GLM_API_KEY", "ZHIPU_API_KEY"},
		"huggingface": {"HF_TOKEN", "HUGGINGFACE_API_KEY"},
		"ollama":      {"OLLAMA_API_KEY"},
		"custom":      {"CUSTOM_API_KEY"},
	}

	if vars, ok := envMap[provider]; ok {
		return vars
	}

	// Generate default env var name.
	upper := strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
	return []string{upper + "_API_KEY"}
}

func providerBaseURL(provider string) string {
	urlMap := map[string]string{
		"openrouter": "https://openrouter.ai/api/v1",
		"anthropic":  "https://api.anthropic.com",
		"openai":     "https://api.openai.com/v1",
		"deepseek":   "https://api.deepseek.com",
		"mistral":    "https://api.mistral.ai/v1",
		"gemini":     "https://generativelanguage.googleapis.com",
		"ollama":     "http://localhost:11434",
	}
	if url, ok := urlMap[provider]; ok {
		return url
	}
	return ""
}

func isLocalProvider(provider, baseURL string) bool {
	if provider == "custom" || provider == "local" || provider == "ollama" {
		return true
	}
	return strings.Contains(baseURL, "localhost") || strings.Contains(baseURL, "127.0.0.1")
}

// NormalizeCustomProviderName normalizes a custom provider name.
func NormalizeCustomProviderName(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, " ", "-")))
}
