package builtin
import ("context";"encoding/json";"fmt";"strings";"github.com/sadewadee/hera/internal/tools")
type FuzzyTool struct{}
type fuzzyArgs struct { Query string `json:"query"`; Items []string `json:"items"` }
func (t *FuzzyTool) Name() string { return "fuzzy" }
func (t *FuzzyTool) Description() string { return "Performs fuzzy string matching against a list of items." }
func (t *FuzzyTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"items":{"type":"array","items":{"type":"string"},"description":"Items to search"}},"required":["query","items"]}`) }
func (t *FuzzyTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a fuzzyArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	query := strings.ToLower(a.Query)
	var matches []string
	for _, item := range a.Items { if strings.Contains(strings.ToLower(item), query) { matches = append(matches, item) } }
	if len(matches) == 0 { return &tools.Result{Content: "No matches found"}, nil }
	return &tools.Result{Content: fmt.Sprintf("Matches (%d): %s", len(matches), strings.Join(matches, ", "))}, nil
}
func RegisterFuzzy(registry *tools.Registry) { registry.Register(&FuzzyTool{}) }
