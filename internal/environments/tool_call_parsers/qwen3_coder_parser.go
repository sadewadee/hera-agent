// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// qwen3_coder_parser.go implements the Qwen3-Coder tool call format
// which uses XML-style nested tags:
//
//	<tool_call>
//	<function=function_name>
//	<parameter=param_name>value</parameter>
//	</function>
//	</tool_call>
package tool_call_parsers

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	qwen3ToolCallRegex  = regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>|<tool_call>(.*?)$`)
	qwen3FunctionRegex  = regexp.MustCompile(`(?s)<function=(.*?)</function>|<function=(.*)$`)
	qwen3ParameterRegex = regexp.MustCompile(`(?s)<parameter=(.*?)(?:</parameter>|$)`)
)

const qwen3FunctionPrefix = "<function="

// Qwen3CoderParser parses Qwen3-Coder XML-format tool calls.
type Qwen3CoderParser struct{}

func (p *Qwen3CoderParser) Name() string { return "qwen3_coder" }

func (p *Qwen3CoderParser) CanParse(text string) bool {
	return strings.Contains(text, qwen3FunctionPrefix)
}

func (p *Qwen3CoderParser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	// Find all tool_call blocks.
	tcMatches := qwen3ToolCallRegex.FindAllStringSubmatch(text, -1)
	var rawBlocks []string
	for _, m := range tcMatches {
		if m[1] != "" {
			rawBlocks = append(rawBlocks, m[1])
		} else if m[2] != "" {
			rawBlocks = append(rawBlocks, m[2])
		}
	}

	// Fallback: if no tool_call tags, try the whole text.
	if len(rawBlocks) == 0 {
		rawBlocks = []string{text}
	}

	// Find function blocks within each tool_call.
	var functionStrs []string
	for _, block := range rawBlocks {
		funcMatches := qwen3FunctionRegex.FindAllStringSubmatch(block, -1)
		for _, m := range funcMatches {
			if m[1] != "" {
				functionStrs = append(functionStrs, m[1])
			} else if m[2] != "" {
				functionStrs = append(functionStrs, m[2])
			}
		}
	}

	if len(functionStrs) == 0 {
		return nil, nil
	}

	var calls []ToolCall
	for _, funcStr := range functionStrs {
		tc := parseFunctionCall(funcStr)
		if tc != nil {
			calls = append(calls, *tc)
		}
	}

	return calls, nil
}

// parseFunctionCall parses a single <function=name>...</function> block.
func parseFunctionCall(funcStr string) *ToolCall {
	// Extract function name: everything before the first '>'.
	gtIdx := strings.Index(funcStr, ">")
	if gtIdx < 0 {
		return nil
	}
	funcName := strings.TrimSpace(funcStr[:gtIdx])
	paramsStr := funcStr[gtIdx+1:]

	// Extract parameters.
	paramDict := make(map[string]string)
	paramMatches := qwen3ParameterRegex.FindAllStringSubmatch(paramsStr, -1)
	for _, matchText := range paramMatches {
		raw := matchText[1]
		eqIdx := strings.Index(raw, ">")
		if eqIdx < 0 {
			continue
		}
		paramName := strings.TrimSpace(raw[:eqIdx])
		paramValue := raw[eqIdx+1:]

		// Clean up whitespace.
		paramValue = strings.TrimPrefix(paramValue, "\n")
		paramValue = strings.TrimSuffix(paramValue, "\n")
		paramDict[paramName] = paramValue
	}

	argsJSON, _ := json.Marshal(paramDict)
	return &ToolCall{
		Name: funcName,
		Args: paramDict,
		Raw:  string(argsJSON),
	}
}
