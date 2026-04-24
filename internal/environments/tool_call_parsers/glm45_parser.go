// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// glm45_parser.go implements the GLM 4.5 (GLM-4-MoE) tool call format
// which uses custom arg_key/arg_value tags:
//
//	<tool_call>function_name
//	<arg_key>param1</arg_key><arg_value>value1</arg_value>
//	</tool_call>
package tool_call_parsers

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	glm45FuncCallRegex   = regexp.MustCompile(`(?s)<tool_call>.*?</tool_call>`)
	glm45FuncDetailRegex = regexp.MustCompile(`(?s)<tool_call>([^\n]*)\n(.*)</tool_call>`)
	glm45FuncArgRegex    = regexp.MustCompile(`(?s)<arg_key>(.*?)</arg_key>\s*<arg_value>(.*?)</arg_value>`)
)

// GLM45Parser parses GLM 4.5 tool calls.
type GLM45Parser struct {
	funcCallRegex   *regexp.Regexp
	funcDetailRegex *regexp.Regexp
	funcArgRegex    *regexp.Regexp
}

// NewGLM45Parser creates a new GLM 4.5 parser with default regexes.
func NewGLM45Parser() *GLM45Parser {
	return &GLM45Parser{
		funcCallRegex:   glm45FuncCallRegex,
		funcDetailRegex: glm45FuncDetailRegex,
		funcArgRegex:    glm45FuncArgRegex,
	}
}

func (p *GLM45Parser) Name() string { return "glm45" }

func (p *GLM45Parser) CanParse(text string) bool {
	return strings.Contains(text, "<tool_call>")
}

func (p *GLM45Parser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	matched := p.funcCallRegex.FindAllString(text, -1)
	if len(matched) == 0 {
		return nil, nil
	}

	var calls []ToolCall
	for _, block := range matched {
		detail := p.funcDetailRegex.FindStringSubmatch(block)
		if detail == nil || len(detail) < 3 {
			continue
		}

		funcName := strings.TrimSpace(detail[1])
		funcArgsRaw := detail[2]

		// Parse arg_key/arg_value pairs.
		pairs := p.funcArgRegex.FindAllStringSubmatch(funcArgsRaw, -1)
		argMap := make(map[string]string)
		for _, pair := range pairs {
			if len(pair) >= 3 {
				key := strings.TrimSpace(pair[1])
				value := strings.TrimSpace(pair[2])
				argMap[key] = value
			}
		}

		argsJSON, _ := json.Marshal(argMap)
		calls = append(calls, ToolCall{
			Name: funcName,
			Args: argMap,
			Raw:  string(argsJSON),
		})
	}

	return calls, nil
}
