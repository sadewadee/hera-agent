// Package acp provides the Agent Client Protocol implementation.
//
// tools.go implements ACP tool-call helpers for mapping Hera tools to ACP
// ToolKind values and building human-readable content for tool call events.
package acp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ToolKind represents the type of operation a tool performs.
type ToolKind string

const (
	ToolKindRead    ToolKind = "read"
	ToolKindEdit    ToolKind = "edit"
	ToolKindSearch  ToolKind = "search"
	ToolKindExecute ToolKind = "execute"
	ToolKindFetch   ToolKind = "fetch"
	ToolKindThink   ToolKind = "think"
	ToolKindOther   ToolKind = "other"
)

// toolKindMap maps Hera tool names to ACP ToolKind values.
var toolKindMap = map[string]ToolKind{
	// File operations
	"read_file":    ToolKindRead,
	"write_file":   ToolKindEdit,
	"patch":        ToolKindEdit,
	"search_files": ToolKindSearch,
	// Terminal / execution
	"terminal":     ToolKindExecute,
	"process":      ToolKindExecute,
	"execute_code": ToolKindExecute,
	// Web / fetch
	"web_search":  ToolKindFetch,
	"web_extract": ToolKindFetch,
	// Browser
	"browser_navigate":   ToolKindFetch,
	"browser_click":      ToolKindExecute,
	"browser_type":       ToolKindExecute,
	"browser_snapshot":   ToolKindRead,
	"browser_vision":     ToolKindRead,
	"browser_scroll":     ToolKindExecute,
	"browser_press":      ToolKindExecute,
	"browser_back":       ToolKindExecute,
	"browser_get_images": ToolKindRead,
	// Agent internals
	"delegate_task":  ToolKindExecute,
	"vision_analyze": ToolKindRead,
	"image_generate": ToolKindExecute,
	"text_to_speech": ToolKindExecute,
	// Thinking / meta
	"_thinking": ToolKindThink,
}

// GetToolKind returns the ACP ToolKind for a Hera tool, defaulting to "other".
func GetToolKind(toolName string) ToolKind {
	if kind, ok := toolKindMap[toolName]; ok {
		return kind
	}
	return ToolKindOther
}

// MakeToolCallID generates a unique tool call ID.
func MakeToolCallID() string {
	return generateToolCallID()
}

// ToolCallLocation represents a file-system location referenced by a tool call.
type ToolCallLocation struct {
	Path string `json:"path"`
	Line *int   `json:"line,omitempty"`
}

// BuildToolTitle builds a human-readable title for a tool call.
func BuildToolTitle(toolName string, args map[string]any) string {
	switch toolName {
	case "terminal":
		cmd, _ := args["command"].(string)
		if len(cmd) > 80 {
			cmd = cmd[:77] + "..."
		}
		return fmt.Sprintf("terminal: %s", cmd)
	case "read_file":
		return fmt.Sprintf("read: %s", getArgStr(args, "path", "?"))
	case "write_file":
		return fmt.Sprintf("write: %s", getArgStr(args, "path", "?"))
	case "patch":
		mode := getArgStr(args, "mode", "replace")
		path := getArgStr(args, "path", "?")
		return fmt.Sprintf("patch (%s): %s", mode, path)
	case "search_files":
		return fmt.Sprintf("search: %s", getArgStr(args, "pattern", "?"))
	case "web_search":
		return fmt.Sprintf("web search: %s", getArgStr(args, "query", "?"))
	case "web_extract":
		if urls, ok := args["urls"].([]any); ok && len(urls) > 0 {
			first, _ := urls[0].(string)
			title := fmt.Sprintf("extract: %s", first)
			if len(urls) > 1 {
				title += fmt.Sprintf(" (+%d)", len(urls)-1)
			}
			return title
		}
		return "web extract"
	case "delegate_task":
		goal := getArgStr(args, "goal", "")
		if len(goal) > 60 {
			goal = goal[:57] + "..."
		}
		if goal != "" {
			return fmt.Sprintf("delegate: %s", goal)
		}
		return "delegate task"
	case "execute_code":
		return "execute code"
	case "vision_analyze":
		q := getArgStr(args, "question", "?")
		if len(q) > 50 {
			q = q[:50]
		}
		return fmt.Sprintf("analyze image: %s", q)
	default:
		return toolName
	}
}

// BuildToolStartContent builds the content text for a tool start event.
func BuildToolStartContent(toolName string, args map[string]any) string {
	switch toolName {
	case "terminal":
		return fmt.Sprintf("$ %s", getArgStr(args, "command", ""))
	case "read_file":
		return fmt.Sprintf("Reading %s", getArgStr(args, "path", ""))
	case "search_files":
		pattern := getArgStr(args, "pattern", "")
		target := getArgStr(args, "target", "content")
		return fmt.Sprintf("Searching for '%s' (%s)", pattern, target)
	case "patch":
		mode := getArgStr(args, "mode", "replace")
		if mode == "replace" {
			return fmt.Sprintf("Patching %s", getArgStr(args, "path", ""))
		}
		return getArgStr(args, "patch", "")
	case "write_file":
		return fmt.Sprintf("Writing %s", getArgStr(args, "path", ""))
	default:
		data, err := json.MarshalIndent(args, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", args)
		}
		return string(data)
	}
}

// BuildToolCompleteContent builds the display text for a completed tool call.
// Truncates very large results for the UI.
func BuildToolCompleteContent(result string) string {
	if len(result) > 5000 {
		return result[:4900] + fmt.Sprintf("\n... (%d chars total, truncated)", len(result))
	}
	return result
}

// ExtractLocations extracts file-system locations from tool arguments.
func ExtractLocations(args map[string]any) []ToolCallLocation {
	var locations []ToolCallLocation
	path, _ := args["path"].(string)
	if path != "" {
		loc := ToolCallLocation{Path: path}
		if line, ok := getArgInt(args, "offset"); ok {
			loc.Line = &line
		} else if line, ok := getArgInt(args, "line"); ok {
			loc.Line = &line
		}
		locations = append(locations, loc)
	}
	return locations
}

func getArgStr(args map[string]any, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getArgInt(args map[string]any, key string) (int, bool) {
	switch v := args[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return int(n), true
		}
	}
	return 0, false
}

func init() {
	_ = strings.TrimSpace // avoid unused import
}
