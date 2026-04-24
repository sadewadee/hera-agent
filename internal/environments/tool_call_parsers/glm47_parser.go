// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// glm47_parser.go implements the GLM 4.7 tool call parser. Same as
// GLM 4.5 but with slightly different regex patterns that handle
// newlines between key/value pairs.
package tool_call_parsers

import (
	"regexp"
)

// GLM47Parser parses GLM 4.7 tool calls. Extends GLM 4.5 with
// updated regex patterns.
type GLM47Parser struct {
	GLM45Parser
}

// NewGLM47Parser creates a new GLM 4.7 parser.
func NewGLM47Parser() *GLM47Parser {
	p := &GLM47Parser{}
	p.funcCallRegex = glm45FuncCallRegex
	// GLM 4.7 uses a slightly different detail regex that captures
	// optional arg_key content within the tool_call wrapper.
	p.funcDetailRegex = regexp.MustCompile(`(?s)<tool_call>(.*?)(<arg_key>.*?)?</tool_call>`)
	// GLM 4.7 handles newlines between arg_key and arg_value tags.
	p.funcArgRegex = regexp.MustCompile(`(?s)<arg_key>(.*?)</arg_key>(?:\\n|\s)*<arg_value>(.*?)</arg_value>`)
	return p
}

func (p *GLM47Parser) Name() string { return "glm47" }
