package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAPIKey_FromConfig(t *testing.T) {
	cfg := &Config{
		Provider: map[string]ProviderEntry{
			"openai": {APIKey: "sk-from-config"},
		},
	}
	got := ResolveAPIKey(cfg, "openai")
	if got != "sk-from-config" {
		t.Errorf("ResolveAPIKey() = %q, want %q", got, "sk-from-config")
	}
}

func TestResolveAPIKey_FromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-from-env")
	got := ResolveAPIKey(nil, "openai")
	if got != "sk-from-env" {
		t.Errorf("ResolveAPIKey() = %q, want %q", got, "sk-from-env")
	}
}

func TestResolveAPIKey_ConfigPrecedence(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-env")
	cfg := &Config{
		Provider: map[string]ProviderEntry{
			"anthropic": {APIKey: "sk-from-config"},
		},
	}
	got := ResolveAPIKey(cfg, "anthropic")
	if got != "sk-from-config" {
		t.Errorf("ResolveAPIKey() = %q, want %q (config should take precedence)", got, "sk-from-config")
	}
}

func TestResolveAPIKey_UnknownProvider(t *testing.T) {
	got := ResolveAPIKey(nil, "nonexistent-provider")
	if got != "" {
		t.Errorf("ResolveAPIKey() = %q, want empty string for unknown provider", got)
	}
}

func TestResolveAPIKey_Ollama(t *testing.T) {
	// Ollama does not require an API key, so no env var is mapped.
	got := ResolveAPIKey(nil, "ollama")
	if got != "" {
		t.Errorf("ResolveAPIKey() = %q, want empty string for ollama (no key needed)", got)
	}
}

func TestResolveAPIKey_NilConfig(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gemini-key-123")
	got := ResolveAPIKey(nil, "gemini")
	if got != "gemini-key-123" {
		t.Errorf("ResolveAPIKey(nil, gemini) = %q, want %q", got, "gemini-key-123")
	}
}

func TestResolveAPIKey_EmptyConfigKey(t *testing.T) {
	t.Setenv("MISTRAL_API_KEY", "mistral-env-key")
	cfg := &Config{
		Provider: map[string]ProviderEntry{
			"mistral": {APIKey: ""}, // empty in config
		},
	}
	got := ResolveAPIKey(cfg, "mistral")
	if got != "mistral-env-key" {
		t.Errorf("ResolveAPIKey() = %q, want %q (should fall back to env when config key is empty)", got, "mistral-env-key")
	}
}

func TestResolveAPIKey_AllProviders(t *testing.T) {
	// Verify every known provider maps to the correct env var.
	tests := []struct {
		provider string
		envVar   string
	}{
		{"openai", "OPENAI_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"mistral", "MISTRAL_API_KEY"},
		{"openrouter", "OPENROUTER_API_KEY"},
		{"nous", "NOUS_API_KEY"},
		{"huggingface", "HF_TOKEN"},
		{"glm", "GLM_API_KEY"},
		{"kimi", "MOONSHOT_API_KEY"},
		{"minimax", "MINIMAX_API_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			val := "test-key-" + tt.provider
			t.Setenv(tt.envVar, val)
			got := ResolveAPIKey(nil, tt.provider)
			if got != val {
				t.Errorf("ResolveAPIKey(nil, %q) = %q, want %q", tt.provider, got, val)
			}
		})
	}
}

func TestHeraDir(t *testing.T) {
	// Ensure HERA_HOME is unset so the ~/.hera default path is exercised.
	t.Setenv("HERA_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}
	want := filepath.Join(home, ".hera")
	got := HeraDir()
	if got != want {
		t.Errorf("HeraDir() = %q, want %q", got, want)
	}
}

func TestHeraDirRespectsEnv(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("HERA_HOME", custom)

	got := HeraDir()
	if got != custom {
		t.Errorf("HeraDir() = %q, want %q (HERA_HOME override ignored)", got, custom)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Point config search at a temp dir with no config file so Load uses defaults.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Agent.DefaultProvider != "openai" {
		t.Errorf("default provider = %q, want %q", cfg.Agent.DefaultProvider, "openai")
	}
	if cfg.Agent.DefaultModel != "gpt-4o" {
		t.Errorf("default model = %q, want %q", cfg.Agent.DefaultModel, "gpt-4o")
	}
	if cfg.Agent.MaxToolCalls != 20 {
		t.Errorf("max tool calls = %d, want %d", cfg.Agent.MaxToolCalls, 20)
	}
	if !cfg.Agent.Compression.Enabled {
		t.Error("compression should be enabled by default")
	}
	if cfg.Memory.Provider != "sqlite" {
		t.Errorf("memory provider = %q, want %q", cfg.Memory.Provider, "sqlite")
	}
}

func TestValidate_CatchesMissingDefaults(t *testing.T) {
	cfg := &Config{} // all zero values
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "default_provider") || !strings.Contains(msg, "default_model") {
		t.Errorf("expected default_provider + default_model errors, got:\n%s", msg)
	}
}

func TestValidate_CatchesUnknownDefaultProvider(t *testing.T) {
	cfg := &Config{
		Provider: map[string]ProviderEntry{"openai": {APIKey: "x"}},
	}
	cfg.Agent.DefaultProvider = "bogus"
	cfg.Agent.DefaultModel = "gpt-4o"
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for unknown default_provider")
	}
	if !strings.Contains(err.Error(), "no entry under providers") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestValidate_CatchesEnabledPlatformWithoutToken(t *testing.T) {
	cfg := &Config{
		Provider: map[string]ProviderEntry{"openai": {APIKey: "x"}},
	}
	cfg.Agent.DefaultProvider = "openai"
	cfg.Agent.DefaultModel = "gpt-4o"
	cfg.Gateway.Platforms = map[string]PlatformConfig{
		"telegram": {Enabled: true, Token: ""},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for tokenless enabled platform")
	}
	if !strings.Contains(err.Error(), "telegram") || !strings.Contains(err.Error(), "token is empty") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestValidate_AllowsTokenlessPlatforms(t *testing.T) {
	cfg := &Config{
		Provider: map[string]ProviderEntry{"openai": {APIKey: "x"}},
	}
	cfg.Agent.DefaultProvider = "openai"
	cfg.Agent.DefaultModel = "gpt-4o"
	cfg.Gateway.Platforms = map[string]PlatformConfig{
		"cli":       {Enabled: true, Token: ""},
		"apiserver": {Enabled: true, Token: ""},
		"webhook":   {Enabled: true, Token: ""},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("cli/apiserver/webhook should not require token, got: %v", err)
	}
}

func TestValidate_CatchesUnknownMemoryProvider(t *testing.T) {
	cfg := &Config{
		Provider: map[string]ProviderEntry{"openai": {APIKey: "x"}},
	}
	cfg.Agent.DefaultProvider = "openai"
	cfg.Agent.DefaultModel = "gpt-4o"
	cfg.Memory.Provider = "redis" // not supported
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported memory provider")
	}
	if !strings.Contains(err.Error(), "redis") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("UserHomeDir unavailable")
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"tilde slash", "~/.hera/hera.db", filepath.Join(home, ".hera", "hera.db")},
		{"bare tilde", "~", home},
		{"absolute untouched", "/tmp/foo", "/tmp/foo"},
		{"relative untouched", "configs/file.yaml", "configs/file.yaml"},
		{"empty untouched", "", ""},
		{"tilde in middle untouched", "foo/~/bar", "foo/~/bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpandPath(tc.in)
			if got != tc.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoad_ExpandsTildePaths(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	heraDir := filepath.Join(tmp, ".hera")
	if err := os.MkdirAll(heraDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgYAML := `agent:
  default_provider: openai
  default_model: gpt-4o
  soul_file: ~/.hera/SOUL.md
providers:
  openai:
    api_key: sk-test
memory:
  provider: sqlite
  db_path: ~/.hera/hera.db
security:
  protected_paths:
    - ~/.ssh
    - /etc/passwd
`
	if err := os.WriteFile(filepath.Join(heraDir, "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wantDB := filepath.Join(tmp, ".hera", "hera.db")
	if cfg.Memory.DBPath != wantDB {
		t.Errorf("db_path = %q, want %q", cfg.Memory.DBPath, wantDB)
	}
	wantSoul := filepath.Join(tmp, ".hera", "SOUL.md")
	if cfg.Agent.SoulFile != wantSoul {
		t.Errorf("soul_file = %q, want %q", cfg.Agent.SoulFile, wantSoul)
	}
	if len(cfg.Security.ProtectedPaths) != 2 {
		t.Fatalf("protected_paths len = %d, want 2", len(cfg.Security.ProtectedPaths))
	}
	wantSSH := filepath.Join(tmp, ".ssh")
	if cfg.Security.ProtectedPaths[0] != wantSSH {
		t.Errorf("protected_paths[0] = %q, want %q", cfg.Security.ProtectedPaths[0], wantSSH)
	}
	if cfg.Security.ProtectedPaths[1] != "/etc/passwd" {
		t.Errorf("protected_paths[1] should remain absolute, got %q", cfg.Security.ProtectedPaths[1])
	}
}

func TestValidate_HappyPath(t *testing.T) {
	cfg := &Config{
		Provider: map[string]ProviderEntry{
			"openai": {APIKey: "sk-test"},
		},
	}
	cfg.Agent.DefaultProvider = "openai"
	cfg.Agent.DefaultModel = "gpt-4o"
	cfg.Memory.Provider = "sqlite"
	cfg.Gateway.Platforms = map[string]PlatformConfig{
		"telegram": {Enabled: true, Token: "bot-token"},
		"discord":  {Enabled: false}, // disabled, no token required
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("happy config should validate, got: %v", err)
	}
}

func TestLoad_ResolvesPersonalityByName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	heraDir := filepath.Join(tmpDir, ".hera")
	if err := os.MkdirAll(heraDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgYAML := "agent:\n  personality: kawaii\n"
	if err := os.WriteFile(filepath.Join(heraDir, "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Agent.Personality == "kawaii" {
		t.Fatalf("personality still literal %q, expected resolved guidelines", cfg.Agent.Personality)
	}
	// Signature marker from the expanded kawaii profile.
	if !strings.Contains(cfg.Agent.Personality, "Ohayou gozaimasu") {
		t.Errorf("resolved personality missing expected marker; got:\n%s", cfg.Agent.Personality)
	}
}

func TestLoad_ExpandsPlatformEnvRefs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("TELEGRAM_BOT_TOKEN", "1234:abcXYZ")
	t.Setenv("SLACK_APP_TOKEN", "xapp-1-secret")

	heraDir := filepath.Join(tmpDir, ".hera")
	if err := os.MkdirAll(heraDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgYAML := `
gateway:
  platforms:
    telegram:
      enabled: true
      token: ${TELEGRAM_BOT_TOKEN}
    slack:
      enabled: true
      token: xoxb-literal
      extra:
        app_token: ${SLACK_APP_TOKEN}
`
	if err := os.WriteFile(filepath.Join(heraDir, "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tg := cfg.Gateway.Platforms["telegram"]
	if tg.Token != "1234:abcXYZ" {
		t.Errorf("telegram.token = %q, want expansion of ${TELEGRAM_BOT_TOKEN}", tg.Token)
	}

	sl := cfg.Gateway.Platforms["slack"]
	if sl.Token != "xoxb-literal" {
		t.Errorf("slack.token literal changed: %q", sl.Token)
	}
	if sl.Extra["app_token"] != "xapp-1-secret" {
		t.Errorf("slack.extra.app_token = %q, want expansion", sl.Extra["app_token"])
	}
}
