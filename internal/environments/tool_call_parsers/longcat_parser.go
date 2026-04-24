// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// longcat_parser.go implements the Longcat Flash Chat tool call parser.
// Same JSON-inside-XML logic as the default JSON tool-call parser, but
// uses <longcat_tool_call> tags instead of <tool_call>.
package tool_call_parsers

import (
	"encoding/json"
	"regexp"
	"strings"
)

var longcatPattern = regexp.MustCompile(
	`(?s)<longcat_tool_call>\s*(.*?)\s*</longcat_tool_call>|<longcat_tool_call>\s*(.*)`,
)

// LongcatParser parses Longcat Flash Chat tool calls.
type LongcatParser struct{}

func (p *LongcatParser) Name() string { return "longcat" }

func (p *LongcatParser) CanParse(text string) bool {
	return strings.Contains(text, "<longcat_tool_call>")
}

func (p *LongcatParser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	matches := longcatPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var calls []ToolCall
	for _, match := range matches {
		rawJSON := match[1]
		if rawJSON == "" {
			rawJSON = match[2]
		}
		rawJSON = strings.TrimSpace(rawJSON)
		if rawJSON == "" {
			continue
		}

		var tcData struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(rawJSON), &tcData); err != nil {
			continue
		}

		argsMap := make(map[string]string)
		for k, v := range tcData.Arguments {
			switch val := v.(type) {
			case string:
				argsMap[k] = val
			default:
				data, _ := json.Marshal(val)
				argsMap[k] = string(data)
			}
		}

		argsJSON, _ := json.Marshal(tcData.Arguments)
		calls = append(calls, ToolCall{
			Name: tcData.Name,
			Args: argsMap,
			Raw:  string(argsJSON),
		})
	}

	return calls, nil
}
