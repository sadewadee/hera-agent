// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// deepseek_v3_1_parser.go implements the DeepSeek V3.1 tool call format.
// Similar to V3 but with a different layout:
//
//	<|tool_call_begin|>function_name<|tool_sep|>arguments<|tool_call_end|>
//
// V3 has type+name before the separator; V3.1 has name before and args after.
package tool_call_parsers

import (
	"regexp"
	"strings"
)

var deepseekV31Pattern = regexp.MustCompile(
	`<\x{ff5c}tool\x{2581}call\x{2581}begin\x{ff5c}>(?P<function_name>.*?)<\x{ff5c}tool\x{2581}sep\x{ff5c}>(?P<function_arguments>.*?)<\x{ff5c}tool\x{2581}call\x{2581}end\x{ff5c}>`,
)

// DeepSeekV31Parser parses DeepSeek V3.1 tool calls.
type DeepSeekV31Parser struct{}

func (p *DeepSeekV31Parser) Name() string { return "deepseek_v3_1" }

func (p *DeepSeekV31Parser) CanParse(text string) bool {
	return strings.Contains(text, "tool\u2581calls\u2581begin")
}

func (p *DeepSeekV31Parser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	matches := deepseekV31Pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var calls []ToolCall
	for _, match := range matches {
		funcName := ""
		funcArgs := ""
		for i, name := range deepseekV31Pattern.SubexpNames() {
			switch name {
			case "function_name":
				funcName = strings.TrimSpace(match[i])
			case "function_arguments":
				funcArgs = strings.TrimSpace(match[i])
			}
		}
		if funcName != "" {
			calls = append(calls, ToolCall{
				Name: funcName,
				Args: map[string]string{"_raw": funcArgs},
				Raw:  match[0],
			})
		}
	}

	return calls, nil
}
