package tool_call_parsers
import ("regexp"; "strings")
type XMLParser struct{}
func (p *XMLParser) Name() string { return "xml" }
func (p *XMLParser) CanParse(text string) bool { return strings.Contains(text, "<tool_call") || strings.Contains(text, "<function_call") }
func (p *XMLParser) Parse(text string) ([]ToolCall, error) {
	re := regexp.MustCompile(`<tool_call\s+name="([^"]+)">([\s\S]*?)</tool_call>`)
	matches := re.FindAllStringSubmatch(text, -1)
	var calls []ToolCall
	for _, m := range matches { calls = append(calls, ToolCall{Name: m[1], Raw: m[0]}) }
	return calls, nil
}
