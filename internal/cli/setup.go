package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
)

// providerChoices lists the available LLM providers for setup.
var providerChoices = []struct {
	Name   string
	Type   string
	EnvVar string
	Models []string
}{
	{"OpenAI", "openai", "OPENAI_API_KEY", []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1"}},
	{"Anthropic", "anthropic", "ANTHROPIC_API_KEY", []string{"claude-sonnet-4-20250514", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"}},
	{"Google Gemini", "gemini", "GEMINI_API_KEY", []string{"gemini-2.0-flash", "gemini-2.0-pro", "gemini-1.5-pro"}},
	{"Mistral", "mistral", "MISTRAL_API_KEY", []string{"mistral-large-latest", "mistral-medium-latest", "codestral-latest"}},
	{"OpenRouter", "openrouter", "OPENROUTER_API_KEY", []string{"openai/gpt-4o", "anthropic/claude-sonnet-4-20250514", "google/gemini-2.0-flash"}},
	{"OpenAI-compatible", "compatible", "", []string{"local-model"}},
}

// personalityChoices lists available personalities.
var personalityChoices = []string{
	"helpful",
	"creative",
	"concise",
	"technical",
	"friendly",
	"professional",
	"witty",
}

// RunSetupWizard runs the interactive setup wizard.
func RunSetupWizard() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("=== Hera Setup Wizard ===")
	fmt.Println()
	fmt.Println("This wizard will help you configure Hera.")
	fmt.Println()

	// Step 1: Provider selection
	fmt.Println("Step 1: Choose your LLM provider")
	fmt.Println()
	for i, p := range providerChoices {
		fmt.Printf("  %d. %s\n", i+1, p.Name)
	}
	fmt.Println()

	providerIdx := promptChoice(reader, "Select provider", 1, len(providerChoices))
	provider := providerChoices[providerIdx-1]

	// Step 2: API Key + Base URL
	var apiKey string
	var baseURL string

	if provider.EnvVar != "" {
		fmt.Printf("\nStep 2: Enter your %s API key\n", provider.Name)
		fmt.Printf("  (You can also set the %s environment variable)\n", provider.EnvVar)
		fmt.Print("  API Key: ")
		apiKey = readLine(reader)
		apiKey = strings.TrimSpace(apiKey)

		// Optional custom base URL
		fmt.Printf("\n  Custom base URL? (leave empty for default %s endpoint)\n", provider.Name)
		fmt.Print("  Base URL: ")
		baseURL = readLine(reader)
		baseURL = strings.TrimSpace(baseURL)
	} else {
		// OpenAI-compatible: base URL required, API key optional
		fmt.Println("\nStep 2: Enter the base URL for your endpoint")
		fmt.Print("  Base URL (default: http://localhost:11434/v1): ")
		baseURL = readLine(reader)
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}

		fmt.Print("  API Key (leave empty if not needed): ")
		apiKey = readLine(reader)
		apiKey = strings.TrimSpace(apiKey)
	}

	// Step 3: Model selection
	// Try to auto-fetch models from the endpoint
	availableModels := provider.Models
	if baseURL != "" {
		fmt.Printf("\n  Fetching models from %s ...\n", baseURL)
		fetched, err := llm.FetchModels(baseURL, apiKey)
		if err != nil {
			fmt.Printf("  Could not fetch models: %v\n", err)
			fmt.Println("  Using default model list instead.")
		} else if len(fetched) > 0 {
			fmt.Printf("  Found %d models!\n", len(fetched))
			availableModels = fetched
		}
	}

	fmt.Printf("\nStep 3: Choose your default model\n")
	fmt.Println()
	for i, m := range availableModels {
		fmt.Printf("  %d. %s\n", i+1, m)
	}
	fmt.Printf("  %d. Custom (type your own)\n", len(availableModels)+1)
	fmt.Println()

	modelIdx := promptChoice(reader, "Select model", 1, len(availableModels)+1)
	var model string
	if modelIdx <= len(availableModels) {
		model = availableModels[modelIdx-1]
	} else {
		fmt.Print("  Model name: ")
		model = readLine(reader)
	}

	// Step 4: Personality
	fmt.Println("\nStep 4: Choose a personality")
	fmt.Println()
	for i, p := range personalityChoices {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
	fmt.Println()

	personalityIdx := promptChoice(reader, "Select personality", 1, len(personalityChoices))
	personality := personalityChoices[personalityIdx-1]

	// Write config
	heraDir := config.HeraDir()
	if err := os.MkdirAll(heraDir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	configPath := filepath.Join(heraDir, "config.yaml")
	configContent := buildConfigYAML(provider.Type, apiKey, baseURL, model, personality)

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Write .env with API key and base URL
	envPath := filepath.Join(heraDir, ".env")
	var envLines []string
	if apiKey != "" && provider.EnvVar != "" {
		envLines = append(envLines, fmt.Sprintf("%s=%s", provider.EnvVar, apiKey))
	}
	if baseURL != "" {
		envKey := strings.ToUpper(provider.Type) + "_BASE_URL"
		envLines = append(envLines, fmt.Sprintf("%s=%s", envKey, baseURL))
	}
	if len(envLines) > 0 {
		if err := os.WriteFile(envPath, []byte(strings.Join(envLines, "\n")+"\n"), 0o600); err != nil {
			return fmt.Errorf("write .env: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Run 'hera chat' to start chatting.")
	fmt.Println("Run 'hera doctor' to verify your setup.")

	return nil
}

func promptChoice(reader *bufio.Reader, prompt string, min, max int) int {
	for {
		fmt.Printf("  %s [%d-%d]: ", prompt, min, max)
		line := readLine(reader)
		var choice int
		if _, err := fmt.Sscanf(line, "%d", &choice); err == nil && choice >= min && choice <= max {
			return choice
		}
		fmt.Printf("  Please enter a number between %d and %d.\n", min, max)
	}
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func buildConfigYAML(providerType, apiKey, baseURL, model, personality string) string {
	var sb strings.Builder
	sb.WriteString("# Hera Configuration\n")
	sb.WriteString("# Generated by 'hera setup'\n\n")

	sb.WriteString("agent:\n")
	sb.WriteString(fmt.Sprintf("  default_provider: %s\n", providerType))
	sb.WriteString(fmt.Sprintf("  default_model: %s\n", model))
	sb.WriteString(fmt.Sprintf("  personality: %s\n", personality))
	sb.WriteString("  max_tool_calls: 20\n")
	sb.WriteString("  smart_routing: true\n")
	sb.WriteString("  compression:\n")
	sb.WriteString("    enabled: true\n")
	sb.WriteString("    threshold: 0.5\n")
	sb.WriteString("    target_ratio: 0.2\n")
	sb.WriteString("    protected_turns: 5\n")
	sb.WriteString("\n")

	sb.WriteString("providers:\n")
	sb.WriteString(fmt.Sprintf("  %s:\n", providerType))
	sb.WriteString(fmt.Sprintf("    type: %s\n", providerType))
	if baseURL != "" {
		sb.WriteString(fmt.Sprintf("    base_url: %s\n", baseURL))
	}
	if apiKey != "" {
		envKey := strings.ToUpper(providerType) + "_API_KEY"
		sb.WriteString(fmt.Sprintf("    api_key: ${%s}\n", envKey))
	}
	sb.WriteString(fmt.Sprintf("    models:\n      - %s\n", model))
	sb.WriteString("\n")

	sb.WriteString("memory:\n")
	sb.WriteString("  provider: sqlite\n")
	sb.WriteString("  max_results: 10\n")
	sb.WriteString("\n")

	sb.WriteString("security:\n")
	sb.WriteString("  redact_pii: false\n")
	sb.WriteString("  dangerous_approve: true\n")
	sb.WriteString("  protected_paths:\n")
	sb.WriteString("    - ~/.ssh\n")
	sb.WriteString("    - ~/.gnupg\n")
	sb.WriteString("    - ~/.aws/credentials\n")
	sb.WriteString("\n")

	sb.WriteString("cli:\n")
	sb.WriteString("  skin: default\n")

	return sb.String()
}
