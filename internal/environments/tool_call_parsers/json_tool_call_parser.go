package tool_call_parsers

import (
	"regexp"
	"strings"
)

// JSONToolCallParser parses tool calls expressed as JSON inside <tool_call>...</tool_call> tags.
type JSONToolCallParser struct{}

func (p *JSONToolCallParser) Name() string { return "json_tool_call" }
func (p *JSONToolCallParser) CanParse(text string) bool {
	return strings.Contains(text, "<tool_call>")
}

func (p *JSONToolCallParser) Parse(text string) ([]ToolCall, error) {
	re := regexp.MustCompile(`<tool_call>([\s\S]*?)</tool_call>`)
	matches := re.FindAllStringSubmatch(text, -1)
	var calls []ToolCall
	for _, m := range matches {
		calls = append(calls, ToolCall{Raw: m[1]})
	}
	return calls, nil
}
