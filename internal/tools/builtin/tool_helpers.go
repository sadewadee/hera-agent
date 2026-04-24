// Package builtin provides built-in tool implementations.
//
// tool_helpers.go provides shared helpers for tool backend selection,
// including browser provider normalisation, Modal execution mode
// resolution, and OpenAI audio API key fallback.
package builtin

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultBrowserProvider = "local"
	defaultModalMode       = "auto"
)

var validModalModes = map[string]bool{
	"auto":    true,
	"direct":  true,
	"managed": true,
}

// ManagedNousToolsEnabled returns true when the hidden Nous-managed
// tools feature flag is enabled via environment variable.
func ManagedNousToolsEnabled() bool {
	return envVarEnabled("HERA_ENABLE_NOUS_MANAGED_TOOLS")
}

// NormalizeBrowserCloudProvider returns a normalised browser provider key.
func NormalizeBrowserCloudProvider(value string) string {
	provider := strings.TrimSpace(strings.ToLower(value))
	if provider == "" {
		return defaultBrowserProvider
	}
	return provider
}

// CoerceModalMode returns the requested modal mode when valid, else the default.
func CoerceModalMode(value string) string {
	mode := strings.TrimSpace(strings.ToLower(value))
	if validModalModes[mode] {
		return mode
	}
	return defaultModalMode
}

// NormalizeModalMode returns a normalised modal execution mode.
func NormalizeModalMode(value string) string {
	return CoerceModalMode(value)
}

// HasDirectModalCredentials returns true when direct Modal
// credentials or config are available.
func HasDirectModalCredentials() bool {
	if os.Getenv("MODAL_TOKEN_ID") != "" && os.Getenv("MODAL_TOKEN_SECRET") != "" {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".modal.toml"))
	return err == nil
}

// ModalBackendState holds the resolved Modal backend selection state.
type ModalBackendState struct {
	RequestedMode   string `json:"requested_mode"`
	Mode            string `json:"mode"`
	HasDirect       bool   `json:"has_direct"`
	ManagedReady    bool   `json:"managed_ready"`
	ManagedBlocked  bool   `json:"managed_mode_blocked"`
	SelectedBackend string `json:"selected_backend,omitempty"`
}

// ResolveModalBackendState resolves direct vs managed Modal backend selection.
//
// Semantics:
//   - "direct"  means direct-only
//   - "managed" means managed-only
//   - "auto"    prefers managed when available, then falls back to direct
func ResolveModalBackendState(modalMode string, hasDirect, managedReady bool) ModalBackendState {
	requestedMode := CoerceModalMode(modalMode)
	normalizedMode := NormalizeModalMode(modalMode)
	managedBlocked := requestedMode == "managed" && !ManagedNousToolsEnabled()

	var selectedBackend string
	switch normalizedMode {
	case "managed":
		if ManagedNousToolsEnabled() && managedReady {
			selectedBackend = "managed"
		}
	case "direct":
		if hasDirect {
			selectedBackend = "direct"
		}
	default: // auto
		if ManagedNousToolsEnabled() && managedReady {
			selectedBackend = "managed"
		} else if hasDirect {
			selectedBackend = "direct"
		}
	}

	return ModalBackendState{
		RequestedMode:   requestedMode,
		Mode:            normalizedMode,
		HasDirect:       hasDirect,
		ManagedReady:    managedReady,
		ManagedBlocked:  managedBlocked,
		SelectedBackend: selectedBackend,
	}
}

// ResolveOpenAIAudioAPIKey returns the OpenAI API key for voice tools,
// preferring VOICE_TOOLS_OPENAI_KEY over OPENAI_API_KEY.
func ResolveOpenAIAudioAPIKey() string {
	key := strings.TrimSpace(os.Getenv("VOICE_TOOLS_OPENAI_KEY"))
	if key != "" {
		return key
	}
	return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
}

// envVarEnabled checks if an env var is set to a truthy value.
func envVarEnabled(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}
