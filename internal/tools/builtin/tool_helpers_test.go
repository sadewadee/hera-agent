package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeBrowserCloudProvider_Empty(t *testing.T) {
	assert.Equal(t, "local", NormalizeBrowserCloudProvider(""))
}

func TestNormalizeBrowserCloudProvider_WithWhitespace(t *testing.T) {
	assert.Equal(t, "browserbase", NormalizeBrowserCloudProvider("  BrowserBase  "))
}

func TestNormalizeBrowserCloudProvider_Lowercase(t *testing.T) {
	assert.Equal(t, "chromium", NormalizeBrowserCloudProvider("CHROMIUM"))
}

func TestNormalizeBrowserCloudProvider_AlreadyNormalized(t *testing.T) {
	assert.Equal(t, "local", NormalizeBrowserCloudProvider("local"))
}

func TestCoerceModalMode_ValidModes(t *testing.T) {
	for _, mode := range []string{"auto", "direct", "managed"} {
		assert.Equal(t, mode, CoerceModalMode(mode))
	}
}

func TestCoerceModalMode_Invalid(t *testing.T) {
	assert.Equal(t, "auto", CoerceModalMode("unknown"))
	assert.Equal(t, "auto", CoerceModalMode(""))
	assert.Equal(t, "auto", CoerceModalMode("cloud"))
}

func TestCoerceModalMode_CaseInsensitive(t *testing.T) {
	assert.Equal(t, "auto", CoerceModalMode("AUTO"))
	assert.Equal(t, "direct", CoerceModalMode("DIRECT"))
	assert.Equal(t, "managed", CoerceModalMode("Managed"))
}

func TestNormalizeModalMode_DelegatesToCoerce(t *testing.T) {
	assert.Equal(t, CoerceModalMode("auto"), NormalizeModalMode("auto"))
	assert.Equal(t, CoerceModalMode("bad"), NormalizeModalMode("bad"))
}

func TestResolveModalBackendState_AutoNoDirect(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "")
	state := ResolveModalBackendState("auto", false, false)
	assert.Equal(t, "auto", state.Mode)
	assert.Empty(t, state.SelectedBackend)
	assert.False(t, state.HasDirect)
	assert.False(t, state.ManagedReady)
}

func TestResolveModalBackendState_AutoWithDirect(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "")
	state := ResolveModalBackendState("auto", true, false)
	assert.Equal(t, "direct", state.SelectedBackend)
	assert.True(t, state.HasDirect)
}

func TestResolveModalBackendState_DirectMode(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "")
	state := ResolveModalBackendState("direct", true, false)
	assert.Equal(t, "direct", state.Mode)
	assert.Equal(t, "direct", state.SelectedBackend)
}

func TestResolveModalBackendState_DirectModeNoDirect(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "")
	state := ResolveModalBackendState("direct", false, false)
	assert.Equal(t, "direct", state.Mode)
	assert.Empty(t, state.SelectedBackend)
}

func TestResolveModalBackendState_ManagedNotEnabled(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "")
	state := ResolveModalBackendState("managed", true, true)
	assert.True(t, state.ManagedBlocked)
	assert.Empty(t, state.SelectedBackend)
}

func TestResolveModalBackendState_ManagedEnabled(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "true")
	state := ResolveModalBackendState("managed", false, true)
	assert.False(t, state.ManagedBlocked)
	assert.Equal(t, "managed", state.SelectedBackend)
}

func TestResolveModalBackendState_InvalidModeDefaultsAuto(t *testing.T) {
	t.Setenv("HERA_ENABLE_NOUS_MANAGED_TOOLS", "")
	state := ResolveModalBackendState("invalid", false, false)
	assert.Equal(t, "auto", state.Mode)
}

func TestResolveOpenAIAudioAPIKey_PreferVoice(t *testing.T) {
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "voice-key")
	t.Setenv("OPENAI_API_KEY", "openai-key")
	assert.Equal(t, "voice-key", ResolveOpenAIAudioAPIKey())
}

func TestResolveOpenAIAudioAPIKey_FallbackOpenAI(t *testing.T) {
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "openai-key")
	assert.Equal(t, "openai-key", ResolveOpenAIAudioAPIKey())
}

func TestResolveOpenAIAudioAPIKey_BothEmpty(t *testing.T) {
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	assert.Equal(t, "", ResolveOpenAIAudioAPIKey())
}

func TestEnvVarEnabled(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"TRUE", true},
		{"YES", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("TEST_ENV_VAR", tt.value)
			assert.Equal(t, tt.want, envVarEnabled("TEST_ENV_VAR"))
		})
	}
}
