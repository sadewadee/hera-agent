package llm

// KnownModels is the comprehensive model metadata database containing context
// windows, pricing, and capabilities for all known models across all providers.
var KnownModels = map[string]ModelMetadata{
	// --- OpenAI ---
	"gpt-4o": {
		ID: "gpt-4o", Provider: "openai",
		ContextWindow: 128000, MaxOutput: 16384,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.0025, CostPer1kOut: 0.01,
	},
	"gpt-4o-mini": {
		ID: "gpt-4o-mini", Provider: "openai",
		ContextWindow: 128000, MaxOutput: 16384,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.00015, CostPer1kOut: 0.0006,
	},
	"gpt-4-turbo": {
		ID: "gpt-4-turbo", Provider: "openai",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.01, CostPer1kOut: 0.03,
	},
	"gpt-4": {
		ID: "gpt-4", Provider: "openai",
		ContextWindow: 8192, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.03, CostPer1kOut: 0.06,
	},
	"gpt-3.5-turbo": {
		ID: "gpt-3.5-turbo", Provider: "openai",
		ContextWindow: 16385, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0005, CostPer1kOut: 0.0015,
	},
	"o1": {
		ID: "o1", Provider: "openai",
		ContextWindow: 200000, MaxOutput: 100000,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.015, CostPer1kOut: 0.06,
	},
	"o1-mini": {
		ID: "o1-mini", Provider: "openai",
		ContextWindow: 128000, MaxOutput: 65536,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.003, CostPer1kOut: 0.012,
	},
	"o3-mini": {
		ID: "o3-mini", Provider: "openai",
		ContextWindow: 200000, MaxOutput: 100000,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0011, CostPer1kOut: 0.0044,
	},

	// --- Anthropic ---
	"claude-opus-4-20250514": {
		ID: "claude-opus-4-20250514", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 32000,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.015, CostPer1kOut: 0.075,
	},
	"claude-sonnet-4-20250514": {
		ID: "claude-sonnet-4-20250514", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 64000,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.003, CostPer1kOut: 0.015,
	},
	"claude-3-5-sonnet-20241022": {
		ID: "claude-3-5-sonnet-20241022", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 8192,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.003, CostPer1kOut: 0.015,
	},
	"claude-3-5-haiku-20241022": {
		ID: "claude-3-5-haiku-20241022", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 8192,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.0008, CostPer1kOut: 0.004,
	},
	"claude-3-opus-20240229": {
		ID: "claude-3-opus-20240229", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.015, CostPer1kOut: 0.075,
	},
	"claude-3-haiku-20240307": {
		ID: "claude-3-haiku-20240307", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.00025, CostPer1kOut: 0.00125,
	},

	// --- Google Gemini ---
	"gemini-2.0-flash": {
		ID: "gemini-2.0-flash", Provider: "gemini",
		ContextWindow: 1048576, MaxOutput: 8192,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.0001, CostPer1kOut: 0.0004,
	},
	"gemini-1.5-pro": {
		ID: "gemini-1.5-pro", Provider: "gemini",
		ContextWindow: 2097152, MaxOutput: 8192,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.00125, CostPer1kOut: 0.005,
	},
	"gemini-1.5-flash": {
		ID: "gemini-1.5-flash", Provider: "gemini",
		ContextWindow: 1048576, MaxOutput: 8192,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.000075, CostPer1kOut: 0.0003,
	},

	// --- Mistral ---
	"mistral-large-latest": {
		ID: "mistral-large-latest", Provider: "mistral",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.002, CostPer1kOut: 0.006,
	},
	"mistral-small-latest": {
		ID: "mistral-small-latest", Provider: "mistral",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0002, CostPer1kOut: 0.0006,
	},
	"codestral-latest": {
		ID: "codestral-latest", Provider: "mistral",
		ContextWindow: 32768, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0003, CostPer1kOut: 0.0009,
	},
	"open-mistral-nemo": {
		ID: "open-mistral-nemo", Provider: "mistral",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.00015, CostPer1kOut: 0.00015,
	},

	// --- Nous ---
	"hermes-3-llama-3.1-405b": {
		ID: "hermes-3-llama-3.1-405b", Provider: "nous",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.005, CostPer1kOut: 0.015,
	},
	"hermes-3-llama-3.1-70b": {
		ID: "hermes-3-llama-3.1-70b", Provider: "nous",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.001, CostPer1kOut: 0.003,
	},
	"deephermes-3-llama-3-8b": {
		ID: "deephermes-3-llama-3-8b", Provider: "nous",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0002, CostPer1kOut: 0.0006,
	},

	// --- OpenRouter (pass-through, prices vary) ---
	"openrouter/auto": {
		ID: "openrouter/auto", Provider: "openrouter",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},

	// --- HuggingFace ---
	"meta-llama/Llama-3.1-70B-Instruct": {
		ID: "meta-llama/Llama-3.1-70B-Instruct", Provider: "huggingface",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: false, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},

	// --- GLM ---
	"glm-4-plus": {
		ID: "glm-4-plus", Provider: "glm",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0007, CostPer1kOut: 0.0007,
	},
	"glm-4v-plus": {
		ID: "glm-4v-plus", Provider: "glm",
		ContextWindow: 8192, MaxOutput: 1024,
		SupportsTools: false, SupportsVision: true,
		CostPer1kIn: 0.001, CostPer1kOut: 0.001,
	},

	// --- Kimi (Moonshot) ---
	"moonshot-v1-128k": {
		ID: "moonshot-v1-128k", Provider: "kimi",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0008, CostPer1kOut: 0.0008,
	},
	"moonshot-v1-32k": {
		ID: "moonshot-v1-32k", Provider: "kimi",
		ContextWindow: 32000, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0003, CostPer1kOut: 0.0003,
	},

	// --- MiniMax ---
	"abab6.5s-chat": {
		ID: "abab6.5s-chat", Provider: "minimax",
		ContextWindow: 245760, MaxOutput: 8192,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0001, CostPer1kOut: 0.0001,
	},

	// --- Ollama (local, no cost) ---
	"llama3.1": {
		ID: "llama3.1", Provider: "ollama",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},
	"llama3.2": {
		ID: "llama3.2", Provider: "ollama",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: true,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},
	"mistral": {
		ID: "mistral", Provider: "ollama",
		ContextWindow: 32768, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},
	"codellama": {
		ID: "codellama", Provider: "ollama",
		ContextWindow: 16384, MaxOutput: 4096,
		SupportsTools: false, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},
	"phi3": {
		ID: "phi3", Provider: "ollama",
		ContextWindow: 128000, MaxOutput: 4096,
		SupportsTools: false, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},
	"qwen2.5": {
		ID: "qwen2.5", Provider: "ollama",
		ContextWindow: 131072, MaxOutput: 4096,
		SupportsTools: true, SupportsVision: false,
		CostPer1kIn: 0.0, CostPer1kOut: 0.0,
	},
}

// LookupModel returns the metadata for a model by ID. Returns the zero value
// and false if the model is not in the database.
func LookupModel(modelID string) (ModelMetadata, bool) {
	m, ok := KnownModels[modelID]
	return m, ok
}

// ModelsByProvider returns all models for a given provider name.
func ModelsByProvider(provider string) []ModelMetadata {
	var result []ModelMetadata
	for _, m := range KnownModels {
		if m.Provider == provider {
			result = append(result, m)
		}
	}
	return result
}

// AllProviders returns a deduplicated list of all providers in the database.
func AllProviders() []string {
	seen := make(map[string]bool)
	var result []string
	for _, m := range KnownModels {
		if !seen[m.Provider] {
			seen[m.Provider] = true
			result = append(result, m.Provider)
		}
	}
	return result
}
