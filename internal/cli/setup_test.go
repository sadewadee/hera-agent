package cli

import (
	"strings"
	"testing"
)

func TestBuildConfigYAML(t *testing.T) {
	t.Run("includes provider and model", func(t *testing.T) {
		yaml := buildConfigYAML("openai", "sk-test123", "", "gpt-4o", "helpful")
		if !strings.Contains(yaml, "default_provider: openai") {
			t.Error("config missing default_provider")
		}
		if !strings.Contains(yaml, "default_model: gpt-4o") {
			t.Error("config missing default_model")
		}
		if !strings.Contains(yaml, "personality: helpful") {
			t.Error("config missing personality")
		}
	})

	t.Run("includes provider section", func(t *testing.T) {
		yaml := buildConfigYAML("anthropic", "sk-ant-test", "", "claude-sonnet-4-20250514", "creative")
		if !strings.Contains(yaml, "providers:") {
			t.Error("config missing providers section")
		}
		if !strings.Contains(yaml, "type: anthropic") {
			t.Error("config missing provider type")
		}
	})

	t.Run("includes security section", func(t *testing.T) {
		yaml := buildConfigYAML("openai", "test", "", "gpt-4", "helpful")
		if !strings.Contains(yaml, "security:") {
			t.Error("config missing security section")
		}
		if !strings.Contains(yaml, "redact_pii: false") {
			t.Error("config missing redact_pii setting")
		}
		if !strings.Contains(yaml, "~/.ssh") {
			t.Error("config missing protected paths")
		}
	})

	t.Run("includes memory section", func(t *testing.T) {
		yaml := buildConfigYAML("gemini", "key", "", "gemini-2.0-flash", "concise")
		if !strings.Contains(yaml, "memory:") {
			t.Error("config missing memory section")
		}
		if !strings.Contains(yaml, "provider: sqlite") {
			t.Error("config missing memory provider")
		}
	})

	t.Run("includes compression settings", func(t *testing.T) {
		yaml := buildConfigYAML("openai", "key", "", "gpt-4", "helpful")
		if !strings.Contains(yaml, "compression:") {
			t.Error("config missing compression section")
		}
		if !strings.Contains(yaml, "threshold: 0.5") {
			t.Error("config missing compression threshold")
		}
		if !strings.Contains(yaml, "protected_turns: 5") {
			t.Error("config missing protected_turns")
		}
	})

	t.Run("includes api_key env reference when key provided", func(t *testing.T) {
		yaml := buildConfigYAML("openai", "sk-test", "", "gpt-4", "helpful")
		if !strings.Contains(yaml, "api_key: ${OPENAI_API_KEY}") {
			t.Error("config missing api_key env var reference")
		}
	})

	t.Run("omits api_key when empty", func(t *testing.T) {
		yaml := buildConfigYAML("compatible", "", "http://localhost:11434/v1", "local-model", "helpful")
		if strings.Contains(yaml, "api_key:") {
			t.Error("config should not contain api_key when key is empty")
		}
	})

	t.Run("includes cli skin setting", func(t *testing.T) {
		yaml := buildConfigYAML("openai", "key", "", "gpt-4", "helpful")
		if !strings.Contains(yaml, "skin: default") {
			t.Error("config missing cli skin setting")
		}
	})
}

func TestProviderChoices(t *testing.T) {
	t.Run("has at least 5 providers", func(t *testing.T) {
		if len(providerChoices) < 5 {
			t.Errorf("expected at least 5 providers, got %d", len(providerChoices))
		}
	})

	t.Run("each provider has name, type, and models", func(t *testing.T) {
		for _, p := range providerChoices {
			if p.Name == "" {
				t.Error("provider has empty Name")
			}
			if p.Type == "" {
				t.Errorf("provider %q has empty Type", p.Name)
			}
			if len(p.Models) == 0 {
				t.Errorf("provider %q has no models", p.Name)
			}
		}
	})

	t.Run("local provider has no env var", func(t *testing.T) {
		for _, p := range providerChoices {
			if strings.Contains(p.Name, "compatible") {
				if p.EnvVar != "" {
					t.Errorf("local provider should have empty EnvVar, got %q", p.EnvVar)
				}
				return
			}
		}
		t.Error("did not find local provider in choices")
	})
}

func TestPersonalityChoices(t *testing.T) {
	t.Run("has at least 5 personalities", func(t *testing.T) {
		if len(personalityChoices) < 5 {
			t.Errorf("expected at least 5 personalities, got %d", len(personalityChoices))
		}
	})

	t.Run("includes helpful", func(t *testing.T) {
		found := false
		for _, p := range personalityChoices {
			if p == "helpful" {
				found = true
				break
			}
		}
		if !found {
			t.Error("personality choices missing 'helpful'")
		}
	})
}
