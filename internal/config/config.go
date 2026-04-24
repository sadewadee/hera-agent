package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/personalities"
)

// Config is the root configuration for Hera.
type Config struct {
	// ConfigVersion is written by `hera init` and `hera setup` to track
	// which binary version last seeded this config. `hera init` uses it
	// to decide whether a light re-seed (new bundled skills only) is
	// sufficient or a full wizard re-run is needed.
	ConfigVersion string                   `json:"_config_version" yaml:"_config_version" mapstructure:"_config_version"`
	Agent         AgentConfig              `json:"agent" yaml:"agent" mapstructure:"agent"`
	Provider      map[string]ProviderEntry `json:"providers" yaml:"providers" mapstructure:"providers"`
	Memory        MemoryConfig             `json:"memory" yaml:"memory" mapstructure:"memory"`
	Gateway       GatewayConfig            `json:"gateway" yaml:"gateway" mapstructure:"gateway"`
	CLI           CLIConfig                `json:"cli" yaml:"cli" mapstructure:"cli"`
	Cron          CronConfig               `json:"cron" yaml:"cron" mapstructure:"cron"`
	Security      SecurityConfig           `json:"security" yaml:"security" mapstructure:"security"`
	CustomTools   []CustomToolConfig       `json:"tools" yaml:"tools" mapstructure:"tools"`
	Hooks         []HookConfig             `json:"hooks" yaml:"hooks" mapstructure:"hooks"`
	MCPServers    []MCPServerEntry         `json:"mcp_servers" yaml:"mcp_servers" mapstructure:"mcp_servers"`
}

// MCPServerEntry configures an external MCP server to connect to.
type MCPServerEntry struct {
	Name    string            `json:"name" yaml:"name" mapstructure:"name"`
	Command string            `json:"command" yaml:"command" mapstructure:"command"`
	Args    []string          `json:"args,omitempty" yaml:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty" mapstructure:"env"`
	// Mode: "daemon" keeps the subprocess alive, "on_demand" (default)
	// kills it after IdleTimeout and respawns on next call.
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty" mapstructure:"mode"`
	// IdleTimeout is parsed as a Go duration string ("5m", "30s").
	// Default 5m. Only applies when Mode == on_demand.
	IdleTimeout string `json:"idle_timeout,omitempty" yaml:"idle_timeout,omitempty" mapstructure:"idle_timeout"`
}

// CustomToolConfig defines a user-defined tool that runs a shell command or HTTP call.
type CustomToolConfig struct {
	Name        string            `json:"name" yaml:"name" mapstructure:"name"`
	Description string            `json:"description" yaml:"description" mapstructure:"description"`
	Type        string            `json:"type" yaml:"type" mapstructure:"type"` // "command", "http", "script"
	Command     string            `json:"command,omitempty" yaml:"command,omitempty" mapstructure:"command"`
	URL         string            `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"`
	Method      string            `json:"method,omitempty" yaml:"method,omitempty" mapstructure:"method"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	Parameters  []ToolParam       `json:"parameters,omitempty" yaml:"parameters,omitempty" mapstructure:"parameters"`
	Timeout     int               `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"` // seconds
}

// ToolParam defines a parameter for a custom tool.
type ToolParam struct {
	Name        string `json:"name" yaml:"name" mapstructure:"name"`
	Type        string `json:"type" yaml:"type" mapstructure:"type"` // "string", "integer", "boolean"
	Description string `json:"description" yaml:"description" mapstructure:"description"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty" mapstructure:"required"`
}

// HookConfig defines a user-defined hook that runs at agent lifecycle events.
type HookConfig struct {
	Name    string `json:"name" yaml:"name" mapstructure:"name"`
	Event   string `json:"event" yaml:"event" mapstructure:"event"` // "before_message", "after_message", "before_tool", "after_tool", "on_error"
	Type    string `json:"type" yaml:"type" mapstructure:"type"`    // "command", "http", "script"
	Command string `json:"command,omitempty" yaml:"command,omitempty" mapstructure:"command"`
	URL     string `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"`
	Async   bool   `json:"async,omitempty" yaml:"async,omitempty" mapstructure:"async"`
}

// RoutingConfig controls which models are used for simple vs complex queries
// when smart_routing is enabled. All fields are optional; unset fields fall
// back to the DefaultRoutingConfig values in internal/agent/routing.go.
type RoutingConfig struct {
	CheapModel          string `json:"cheap_model,omitempty" yaml:"cheap_model,omitempty" mapstructure:"cheap_model"`
	CapableModel        string `json:"capable_model,omitempty" yaml:"capable_model,omitempty" mapstructure:"capable_model"`
	ShortThresholdChars int    `json:"short_threshold_chars,omitempty" yaml:"short_threshold_chars,omitempty" mapstructure:"short_threshold_chars"`
}

// BudgetConfig configures per-agent token and cost limits.
// Zero values mean unlimited. Mirrors agent.BudgetConfig but lives in config
// so it can be loaded from YAML and passed as part of AgentConfig.
type BudgetConfig struct {
	// MaxTokens is the maximum cumulative prompt+completion tokens. 0 = unlimited.
	MaxTokens int `json:"max_tokens" yaml:"max_tokens" mapstructure:"max_tokens"`
	// MaxUSD is the maximum cumulative cost in USD. 0 = unlimited.
	MaxUSD float64 `json:"max_usd" yaml:"max_usd" mapstructure:"max_usd"`
	// Window is the rolling time window. Empty string / zero = no reset.
	Window string `json:"window,omitempty" yaml:"window,omitempty" mapstructure:"window"`
}

// AgentConfig configures the core agent behavior.
type AgentConfig struct {
	DefaultProvider     string            `json:"default_provider" yaml:"default_provider" mapstructure:"default_provider"`
	DefaultModel        string            `json:"default_model" yaml:"default_model" mapstructure:"default_model"`
	Personality         string            `json:"personality" yaml:"personality" mapstructure:"personality"`
	SoulFile            string            `json:"soul_file" yaml:"soul_file" mapstructure:"soul_file"`
	MaxToolCalls        int               `json:"max_tool_calls" yaml:"max_tool_calls" mapstructure:"max_tool_calls"`
	MemoryNudgeInterval int               `json:"memory_nudge_interval" yaml:"memory_nudge_interval" mapstructure:"memory_nudge_interval"`
	SkillNudgeInterval  int               `json:"skill_nudge_interval" yaml:"skill_nudge_interval" mapstructure:"skill_nudge_interval"`
	SmartRouting        bool              `json:"smart_routing" yaml:"smart_routing" mapstructure:"smart_routing"`
	PromptCaching       bool              `json:"prompt_caching" yaml:"prompt_caching" mapstructure:"prompt_caching"`
	Routing             RoutingConfig     `json:"routing,omitempty" yaml:"routing,omitempty" mapstructure:"routing"`
	Compression         CompressionConfig `json:"compression" yaml:"compression" mapstructure:"compression"`
	FallbackProviders   []string          `json:"fallback_providers,omitempty" yaml:"fallback_providers,omitempty" mapstructure:"fallback_providers"` // ordered, tried on provider-wide outages
	Budget              BudgetConfig      `json:"budget,omitempty" yaml:"budget,omitempty" mapstructure:"budget"`
}

// CompressionConfig configures context compression.
type CompressionConfig struct {
	Enabled        bool    `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Engine         string  `json:"engine" yaml:"engine" mapstructure:"engine"`                   // "compressor" (default) or a registered plugin engine
	Threshold      float64 `json:"threshold" yaml:"threshold" mapstructure:"threshold"`          // 0.0-1.0, fraction of context window
	TargetRatio    float64 `json:"target_ratio" yaml:"target_ratio" mapstructure:"target_ratio"` // target size after compression
	ProtectedTurns int     `json:"protected_turns" yaml:"protected_turns" mapstructure:"protected_turns"`
	SummaryModel   string  `json:"summary_model" yaml:"summary_model" mapstructure:"summary_model"`
}

// ProviderEntry configures a single LLM provider.
type ProviderEntry struct {
	Type    string   `json:"type" yaml:"type" mapstructure:"type"`
	APIKey  string   `json:"api_key" yaml:"api_key" mapstructure:"api_key"`
	APIKeys []string `json:"api_keys,omitempty" yaml:"api_keys,omitempty" mapstructure:"api_keys"` // optional credential pool
	BaseURL string   `json:"base_url,omitempty" yaml:"base_url,omitempty" mapstructure:"base_url"`
	Models  []string `json:"models,omitempty" yaml:"models,omitempty" mapstructure:"models"`
	OrgID   string   `json:"org_id,omitempty" yaml:"org_id,omitempty" mapstructure:"org_id"`
}

// MemoryConfig configures the memory system.
type MemoryConfig struct {
	Provider   string `json:"provider" yaml:"provider" mapstructure:"provider"`
	DBPath     string `json:"db_path" yaml:"db_path" mapstructure:"db_path"`
	MaxResults int    `json:"max_results" yaml:"max_results" mapstructure:"max_results"`
}

// GatewayConfig configures the messaging gateway.
type GatewayConfig struct {
	Platforms      map[string]PlatformConfig `json:"platforms" yaml:"platforms" mapstructure:"platforms"`
	SessionTimeout int                       `json:"session_timeout" yaml:"session_timeout" mapstructure:"session_timeout"` // minutes
	HumanDelay     bool                      `json:"human_delay" yaml:"human_delay" mapstructure:"human_delay"`
	DelayMsPerChar int                       `json:"delay_ms_per_char" yaml:"delay_ms_per_char" mapstructure:"delay_ms_per_char"` // milliseconds per character for typing delay
	AllowAll       bool                      `json:"allow_all" yaml:"allow_all" mapstructure:"allow_all"`
}

// PlatformConfig configures a single platform adapter.
type PlatformConfig struct {
	Enabled   bool              `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Token     string            `json:"token,omitempty" yaml:"token,omitempty" mapstructure:"token"`
	AllowList []string          `json:"allow_list,omitempty" yaml:"allow_list,omitempty" mapstructure:"allow_list"`
	Extra     map[string]string `json:"extra,omitempty" yaml:"extra,omitempty" mapstructure:"extra"`
}

// CLIConfig configures the CLI interface.
type CLIConfig struct {
	Skin    string `json:"skin" yaml:"skin" mapstructure:"skin"`
	Profile string `json:"profile" yaml:"profile" mapstructure:"profile"`
}

// CronConfig configures the cron scheduler.
type CronConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
}

// SecurityConfig configures security features.
type SecurityConfig struct {
	RedactPII        bool     `json:"redact_pii" yaml:"redact_pii" mapstructure:"redact_pii"`
	ProtectedPaths   []string `json:"protected_paths" yaml:"protected_paths" mapstructure:"protected_paths"`
	DangerousApprove bool     `json:"dangerous_approve" yaml:"dangerous_approve" mapstructure:"dangerous_approve"`
}

// ResolveAPIKey returns the API key for a provider by checking config first,
// then falling back to well-known environment variables.
func ResolveAPIKey(cfg *Config, providerName string) string {
	if cfg != nil {
		if entry, ok := cfg.Provider[providerName]; ok && entry.APIKey != "" {
			return entry.APIKey
		}
	}
	envMap := map[string]string{
		"openai":      "OPENAI_API_KEY",
		"anthropic":   "ANTHROPIC_API_KEY",
		"gemini":      "GEMINI_API_KEY",
		"ollama":      "", // no key needed
		"mistral":     "MISTRAL_API_KEY",
		"openrouter":  "OPENROUTER_API_KEY",
		"nous":        "NOUS_API_KEY",
		"huggingface": "HF_TOKEN",
		"glm":         "GLM_API_KEY",
		"kimi":        "MOONSHOT_API_KEY",
		"minimax":     "MINIMAX_API_KEY",
		"compatible":  "COMPATIBLE_API_KEY",
	}
	if envVar, ok := envMap[providerName]; ok && envVar != "" {
		return os.Getenv(envVar)
	}
	return ""
}

// HeraDir returns the Hera configuration directory. Delegates to
// paths.HeraHome() so $HERA_HOME is honored uniformly across the codebase.
// Default: ~/.hera. Override: $HERA_HOME.
func HeraDir() string {
	return paths.HeraHome()
}

// ExpandPath resolves a leading "~" or "~/" to the user's home directory.
// A bare "~" with no separator still maps to $HOME. Returns the path
// unchanged when no tilde prefix is present. This exists because Viper
// delivers db_path, soul_file, and similar fields as literal YAML
// strings — leaving "~" unexpanded creates a directory-named-"~" under
// the process cwd and silently splits data across two databases.
func ExpandPath(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~"+string(filepath.Separator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

// Load reads configuration from file, env vars, and defaults.
func Load() (*Config, error) {
	heraDir := HeraDir()

	// Load .env files early so ${VAR} references in config.yaml resolve
	loadDotEnv(filepath.Join(heraDir, ".env"))
	loadDotEnv(".env")

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(heraDir)

	// Defaults
	v.SetDefault("agent.default_provider", "openai")
	v.SetDefault("agent.default_model", "gpt-4o")
	v.SetDefault("agent.personality", "helpful")
	v.SetDefault("agent.soul_file", filepath.Join(heraDir, "SOUL.md"))
	v.SetDefault("agent.max_tool_calls", 20)
	v.SetDefault("agent.memory_nudge_interval", 10)
	v.SetDefault("agent.skill_nudge_interval", 15)
	v.SetDefault("agent.smart_routing", true)
	v.SetDefault("agent.compression.enabled", true)
	v.SetDefault("agent.compression.engine", "compressor")
	v.SetDefault("agent.compression.threshold", 0.5)
	v.SetDefault("agent.compression.target_ratio", 0.2)
	v.SetDefault("agent.compression.protected_turns", 5)
	v.SetDefault("memory.provider", "sqlite")
	v.SetDefault("memory.db_path", filepath.Join(heraDir, "hera.db"))
	v.SetDefault("memory.max_results", 10)
	v.SetDefault("gateway.session_timeout", 30)
	v.SetDefault("gateway.human_delay", true)
	v.SetDefault("gateway.delay_ms_per_char", 30)
	v.SetDefault("cli.skin", "default")
	v.SetDefault("cron.enabled", false)
	v.SetDefault("security.redact_pii", false)
	v.SetDefault("security.dangerous_approve", false)
	v.SetDefault("security.protected_paths", []string{
		"~/.ssh",
		"~/.gnupg",
		"~/.aws/credentials",
	})

	// Env vars: HERA_AGENT_DEFAULT_PROVIDER, etc.
	v.SetEnvPrefix("HERA")
	v.AutomaticEnv()

	// .env already loaded above via loadDotenv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Expand ${ENV_VAR} references in provider config values.
	for name, entry := range cfg.Provider {
		entry.APIKey = expandEnvRefs(entry.APIKey)
		entry.BaseURL = expandEnvRefs(entry.BaseURL)
		for i, k := range entry.APIKeys {
			entry.APIKeys[i] = expandEnvRefs(k)
		}
		cfg.Provider[name] = entry
	}

	// Expand ${ENV_VAR} references in gateway platform config values
	// (token + every value under extra). Allow lists stay literal since
	// they're user/chat IDs, not secrets.
	for name, plat := range cfg.Gateway.Platforms {
		plat.Token = expandEnvRefs(plat.Token)
		for k, v := range plat.Extra {
			plat.Extra[k] = expandEnvRefs(v)
		}
		cfg.Gateway.Platforms[name] = plat
	}

	// Resolve personality: if `agent.personality` is a short name like
	// "kawaii", look up the bundled (or user-overridden) profile and
	// inject its guidelines. Multi-line custom values pass through.
	cfg.Agent.Personality = personalities.Resolve(cfg.Agent.Personality)

	// Expand "~" in filesystem paths. Viper passes strings through
	// verbatim, and a bare "~" in db_path silently creates a literal
	// tilde directory under the process cwd — which is exactly how
	// memory writes ended up going to the wrong database.
	cfg.Memory.DBPath = ExpandPath(cfg.Memory.DBPath)
	cfg.Agent.SoulFile = ExpandPath(cfg.Agent.SoulFile)
	for i, p := range cfg.Security.ProtectedPaths {
		cfg.Security.ProtectedPaths[i] = ExpandPath(p)
	}

	// Validate structure and log any issues as warnings. Never fatal —
	// the caller can still run on a half-broken config for debugging.
	logValidationWarnings(Validate(&cfg))

	return &cfg, nil
}

// expandEnvRefs replaces ${VAR} patterns with their environment variable values.
func expandEnvRefs(s string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	return os.Expand(s, os.Getenv)
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range splitLines(string(data)) {
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		for i := 0; i < len(line); i++ {
			if line[i] == '=' {
				key := line[:i]
				val := line[i+1:]
				// Strip quotes
				if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
					val = val[1 : len(val)-1]
				}
				os.Setenv(key, val)
				break
			}
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
