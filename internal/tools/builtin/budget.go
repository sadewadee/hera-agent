package builtin
import ("context";"encoding/json";"fmt";"sync";"github.com/sadewadee/hera/internal/tools")
type BudgetTool struct{ mu sync.Mutex; tokensUsed int; maxTokens int; costUSD float64 }
type budgetArgs struct { Action string `json:"action"`; Tokens int `json:"tokens,omitempty"`; MaxTokens int `json:"max_tokens,omitempty"` }
func (t *BudgetTool) Name() string { return "budget" }
func (t *BudgetTool) Description() string { return "Tracks and manages token/cost budget for the current session." }
func (t *BudgetTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["status","add_usage","set_limit","reset"],"description":"Budget action"},"tokens":{"type":"integer","description":"Token count"},"max_tokens":{"type":"integer","description":"Max token limit"}},"required":["action"]}`) }
func (t *BudgetTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a budgetArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	t.mu.Lock(); defer t.mu.Unlock()
	switch a.Action {
	case "status": return &tools.Result{Content: fmt.Sprintf("Budget: %d/%d tokens used, $%.4f cost", t.tokensUsed, t.maxTokens, t.costUSD)}, nil
	case "add_usage": t.tokensUsed += a.Tokens; t.costUSD += float64(a.Tokens) * 0.00001; return &tools.Result{Content: fmt.Sprintf("Added %d tokens", a.Tokens)}, nil
	case "set_limit": t.maxTokens = a.MaxTokens; return &tools.Result{Content: fmt.Sprintf("Budget limit set to %d tokens", a.MaxTokens)}, nil
	case "reset": t.tokensUsed = 0; t.costUSD = 0; return &tools.Result{Content: "Budget reset"}, nil
	default: return &tools.Result{Content: "unknown action", IsError: true}, nil
	}
}
func RegisterBudget(registry *tools.Registry) { registry.Register(&BudgetTool{}) }
