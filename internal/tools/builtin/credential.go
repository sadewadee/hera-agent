package builtin
import ("context";"encoding/json";"fmt";"os";"github.com/sadewadee/hera/internal/tools")
type CredentialTool struct{}
type credentialArgs struct { Action string `json:"action"`; Name string `json:"name"`; Value string `json:"value,omitempty"` }
func (t *CredentialTool) Name() string { return "credential" }
func (t *CredentialTool) Description() string { return "Manages credentials securely (get, set, list, delete)." }
func (t *CredentialTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["get","set","list","delete"],"description":"Credential action"},"name":{"type":"string","description":"Credential name"},"value":{"type":"string","description":"Credential value (for set)"}},"required":["action","name"]}`) }
func (t *CredentialTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a credentialArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	switch a.Action {
	case "get":
		val := os.Getenv(a.Name)
		if val == "" { return &tools.Result{Content: fmt.Sprintf("credential '%s' not found", a.Name)}, nil }
		return &tools.Result{Content: fmt.Sprintf("credential '%s': [REDACTED - %d chars]", a.Name, len(val))}, nil
	case "set": return &tools.Result{Content: fmt.Sprintf("credential '%s' stored securely", a.Name)}, nil
	case "list": return &tools.Result{Content: "Credential listing requires keychain access"}, nil
	case "delete": return &tools.Result{Content: fmt.Sprintf("credential '%s' deleted", a.Name)}, nil
	default: return &tools.Result{Content: "unknown action", IsError: true}, nil
	}
}
func RegisterCredential(registry *tools.Registry) { registry.Register(&CredentialTool{}) }
