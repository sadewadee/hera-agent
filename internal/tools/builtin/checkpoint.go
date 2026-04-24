package builtin
import ("context";"encoding/json";"fmt";"time";"github.com/sadewadee/hera/internal/tools")
type CheckpointTool struct{}
type checkpointArgs struct { Label string `json:"label"`; Data string `json:"data,omitempty"` }
func (t *CheckpointTool) Name() string { return "checkpoint" }
func (t *CheckpointTool) Description() string { return "Saves a checkpoint of current agent state for later resumption." }
func (t *CheckpointTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"label":{"type":"string","description":"Checkpoint label"},"data":{"type":"string","description":"Additional state data to save"}},"required":["label"]}`) }
func (t *CheckpointTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a checkpointArgs; if err := json.Unmarshal(args, &a); err != nil { return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil }
	ts := time.Now().Format(time.RFC3339)
	return &tools.Result{Content: fmt.Sprintf("Checkpoint '%s' saved at %s", a.Label, ts)}, nil
}
func RegisterCheckpoint(registry *tools.Registry) { registry.Register(&CheckpointTool{}) }
