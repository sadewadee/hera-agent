package builtin
import ("context";"encoding/json";"fmt";"strings";"github.com/sadewadee/hera/internal/tools")
type ToolResultFormatter struct{}
type toolResultArgs struct { Content string `json:"content"`; Format string `json:"format,omitempty"`; MaxLength int `json:"max_length,omitempty"` }
func (t *ToolResultFormatter) Name() string { return "tool_result" }
func (t *ToolResultFormatter) Description() string { return "Formats and processes tool output for display." }
func (t *ToolResultFormatter) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"Raw tool output"},"format":{"type":"string","enum":["text","json","markdown","table"],"description":"Output format"},"max_length":{"type":"integer","description":"Max output length"}},"required":["content"]}`) }
func (t *ToolResultFormatter) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a toolResultArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	content := a.Content
	if a.MaxLength > 0 && len(content) > a.MaxLength { content = content[:a.MaxLength] + "..." }
	switch a.Format {
	case "json": content = "```json\n" + content + "\n```"
	case "markdown": break
	case "table": content = strings.ReplaceAll(content, "\t", " | ")
	}
	return &tools.Result{Content: content}, nil
}
func RegisterToolResult(registry *tools.Registry) { registry.Register(&ToolResultFormatter{}) }
