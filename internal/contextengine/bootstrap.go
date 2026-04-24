package contextengine

import (
	"fmt"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/plugins"
)

// RegisterBuiltinEngines registers the built-in "compressor" context engine
// into the given registry. Call this at startup before NewFromConfig so that
// agent.compression.engine config resolves correctly.
func RegisterBuiltinEngines(reg *plugins.Registry, summarizer agent.Summarizer) {
	reg.RegisterContextEngine(agent.NewCompressorEngine(summarizer))
}

// NewFromConfig looks up the engine named in cfg.Compression.Engine from the
// registry, initialises it with config-derived parameters, and returns it
// ready for use by the agent. Returns an error if the engine name is unknown.
func NewFromConfig(cfg config.AgentConfig, modelInfo llm.ModelMetadata, reg *plugins.Registry) (plugins.ContextEngine, error) {
	name := cfg.Compression.Engine
	if name == "" {
		name = "compressor"
	}

	eng := reg.GetContextEngine(name)
	if eng == nil {
		available := reg.ListContextEngines()
		names := make([]string, len(available))
		for i, info := range available {
			names[i] = info.Name
		}
		return nil, fmt.Errorf("unknown context engine %q (available: %v)", name, names)
	}

	if !eng.IsAvailable() {
		return nil, fmt.Errorf("context engine %q is not available (missing dependencies)", name)
	}

	threshold := cfg.Compression.Threshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.75
	}

	protectFirst := 3
	protectLast := cfg.Compression.ProtectedTurns
	if protectLast <= 0 {
		protectLast = 6
	}

	engineCfg := plugins.ContextEngineConfig{
		Name:             name,
		ContextLength:    modelInfo.ContextWindow,
		ThresholdPercent: threshold,
		ProtectFirstN:    protectFirst,
		ProtectLastN:     protectLast,
	}

	if err := eng.Initialize(engineCfg); err != nil {
		return nil, fmt.Errorf("initialize context engine %q: %w", name, err)
	}

	return eng, nil
}
