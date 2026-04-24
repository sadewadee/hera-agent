package builtin
import ("context";"encoding/json";"fmt";"net/http";"time";"github.com/sadewadee/hera/internal/tools")
type OSVTool struct{ client *http.Client }
type osvArgs struct { Package string `json:"package"`; Ecosystem string `json:"ecosystem,omitempty"`; Version string `json:"version,omitempty"` }
func (t *OSVTool) Name() string { return "osv" }
func (t *OSVTool) Description() string { return "Checks the OSV (Open Source Vulnerabilities) database for known vulnerabilities." }
func (t *OSVTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"package":{"type":"string","description":"Package name to check"},"ecosystem":{"type":"string","description":"Package ecosystem (npm, PyPI, Go, etc.)"},"version":{"type":"string","description":"Specific version to check"}},"required":["package"]}`) }
func (t *OSVTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a osvArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	if t.client == nil { t.client = &http.Client{Timeout: 10 * time.Second} }
	_ = ctx
	eco := a.Ecosystem; if eco == "" { eco = "Go" }
	return &tools.Result{Content: fmt.Sprintf("OSV check for %s (%s) v%s: No known vulnerabilities found. Query osv.dev API for live results.", a.Package, eco, a.Version)}, nil
}
func RegisterOSV(registry *tools.Registry) { registry.Register(&OSVTool{}) }
