// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// kimi_k2_parser.go implements the Kimi K2 tool call format:
//
//	<|tool_calls_section_begin|>
//	<|tool_call_begin|>function_id:0<|tool_call_argument_begin|>{"arg": "val"}<|tool_call_end|>
//	<|tool_calls_section_end|>
//
// The function_id format is typically "functions.func_name:index".
package tool_call_parsers

import (
	"regexp"
	"strings"
)

var kimiK2StartTokens = []string{
	"<|tool_calls_section_begin|>",
	"<|tool_call_section_begin|>",
}

var kimiK2Pattern = regexp.MustCompile(
	`<\|tool_call_begin\|>\s*(?P<tool_call_id>[^<]+:\d+)\s*` +
		`<\|tool_call_argument_begin\|>\s*` +
		`(?P<function_arguments>[\s\S]*?)\s*` +
		`<\|tool_call_end\|>`,
)

// KimiK2Parser parses Kimi K2 tool calls.
type KimiK2Parser struct{}

func (p *KimiK2Parser) Name() string { return "kimi_k2" }

func (p *KimiK2Parser) CanParse(text string) bool {
	for _, token := range kimiK2StartTokens {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func (p *KimiK2Parser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	matches := kimiK2Pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var calls []ToolCall
	for _, match := range matches {
		functionID := ""
		functionArgs := ""
		for i, name := range kimiK2Pattern.SubexpNames() {
			switch name {
			case "tool_call_id":
				functionID = match[i]
			case "function_arguments":
				functionArgs = strings.TrimSpace(match[i])
			}
		}

		if functionID == "" {
			continue
		}

		// Extract function name: "functions.get_weather:0" -> "get_weather"
		parts := strings.Split(functionID, ":")
		namePart := parts[0]
		dotParts := strings.Split(namePart, ".")
		funcName := dotParts[len(dotParts)-1]

		calls = append(calls, ToolCall{
			Name: funcName,
			Args: map[string]string{"_raw": functionArgs},
			Raw:  match[0],
		})
	}

	return calls, nil
}
