// Package cli provides the Hera CLI application.
//
// tools_config.go implements tools configuration commands. Manages which
// built-in tools are enabled/disabled and configures tool-specific settings.
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
)

// ToolConfig holds per-tool configuration.
type ToolConfig struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

// GetDisabledTools returns the set of disabled tool names from config.
func GetDisabledTools(config map[string]any) map[string]bool {
	toolsCfg, _ := config["tools"].(map[string]any)
	if toolsCfg == nil {
		return map[string]bool{}
	}
	return toStringSet(toolsCfg["disabled"])
}

// SaveDisabledTools persists disabled tool names to config.
func SaveDisabledTools(config map[string]any, disabled map[string]bool) {
	if config["tools"] == nil {
		config["tools"] = map[string]any{}
	}
	toolsCfg := config["tools"].(map[string]any)
	toolsCfg["disabled"] = sortedKeys(disabled)
}

// GetToolAliases returns tool name aliases from config.
func GetToolAliases(config map[string]any) map[string]string {
	toolsCfg, _ := config["tools"].(map[string]any)
	if toolsCfg == nil {
		return map[string]string{}
	}
	aliases, _ := toolsCfg["aliases"].(map[string]any)
	result := make(map[string]string)
	for k, v := range aliases {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

// ToolApprovalConfig holds approval settings for tools.
type ToolApprovalConfig struct {
	AutoApprove  []string `json:"auto_approve,omitempty"`
	AlwaysDeny   []string `json:"always_deny,omitempty"`
	RequireOnce  []string `json:"require_once,omitempty"`
	SessionAllow []string `json:"session_allow,omitempty"`
}

// GetToolApprovalConfig returns the tool approval configuration.
func GetToolApprovalConfig(config map[string]any) *ToolApprovalConfig {
	toolsCfg, _ := config["tools"].(map[string]any)
	if toolsCfg == nil {
		return &ToolApprovalConfig{}
	}
	approvals, _ := toolsCfg["approvals"].(map[string]any)
	if approvals == nil {
		return &ToolApprovalConfig{}
	}

	return &ToolApprovalConfig{
		AutoApprove:  toStringSlice(approvals["auto_approve"]),
		AlwaysDeny:   toStringSlice(approvals["always_deny"]),
		RequireOnce:  toStringSlice(approvals["require_once"]),
		SessionAllow: toStringSlice(approvals["session_allow"]),
	}
}

// ToolsDirPaths returns the directories to search for tool implementations.
func ToolsDirPaths() []string {
	heraHome := paths.HeraHome()
	return []string{
		filepath.Join(heraHome, "tools"),
		filepath.Join(heraHome, "plugins", "tools"),
	}
}

// ListInstalledTools returns all installed tool plugin files.
func ListInstalledTools() []string {
	var tools []string
	for _, dir := range ToolsDirPaths() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			tools = append(tools, e.Name())
		}
	}
	sort.Strings(tools)
	return tools
}

// ToolsCommand is the entry point for 'hera tools'.
func ToolsCommand(action string) {
	slog.Debug("tools command", "action", action)

	switch action {
	case "list", "ls", "":
		fmt.Println("\nInstalled Tools:")
		tools := ListInstalledTools()
		if len(tools) == 0 {
			fmt.Println("  (no custom tools installed)")
		}
		for _, t := range tools {
			fmt.Printf("  %s\n", t)
		}
		fmt.Println()

	case "enable":
		fmt.Println("Usage: hera tools enable <name>")
	case "disable":
		fmt.Println("Usage: hera tools disable <name>")
	case "install":
		fmt.Println("Usage: hera tools install <source>")
	default:
		fmt.Printf("Unknown tools command: %s\n", action)
	}
}

// ToolCategoryOrder defines the display order for tool categories.
var ToolCategoryOrder = []string{
	"system", "files", "code", "web", "communication",
	"media", "data", "security", "utility",
}

func toStringSlice(v any) []string {
	switch list := v.(type) {
	case []any:
		result := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return list
	}
	return nil
}

// IsToolDisabled checks if a specific tool is disabled.
func IsToolDisabled(config map[string]any, toolName string) bool {
	disabled := GetDisabledTools(config)
	return disabled[strings.ToLower(toolName)]
}
