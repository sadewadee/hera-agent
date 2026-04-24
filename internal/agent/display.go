package agent

import (
	"fmt"
	"strings"
)

// ToolDisplay formats tool call information for terminal display.
type ToolDisplay struct {
	ShowSpinner bool
	Compact     bool
}

// FormatToolCall creates a human-readable preview of a tool invocation.
func (d *ToolDisplay) FormatToolCall(toolName string, args map[string]interface{}) string {
	var b strings.Builder
	if d.Compact {
		fmt.Fprintf(&b, "  > %s(", toolName)
		first := true
		for k, v := range args {
			if !first { b.WriteString(", ") }
			fmt.Fprintf(&b, "%s=%v", k, v)
			first = false
		}
		b.WriteString(")")
	} else {
		fmt.Fprintf(&b, "  Tool: %s\n", toolName)
		for k, v := range args {
			fmt.Fprintf(&b, "    %s: %v\n", k, v)
		}
	}
	return b.String()
}

// FormatToolResult creates a summary of a tool result.
func (d *ToolDisplay) FormatToolResult(toolName string, result string, isError bool) string {
	if isError {
		return fmt.Sprintf("  < %s: ERROR: %s", toolName, truncate(result, 200))
	}
	return fmt.Sprintf("  < %s: %s", toolName, truncate(result, 200))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen { return s }
	return s[:maxLen] + "..."
}
