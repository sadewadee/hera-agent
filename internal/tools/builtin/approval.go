package builtin
import ("context";"encoding/json";"fmt";"github.com/sadewadee/hera/internal/tools")
type ApprovalTool struct{}
type approvalArgs struct { Action string `json:"action"`; Description string `json:"description"`; Risk string `json:"risk,omitempty"` }
func (t *ApprovalTool) Name() string { return "approval" }
func (t *ApprovalTool) Description() string { return "Requests human approval before performing a sensitive action." }
func (t *ApprovalTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","description":"Action requiring approval"},"description":{"type":"string","description":"Detailed description"},"risk":{"type":"string","enum":["low","medium","high","critical"],"description":"Risk level"}},"required":["action","description"]}`) }
func (t *ApprovalTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a approvalArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	risk := a.Risk; if risk == "" { risk = "medium" }
	return &tools.Result{Content: fmt.Sprintf("APPROVAL REQUIRED [%s risk]: %s\n%s\nAwaiting human approval...", risk, a.Action, a.Description)}, nil
}
func RegisterApproval(registry *tools.Registry) { registry.Register(&ApprovalTool{}) }
