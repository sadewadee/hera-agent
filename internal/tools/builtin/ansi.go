package builtin
import ("context";"encoding/json";"fmt";"github.com/sadewadee/hera/internal/tools")
type ANSITool struct{}
type ansiArgs struct { Text string `json:"text"`; Style string `json:"style,omitempty"`; Color string `json:"color,omitempty"` }
func (t *ANSITool) Name() string { return "ansi" }
func (t *ANSITool) Description() string { return "Applies ANSI formatting (colors, styles) to text for terminal display." }
func (t *ANSITool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"Text to format"},"style":{"type":"string","enum":["bold","dim","italic","underline","blink","reverse","hidden","strikethrough"],"description":"Style"},"color":{"type":"string","description":"Color name or hex (#RRGGBB)"}},"required":["text"]}`) }
func (t *ANSITool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a ansiArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	styles := map[string]string{"bold": "1", "dim": "2", "italic": "3", "underline": "4", "blink": "5", "reverse": "7", "hidden": "8", "strikethrough": "9"}
	code := "0"
	if s, ok := styles[a.Style]; ok { code = s }
	return &tools.Result{Content: fmt.Sprintf("\033[%sm%s\033[0m", code, a.Text)}, nil
}
func RegisterANSI(registry *tools.Registry) { registry.Register(&ANSITool{}) }
