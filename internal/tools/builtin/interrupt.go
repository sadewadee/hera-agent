package builtin
import ("context";"encoding/json";"fmt";"github.com/sadewadee/hera/internal/tools")
type InterruptTool struct{}
type interruptArgs struct { Reason string `json:"reason"`; Graceful bool `json:"graceful,omitempty"` }
func (t *InterruptTool) Name() string { return "interrupt" }
func (t *InterruptTool) Description() string { return "Signals an interruption of the current task execution." }
func (t *InterruptTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"reason":{"type":"string","description":"Reason for interruption"},"graceful":{"type":"boolean","description":"Whether to allow cleanup"}},"required":["reason"]}`) }
func (t *InterruptTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a interruptArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	mode := "immediate"; if a.Graceful { mode = "graceful" }
	return &tools.Result{Content: fmt.Sprintf("INTERRUPT (%s): %s", mode, a.Reason)}, nil
}
func RegisterInterrupt(registry *tools.Registry) { registry.Register(&InterruptTool{}) }
