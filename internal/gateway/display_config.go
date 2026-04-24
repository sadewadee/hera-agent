// Package gateway provides the multi-platform messaging gateway.
//
// display_config.go implements per-platform display/verbosity configuration
// resolution with tiered defaults based on platform capabilities.
package gateway

import (
	"strconv"
	"strings"
)

// Display setting constants.
const (
	ToolProgressAll = "all"
	ToolProgressNew = "new"
	ToolProgressOff = "off"
)

// globalDefaults are the fallback display settings.
var globalDefaults = map[string]any{
	"tool_progress":       ToolProgressAll,
	"show_reasoning":      false,
	"tool_preview_length": 0,
	"streaming":           nil,
}

// Tier presets based on platform capability.
var tierHigh = map[string]any{
	"tool_progress":       ToolProgressAll,
	"show_reasoning":      false,
	"tool_preview_length": 40,
	"streaming":           nil,
}

var tierMedium = map[string]any{
	"tool_progress":       ToolProgressNew,
	"show_reasoning":      false,
	"tool_preview_length": 40,
	"streaming":           nil,
}

var tierLow = map[string]any{
	"tool_progress":       ToolProgressOff,
	"show_reasoning":      false,
	"tool_preview_length": 40,
	"streaming":           false,
}

var tierMinimal = map[string]any{
	"tool_progress":       ToolProgressOff,
	"show_reasoning":      false,
	"tool_preview_length": 0,
	"streaming":           false,
}

// platformDefaults maps platform keys to their default display settings.
var platformDefaults = map[string]map[string]any{
	// Tier 1: full edit support
	"telegram": tierHigh,
	"discord":  tierHigh,
	// Tier 2: edit support, workspace channels
	"slack":       tierMedium,
	"mattermost":  tierMedium,
	"matrix":      tierMedium,
	"feishu":      tierMedium,
	// Tier 3: no edit support
	"signal":         tierLow,
	"whatsapp":       tierLow,
	"bluebubbles":    tierLow,
	"weixin":         tierLow,
	"wecom":          tierLow,
	"wecom_callback": tierLow,
	"dingtalk":       tierLow,
	// Tier 4: batch/non-interactive
	"email":          tierMinimal,
	"sms":            tierMinimal,
	"webhook":        tierMinimal,
	"homeassistant":  tierMinimal,
	"api_server": func() map[string]any {
		m := make(map[string]any)
		for k, v := range tierHigh {
			m[k] = v
		}
		m["tool_preview_length"] = 0
		return m
	}(),
}

// OverrideableKeys is the set of display settings that support per-platform overrides.
var OverrideableKeys = []string{
	"tool_progress",
	"show_reasoning",
	"tool_preview_length",
	"streaming",
}

// ResolveDisplaySetting resolves a display setting with per-platform override support.
//
// Resolution order (first non-nil wins):
//  1. display.platforms.<platform>.<key>
//  2. display.<key>
//  3. platformDefaults[platform][key]
//  4. globalDefaults[key]
func ResolveDisplaySetting(userConfig map[string]any, platformKey, setting string, fallback any) any {
	displayCfg, _ := userConfig["display"].(map[string]any)
	if displayCfg == nil {
		displayCfg = map[string]any{}
	}

	// 1. Explicit per-platform override.
	if platforms, ok := displayCfg["platforms"].(map[string]any); ok {
		if platOverrides, ok := platforms[platformKey].(map[string]any); ok {
			if val, exists := platOverrides[setting]; exists && val != nil {
				return normalise(setting, val)
			}
		}
	}

	// 1b. Backward compat: display.tool_progress_overrides.<platform>.
	if setting == "tool_progress" {
		if legacy, ok := displayCfg["tool_progress_overrides"].(map[string]any); ok {
			if val, exists := legacy[platformKey]; exists && val != nil {
				return normalise(setting, val)
			}
		}
	}

	// 2. Global user setting.
	if val, exists := displayCfg[setting]; exists && val != nil {
		return normalise(setting, val)
	}

	// 3. Built-in platform default.
	if platDef, ok := platformDefaults[platformKey]; ok {
		if val, exists := platDef[setting]; exists && val != nil {
			return val
		}
	}

	// 4. Built-in global default.
	if val, exists := globalDefaults[setting]; exists && val != nil {
		return val
	}

	return fallback
}

// GetPlatformDefaults returns the built-in default display settings for a platform.
func GetPlatformDefaults(platformKey string) map[string]any {
	if d, ok := platformDefaults[platformKey]; ok {
		result := make(map[string]any, len(d))
		for k, v := range d {
			result[k] = v
		}
		return result
	}
	result := make(map[string]any, len(globalDefaults))
	for k, v := range globalDefaults {
		result[k] = v
	}
	return result
}

// GetEffectiveDisplay returns fully-resolved display settings for a platform.
func GetEffectiveDisplay(userConfig map[string]any, platformKey string) map[string]any {
	result := make(map[string]any, len(OverrideableKeys))
	for _, key := range OverrideableKeys {
		result[key] = ResolveDisplaySetting(userConfig, platformKey, key, nil)
	}
	return result
}

// normalise handles YAML quirks (bare off -> false in YAML 1.1).
func normalise(setting string, value any) any {
	switch setting {
	case "tool_progress":
		switch v := value.(type) {
		case bool:
			if !v {
				return ToolProgressOff
			}
			return ToolProgressAll
		case string:
			return strings.ToLower(v)
		default:
			return ToolProgressAll
		}
	case "show_reasoning", "streaming":
		switch v := value.(type) {
		case bool:
			return v
		case string:
			lower := strings.ToLower(v)
			return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
		default:
			return false
		}
	case "tool_preview_length":
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
			return 0
		default:
			return 0
		}
	}
	return value
}
