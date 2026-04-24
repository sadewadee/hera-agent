// Package cli provides the Hera CLI application.
//
// mcp_config.go implements MCP Server Management CLI commands:
// add, remove, list, test, and configure MCP servers.
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"
)

var envVarNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// MCPServerConfig represents an MCP server entry in config.
type MCPServerConfig struct {
	URL     string            `yaml:"url,omitempty" json:"url,omitempty"`
	Command string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Auth    string            `yaml:"auth,omitempty" json:"auth,omitempty"`
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Tools   *MCPToolsFilter   `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// MCPToolsFilter controls which tools are enabled for an MCP server.
type MCPToolsFilter struct {
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// MCPTool represents a discovered MCP tool.
type MCPTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MCPPreset represents a known MCP server preset.
type MCPPreset struct {
	URL     string   `json:"url,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// MCPPresets is the registry of known presets.
var MCPPresets = map[string]*MCPPreset{}

// GetMCPServers returns the mcp_servers map from config.
func GetMCPServers(config map[string]any) map[string]map[string]any {
	servers, ok := config["mcp_servers"].(map[string]any)
	if !ok {
		return map[string]map[string]any{}
	}
	result := make(map[string]map[string]any)
	for name, v := range servers {
		if cfg, ok := v.(map[string]any); ok {
			result[name] = cfg
		}
	}
	return result
}

// EnvKeyForServer converts a server name to an env-var key.
func EnvKeyForServer(name string) string {
	upper := strings.ToUpper(name)
	upper = strings.ReplaceAll(upper, "-", "_")
	return fmt.Sprintf("MCP_%s_API_KEY", upper)
}

// ParseEnvAssignments parses KEY=VALUE strings from CLI args.
func ParseEnvAssignments(rawEnv []string) (map[string]string, error) {
	parsed := make(map[string]string)
	for _, item := range rawEnv {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		idx := strings.Index(text, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid --env value '%s' (expected KEY=VALUE)", text)
		}
		key := strings.TrimSpace(text[:idx])
		value := text[idx+1:]
		if key == "" {
			return nil, fmt.Errorf("invalid --env value '%s' (missing variable name)", text)
		}
		if !envVarNameRE.MatchString(key) {
			return nil, fmt.Errorf("invalid --env variable name '%s'", key)
		}
		parsed[key] = value
	}
	return parsed, nil
}

// ApplyMCPPreset applies a known preset when transport details are omitted.
func ApplyMCPPreset(presetName, url, command string, cmdArgs []string) (string, string, []string, bool, error) {
	if presetName == "" {
		return url, command, cmdArgs, false, nil
	}
	preset, ok := MCPPresets[presetName]
	if !ok {
		return "", "", nil, false, fmt.Errorf("unknown MCP preset: %s", presetName)
	}
	if url != "" || command != "" {
		return url, command, cmdArgs, false, nil
	}
	return preset.URL, preset.Command, preset.Args, true, nil
}

// InterpolateValue resolves ${ENV_VAR} references in a string.
func InterpolateValue(value string) string {
	return os.Expand(value, func(key string) string {
		return os.Getenv(key)
	})
}

// MCPListEntry formats a single MCP server for display.
type MCPListEntry struct {
	Name      string
	Transport string
	ToolsStr  string
	Enabled   bool
}

// FormatMCPList formats the MCP server list for display.
func FormatMCPList(servers map[string]map[string]any) []MCPListEntry {
	var entries []MCPListEntry
	for name, cfg := range servers {
		entry := MCPListEntry{Name: name}

		// Transport.
		if url, ok := cfg["url"].(string); ok {
			if len(url) > 28 {
				url = url[:25] + "..."
			}
			entry.Transport = url
		} else if cmd, ok := cfg["command"].(string); ok {
			transport := cmd
			if args, ok := cfg["args"].([]any); ok && len(args) > 0 {
				parts := []string{cmd}
				for _, a := range args {
					if len(parts) >= 3 {
						break
					}
					parts = append(parts, fmt.Sprintf("%v", a))
				}
				transport = strings.Join(parts, " ")
			}
			if len(transport) > 28 {
				transport = transport[:25] + "..."
			}
			entry.Transport = transport
		} else {
			entry.Transport = "?"
		}

		// Tools.
		if toolsCfg, ok := cfg["tools"].(map[string]any); ok {
			if inc, ok := toolsCfg["include"].([]any); ok {
				entry.ToolsStr = fmt.Sprintf("%d selected", len(inc))
			} else if exc, ok := toolsCfg["exclude"].([]any); ok {
				entry.ToolsStr = fmt.Sprintf("-%d excluded", len(exc))
			} else {
				entry.ToolsStr = "all"
			}
		} else {
			entry.ToolsStr = "all"
		}

		// Enabled.
		entry.Enabled = true
		if enabled, ok := cfg["enabled"]; ok {
			switch v := enabled.(type) {
			case bool:
				entry.Enabled = v
			case string:
				lower := strings.ToLower(v)
				entry.Enabled = lower == "true" || lower == "1" || lower == "yes"
			}
		}

		entries = append(entries, entry)
	}
	return entries
}

// MCPCommand dispatches MCP subcommands.
func MCPCommand(action string) {
	slog.Debug("mcp command", "action", action)

	switch action {
	case "list", "ls", "":
		fmt.Println("MCP Servers: (use 'hera mcp add' to configure)")
	case "add":
		fmt.Println("Usage: hera mcp add <name> --url <endpoint>")
	case "remove", "rm":
		fmt.Println("Usage: hera mcp remove <name>")
	case "test":
		fmt.Println("Usage: hera mcp test <name>")
	case "configure", "config":
		fmt.Println("Usage: hera mcp configure <name>")
	default:
		fmt.Printf("Unknown MCP command: %s\n", action)
	}
}

// MCPTestTimeout is the default timeout for MCP server probing.
const MCPTestTimeout = 30 * time.Second
