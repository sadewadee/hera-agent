package plugins

// MemoryProvider is the interface that external memory plugins must implement.
//
// Only one memory provider can be active at a time, selected via the
// memory.provider key in config.yaml. The provider is initialized once
// at agent startup and shut down when the agent exits.
type MemoryProvider interface {
	// Name returns the unique identifier for this provider (e.g. "honcho", "mem0").
	Name() string

	// IsAvailable reports whether the provider's dependencies are satisfied.
	// Called during discovery without initializing the provider.
	IsAvailable() bool

	// Initialize prepares the provider for use with the given session.
	Initialize(sessionID string) error

	// SystemPromptBlock returns text to inject into the system prompt,
	// describing the memory capabilities available. Return "" to skip.
	SystemPromptBlock() string

	// Prefetch runs a synchronous query before the agent's first LLM call
	// for a turn, returning context to prepend. Return "" if nothing relevant.
	Prefetch(query string, sessionID string) string

	// SyncTurn records a completed conversation turn (user + assistant messages)
	// into the memory backend. Implementations should be non-blocking where possible.
	SyncTurn(userContent, assistantContent, sessionID string)

	// OnMemoryWrite is called when the built-in memory system writes data,
	// allowing the plugin to mirror or index the write.
	OnMemoryWrite(action, target, content string)

	// OnPreCompress is called before context compression discards messages,
	// allowing the plugin to extract and persist insights. Return any text
	// to inject before compression, or "".
	OnPreCompress(messages []map[string]interface{}) string

	// OnSessionEnd is called when a session ends, giving the plugin a chance
	// to flush pending writes, commit transactions, etc.
	OnSessionEnd(messages []map[string]interface{})

	// GetToolSchemas returns the tool definitions this provider exposes to the LLM.
	GetToolSchemas() []ToolSchema

	// HandleToolCall processes a tool invocation from the LLM.
	// Returns the tool result as a JSON string.
	HandleToolCall(toolName string, args map[string]interface{}) (string, error)

	// GetConfigSchema returns configuration fields the user can set.
	GetConfigSchema() []ConfigField

	// Shutdown performs graceful cleanup (flush queues, close connections).
	Shutdown()
}

// ConfigField describes a user-configurable option for a plugin.
type ConfigField struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Secret      bool   `json:"secret"`
	EnvVar      string `json:"env_var"`
	URL         string `json:"url,omitempty"`
}
