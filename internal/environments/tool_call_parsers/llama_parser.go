// Package tool_call_parsers provides parsers for extracting tool calls
// from LLM output in various formats.
//
// llama_parser.go implements the Llama 3.x/4 tool call parser. The model
// outputs JSON objects with "name" and "arguments" (or "parameters") keys,
// optionally preceded by <|python_tag|>. Supports multiple JSON objects.
package tool_call_parsers

import (
	"encoding/json"
	"strings"
)

const llamaBotToken = "<|python_tag|>"

// LlamaParser parses Llama 3.x and 4 JSON-format tool calls.
type LlamaParser struct{}

func (p *LlamaParser) Name() string { return "llama3_json" }

func (p *LlamaParser) CanParse(text string) bool {
	return strings.Contains(text, llamaBotToken) || strings.Contains(text, "{")
}

func (p *LlamaParser) Parse(text string) ([]ToolCall, error) {
	if !p.CanParse(text) {
		return nil, nil
	}

	var calls []ToolCall
	endIndex := -1

	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		if i <= endIndex {
			continue
		}

		// Try to decode a JSON object starting at this position.
		obj, consumed, err := decodeJSONObject(text[i:])
		if err != nil {
			continue
		}
		endIndex = i + consumed

		name, _ := obj["name"].(string)
		if name == "" {
			continue
		}

		var args any
		if a, ok := obj["arguments"]; ok {
			args = a
		} else if a, ok := obj["parameters"]; ok {
			args = a
		}
		if args == nil {
			continue
		}

		// Normalise arguments to JSON string.
		var argsStr string
		switch v := args.(type) {
		case string:
			argsStr = v
		default:
			data, _ := json.Marshal(v)
			argsStr = string(data)
		}

		calls = append(calls, ToolCall{
			Name: name,
			Args: map[string]string{"_raw": argsStr},
			Raw:  text[i:endIndex],
		})
	}

	return calls, nil
}

// decodeJSONObject attempts to decode a JSON object from the start of s.
// Returns the decoded map, the number of bytes consumed, and any error.
func decodeJSONObject(s string) (map[string]any, int, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, 0, err
	}
	// Calculate consumed bytes from the decoder offset.
	consumed := int(dec.InputOffset())
	return obj, consumed, nil
}
