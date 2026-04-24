package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDisplaySetting_GlobalDefault(t *testing.T) {
	result := ResolveDisplaySetting(nil, "unknown", "tool_progress", "fallback")
	assert.Equal(t, ToolProgressAll, result)
}

func TestResolveDisplaySetting_PlatformDefault(t *testing.T) {
	result := ResolveDisplaySetting(nil, "telegram", "tool_progress", nil)
	assert.Equal(t, ToolProgressAll, result)

	result2 := ResolveDisplaySetting(nil, "signal", "tool_progress", nil)
	assert.Equal(t, ToolProgressOff, result2)
}

func TestResolveDisplaySetting_UserGlobalOverride(t *testing.T) {
	cfg := map[string]any{
		"display": map[string]any{
			"tool_progress": "new",
		},
	}
	result := ResolveDisplaySetting(cfg, "telegram", "tool_progress", nil)
	assert.Equal(t, "new", result)
}

func TestResolveDisplaySetting_PerPlatformOverride(t *testing.T) {
	cfg := map[string]any{
		"display": map[string]any{
			"tool_progress": "all",
			"platforms": map[string]any{
				"telegram": map[string]any{
					"tool_progress": "off",
				},
			},
		},
	}
	result := ResolveDisplaySetting(cfg, "telegram", "tool_progress", nil)
	assert.Equal(t, "off", result)
}

func TestResolveDisplaySetting_Fallback(t *testing.T) {
	result := ResolveDisplaySetting(nil, "unknown", "nonexistent", "my-fallback")
	assert.Equal(t, "my-fallback", result)
}

func TestResolveDisplaySetting_NormaliseBool(t *testing.T) {
	cfg := map[string]any{
		"display": map[string]any{
			"tool_progress": false,
		},
	}
	result := ResolveDisplaySetting(cfg, "telegram", "tool_progress", nil)
	assert.Equal(t, ToolProgressOff, result)
}

func TestResolveDisplaySetting_ShowReasoning(t *testing.T) {
	cfg := map[string]any{
		"display": map[string]any{
			"show_reasoning": "true",
		},
	}
	result := ResolveDisplaySetting(cfg, "telegram", "show_reasoning", false)
	assert.Equal(t, true, result)
}

func TestResolveDisplaySetting_ToolPreviewLength(t *testing.T) {
	cfg := map[string]any{
		"display": map[string]any{
			"tool_preview_length": 100,
		},
	}
	result := ResolveDisplaySetting(cfg, "telegram", "tool_preview_length", 0)
	assert.Equal(t, 100, result)
}

func TestGetPlatformDefaults_Known(t *testing.T) {
	d := GetPlatformDefaults("telegram")
	assert.Equal(t, ToolProgressAll, d["tool_progress"])
}

func TestGetPlatformDefaults_Unknown(t *testing.T) {
	d := GetPlatformDefaults("unknown-platform")
	assert.Equal(t, ToolProgressAll, d["tool_progress"])
}

func TestGetPlatformDefaults_ReturnsCopy(t *testing.T) {
	d1 := GetPlatformDefaults("telegram")
	d1["tool_progress"] = "modified"
	d2 := GetPlatformDefaults("telegram")
	assert.Equal(t, ToolProgressAll, d2["tool_progress"])
}

func TestGetEffectiveDisplay(t *testing.T) {
	cfg := map[string]any{
		"display": map[string]any{
			"tool_progress": "new",
		},
	}
	result := GetEffectiveDisplay(cfg, "telegram")
	assert.Equal(t, "new", result["tool_progress"])
}

func TestGetEffectiveDisplay_AllKeys(t *testing.T) {
	result := GetEffectiveDisplay(nil, "telegram")
	for _, key := range OverrideableKeys {
		_, exists := result[key]
		assert.True(t, exists, "key %s should be present", key)
	}
}
