package tool_call_parsers

// ToolCall represents a parsed tool call.
type ToolCall struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
	Raw  string            `json:"raw"`
}

// ToolCallParser is the interface for all tool call format parsers.
type ToolCallParser interface {
	Name() string
	Parse(text string) ([]ToolCall, error)
	CanParse(text string) bool
}

// ParseAll tries all parsers and returns the first successful result.
func ParseAll(parsers []ToolCallParser, text string) ([]ToolCall, string) {
	for _, p := range parsers {
		if p.CanParse(text) {
			if calls, err := p.Parse(text); err == nil && len(calls) > 0 {
				return calls, p.Name()
			}
		}
	}
	return nil, ""
}
