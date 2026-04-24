package plugins

import (
	"context"
	"encoding/json"

	"github.com/sadewadee/hera/internal/llm"
)

// ContextEngine manages conversation context: when to compact, how to compact,
// and optionally what tools to expose to the agent. The built-in
// "compressor" engine wraps internal/agent.Compressor. Plugins replace it via
// Registry.RegisterContextEngine and user config agent.compression.engine.
//
// Implementations that only need to override a subset of methods should embed
// BaseContextEngine, which provides safe no-op defaults for all optional
// methods.
type ContextEngine interface {
	// Identity
	Name() string
	IsAvailable() bool

	// Lifecycle
	Initialize(cfg ContextEngineConfig) error
	Shutdown()

	// Core — called every agent turn
	UpdateFromResponse(usage llm.Usage)
	ShouldCompress(promptTokens int) bool
	Compress(ctx context.Context, messages []llm.Message, currentTokens int) ([]llm.Message, error)

	// Optional overrides — BaseContextEngine provides safe defaults so plugins
	// need only embed it and override the methods they care about.
	ShouldCompressPreflight(messages []llm.Message) bool
	OnSessionStart(sessionID, platform, userID string) error
	OnSessionEnd(sessionID string, messages []llm.Message) error
	OnSessionReset()
	GetToolSchemas() []llm.ToolDef
	HandleToolCall(ctx context.Context, name string, args json.RawMessage) (string, error)
	Status() ContextEngineStatus
	UpdateModel(model string, contextLength int, baseURL, apiKey, provider string) error
}

// ContextEngineConfig is passed to Initialize.
type ContextEngineConfig struct {
	// Name is the engine identifier from the registry.
	Name string
	// ContextLength is the full context window size for the active model.
	ContextLength int
	// ThresholdPercent is the fraction of ContextLength at which compression
	// is triggered (default 0.75).
	ThresholdPercent float64
	// ProtectFirstN is the number of leading messages to exclude from
	// summarisation (preserves system prompt context; default 3).
	ProtectFirstN int
	// ProtectLastN is the number of most-recent messages to exclude from
	// summarisation (preserves conversational continuity; default 6).
	ProtectLastN int
	// Params holds engine-specific YAML key/value pairs passed through from
	// the user config. Engines must not fail on unknown keys.
	Params map[string]any
}

// ContextEngineStatus is returned by Status() for display and logging.
type ContextEngineStatus struct {
	Name                 string
	LastPromptTokens     int
	LastCompletionTokens int
	LastTotalTokens      int
	ThresholdTokens      int
	ContextLength        int
	CompressionCount     int
	UsagePercent         float64
}
