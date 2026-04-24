package config

import (
	"errors"
	"fmt"
	"log/slog"
)

// Validate checks a Config for common misconfiguration. It returns an
// errors.Join of every violation so the caller can surface all problems
// at once (no whack-a-mole). Violations are non-fatal: Load() logs them
// as warnings and still returns the config so the binary can start.
//
// Covered checks:
//   - agent.default_provider and agent.default_model must be non-empty
//   - the provider named by default_provider must be configured under providers:
//   - any platform with enabled=true must have a token (except cli / apiserver,
//     which don't require one)
//   - memory.provider must be one of the supported backends
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}

	var errs []error

	if cfg.Agent.DefaultProvider == "" {
		errs = append(errs, fmt.Errorf("agent.default_provider is required"))
	}
	if cfg.Agent.DefaultModel == "" {
		errs = append(errs, fmt.Errorf("agent.default_model is required"))
	}
	if cfg.Agent.DefaultProvider != "" {
		if _, ok := cfg.Provider[cfg.Agent.DefaultProvider]; !ok {
			errs = append(errs, fmt.Errorf(
				"agent.default_provider %q has no entry under providers:",
				cfg.Agent.DefaultProvider,
			))
		}
	}

	// Platform adapters that don't need a token.
	tokenlessPlatforms := map[string]bool{
		"cli":       true,
		"apiserver": true,
		"webhook":   true,
	}
	for name, plat := range cfg.Gateway.Platforms {
		if !plat.Enabled {
			continue
		}
		if tokenlessPlatforms[name] {
			continue
		}
		if plat.Token == "" {
			errs = append(errs, fmt.Errorf(
				"gateway.platforms.%s.enabled=true but token is empty",
				name,
			))
		}
	}

	if cfg.Memory.Provider != "" {
		supportedMemory := map[string]bool{
			// Built-in (default)
			"sqlite": true,
			"memory": true,
			// Plugin providers (announced in v0.2.0 / v0.4.0, wired in v0.11.0)
			"mem0":        true,
			"hindsight":   true,
			"holographic": true,
			"honcho":      true,
			"byterover":   true,
			"openviking":  true,
			"retaindb":    true,
			"supermemory": true,
		}
		if !supportedMemory[cfg.Memory.Provider] {
			errs = append(errs, fmt.Errorf(
				"memory.provider %q not supported; valid options: "+
					"sqlite, memory, mem0, hindsight, holographic, honcho, "+
					"byterover, openviking, retaindb, supermemory",
				cfg.Memory.Provider,
			))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// logValidationWarnings splits a joined error into individual entries and
// logs each at WARN level so the user can triage without the startup
// aborting. No-op when err is nil.
func logValidationWarnings(err error) {
	if err == nil {
		return
	}
	// errors.Join produces an error whose Error() is newline-separated;
	// emit each as its own warning line.
	for _, e := range unwrapJoined(err) {
		slog.Warn("config validation", "issue", e.Error())
	}
}

// unwrapJoined returns the individual errors inside an errors.Join result.
// Falls back to a single-element slice when the error isn't joined.
func unwrapJoined(err error) []error {
	type unwrapper interface{ Unwrap() []error }
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return []error{err}
}
