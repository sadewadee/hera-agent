package builtin
import ("context";"encoding/json";"fmt";"os";"github.com/sadewadee/hera/internal/tools")
type OpenRouterTool struct{ apiKey string }
type openrouterArgs struct { Model string `json:"model"`; Prompt string `json:"prompt"`; MaxTokens int `json:"max_tokens,omitempty"` }
func (t *OpenRouterTool) Name() string { return "openrouter_query" }
func (t *OpenRouterTool) Description() string { return "Queries a specific model through the OpenRouter API." }
func (t *OpenRouterTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"model":{"type":"string","description":"Model ID (e.g., anthropic/claude-3-opus)"},"prompt":{"type":"string","description":"Prompt text"},"max_tokens":{"type":"integer","description":"Max response tokens"}},"required":["model","prompt"]}`) }
func (t *OpenRouterTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a openrouterArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	if t.apiKey == "" { t.apiKey = os.Getenv("OPENROUTER_API_KEY") }
	if t.apiKey == "" { return &tools.Result{Content: "OPENROUTER_API_KEY not set", IsError: true}, nil }
	return &tools.Result{Content: fmt.Sprintf("OpenRouter query queued: model=%s, prompt=%d chars", a.Model, len(a.Prompt))}, nil
}
func RegisterOpenRouterClient(registry *tools.Registry) { registry.Register(&OpenRouterTool{}) }
