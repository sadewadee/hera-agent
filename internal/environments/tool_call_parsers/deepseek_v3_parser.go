// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// deepseek_v3_parser.go implements the DeepSeek V3 tool call format which
// uses special unicode tokens with fullwidth angle brackets:
//
//	<|tool_calls_begin|>
//	<|tool_call_begin|>type<|tool_sep|>function_name
//	```json
//	{"arg": "value"}
//	```
//	<|tool_call_end|>
//	<|tool_calls_end|>
package tool_call_parsers

import (
	"regexp"
	"strings"
)

const deepseekV3StartToken = "\u003c\uff5ctool\u2581calls\u2581begin\uff5c\u003e"

var deepseekV3Pattern = regexp.MustCompile(
	`\x{ff5c}tool\x{2581}call\x{2581}begin\x{ff5c}>(?P<type>.*?)<\x{ff5c}tool\x{2581}sep\x{ff5c}>(?P<function_name>.*?)\s*` + "```json" + `\s*(?P<function_arguments>.*?)\s*` + "```" + `\s*<\x{ff5c}tool\x{2581}call\x{2581}end\x{ff5c}>`,
)

// DeepSeekV3Parser parses DeepSeek V3 tool calls.
type DeepSeekV3Parser struct{}

func (p *DeepSeekV3Parser) Name() string { return "deepseek_v3" }

func (p *DeepSeekV3Parser) CanParse(text string) bool {
	return strings.Contains(text, "tool\u2581calls\u2581begin")
}

func (p *DeepSeekV3Parser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	matches := deepseekV3Pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var calls []ToolCall
	for _, match := range matches {
		funcName := ""
		funcArgs := ""
		for i, name := range deepseekV3Pattern.SubexpNames() {
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
