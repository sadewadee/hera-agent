// Package cli provides the Hera CLI application.
//
// skills_config.go implements skills configuration commands. Toggle
// individual skills or categories on/off, globally or per-platform.
package cli

import (
	"fmt"
	"sort"
	"strings"
)

// GetDisabledSkills returns the set of disabled skill names.
// Platform-specific list falls back to global.
func GetDisabledSkills(config map[string]any, platform string) map[string]bool {
	skillsCfg, _ := config["skills"].(map[string]any)
	if skillsCfg == nil {
		return map[string]bool{}
	}

	globalDisabled := toStringSet(skillsCfg["disabled"])

	if platform == "" {
		return globalDisabled
	}

	platDisabled, _ := skillsCfg["platform_disabled"].(map[string]any)
	if platDisabled == nil {
		return globalDisabled
	}

	platList, ok := platDisabled[platform]
	if !ok {
		return globalDisabled
	}

	return toStringSet(platList)
}

// SaveDisabledSkills persists disabled skill names to config.
func SaveDisabledSkills(config map[string]any, disabled map[string]bool, platform string) {
	if config["skills"] == nil {
		config["skills"] = map[string]any{}
	}
	skillsCfg := config["skills"].(map[string]any)

	sorted := sortedKeys(disabled)

	if platform == "" {
		skillsCfg["disabled"] = sorted
	} else {
		if skillsCfg["platform_disabled"] == nil {
			skillsCfg["platform_disabled"] = map[string]any{}
		}
		platDisabled := skillsCfg["platform_disabled"].(map[string]any)
		platDisabled[platform] = sorted
	}
}

// SkillInfo represents a skill for the configuration UI.
type SkillInfo struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

// GetSkillCategories returns sorted unique categories from a skill list.
func GetSkillCategories(skills []SkillInfo) []string {
	cats := make(map[string]bool)
	for _, s := range skills {
		cat := s.Category
		if cat == "" {
			cat = "uncategorized"
		}
		cats[cat] = true
	}
	return sortedKeys(cats)
}

// SkillsCommand is the entry point for 'hera skills'.
func SkillsCommand() {
	fmt.Println("Skills configuration:")
	fmt.Println("  Use 'hera skills' to toggle individual skills or categories.")
	fmt.Println("  Config stored in ~/.hera/config.yaml under skills:")
}

func toStringSet(v any) map[string]bool {
	result := make(map[string]bool)
	switch list := v.(type) {
	case []any:
		for _, item := range list {
			if s, ok := item.(string); ok {
				result[s] = true
			}
		}
	case []string:
		for _, s := range list {
			result[s] = true
		}
	}
	return result
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// PlatformLabels provides display labels for platforms.
var PlatformLabels = map[string]string{
	"telegram":    "Telegram",
	"discord":     "Discord",
	"slack":       "Slack",
	"mattermost":  "Mattermost",
	"matrix":      "Matrix",
	"feishu":      "Feishu",
	"signal":      "Signal",
	"whatsapp":    "WhatsApp",
	"bluebubbles": "BlueBubbles",
	"weixin":      "Weixin",
	"wecom":       "WeCom",
	"dingtalk":    "DingTalk",
	"email":       "Email",
	"sms":         "SMS",
	"webhook":     "Webhook",
	"cli":         "CLI",
}

// GetPlatformLabel returns the display label for a platform key.
func GetPlatformLabel(key string) string {
	if label, ok := PlatformLabels[key]; ok {
		return label
	}
	return strings.Title(key)
}
