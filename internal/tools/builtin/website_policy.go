package builtin
import ("context";"encoding/json";"fmt";"io";"net/http";"strings";"time";"github.com/sadewadee/hera/internal/tools")
type WebsitePolicyTool struct{ client *http.Client }
type websitePolicyArgs struct { URL string `json:"url"`; Check string `json:"check,omitempty"` }
func (t *WebsitePolicyTool) Name() string { return "website_policy" }
func (t *WebsitePolicyTool) Description() string { return "Checks a website's robots.txt and terms of service." }
func (t *WebsitePolicyTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"Website URL to check"},"check":{"type":"string","enum":["robots","tos","all"],"description":"What to check"}},"required":["url"]}`) }
func (t *WebsitePolicyTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a websitePolicyArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	if t.client == nil { t.client = &http.Client{Timeout: 10 * time.Second} }
	robotsURL := strings.TrimRight(a.URL, "/") + "/robots.txt"
	req, _ := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	resp, err := t.client.Do(req)
	if err != nil { return &tools.Result{Content: fmt.Sprintf("Failed to fetch robots.txt: %v", err)}, nil }
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
	return &tools.Result{Content: fmt.Sprintf("robots.txt for %s:\n%s", a.URL, string(body))}, nil
}
func RegisterWebsitePolicy(registry *tools.Registry) { registry.Register(&WebsitePolicyTool{}) }
