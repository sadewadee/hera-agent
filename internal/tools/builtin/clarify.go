package builtin
import ("context";"encoding/json";"fmt";"github.com/sadewadee/hera/internal/tools")
type ClarifyTool struct{}
type clarifyArgs struct { Question string `json:"question"`; Options []string `json:"options,omitempty"` }
func (t *ClarifyTool) Name() string { return "clarify" }
func (t *ClarifyTool) Description() string { return "Asks the user a clarifying question before proceeding with a task." }
func (t *ClarifyTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"question":{"type":"string","description":"Clarifying question to ask"},"options":{"type":"array","items":{"type":"string"},"description":"Optional multiple choice options"}},"required":["question"]}`) }
func (t *ClarifyTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a clarifyArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	result := fmt.Sprintf("CLARIFICATION NEEDED: %s", a.Question)
	if len(a.Options) > 0 { result += fmt.Sprintf("\nOptions: %v", a.Options) }
	return &tools.Result{Content: result}, nil
}
func RegisterClarify(registry *tools.Registry) { registry.Register(&ClarifyTool{}) }
