package builtin
import ("context";"encoding/json";"fmt";"os";"strings";"github.com/sadewadee/hera/internal/tools")
type EnvPassthroughTool struct{ AllowList []string }
type envPassthroughArgs struct { Names []string `json:"names"` }
func (t *EnvPassthroughTool) Name() string { return "env_passthrough" }
func (t *EnvPassthroughTool) Description() string { return "Reads allowed environment variables for tool configuration." }
func (t *EnvPassthroughTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"},"description":"Environment variable names to read"}},"required":["names"]}`) }
func (t *EnvPassthroughTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a envPassthroughArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	var results []string
	for _, name := range a.Names {
		val := os.Getenv(name)
		if val != "" { results = append(results, fmt.Sprintf("%s=[set, %d chars]", name, len(val))) } else { results = append(results, fmt.Sprintf("%s=[not set]", name)) }
	}
	return &tools.Result{Content: strings.Join(results, "\n")}, nil
}
func RegisterEnvPassthrough(registry *tools.Registry) { registry.Register(&EnvPassthroughTool{}) }
