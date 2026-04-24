package builtin
import ("context";"encoding/json";"fmt";"runtime";"time";"github.com/sadewadee/hera/internal/tools")
type DebugTool struct{}
type debugArgs struct { Section string `json:"section,omitempty"` }
func (t *DebugTool) Name() string { return "debug" }
func (t *DebugTool) Description() string { return "Returns debug information about the agent runtime." }
func (t *DebugTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{"section":{"type":"string","enum":["runtime","memory","goroutines","all"],"description":"Debug section"}}}`) }
func (t *DebugTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a debugArgs; json.Unmarshal(args, &a)
	if a.Section == "" { a.Section = "all" }
	var m runtime.MemStats; runtime.ReadMemStats(&m)
	info := fmt.Sprintf("Go Version: %s\nOS/Arch: %s/%s\nGoroutines: %d\nHeap Alloc: %d MB\nSys: %d MB\nGC Cycles: %d\nUptime: %s",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumGoroutine(), m.HeapAlloc/1024/1024, m.Sys/1024/1024, m.NumGC, time.Since(time.Time{}).String())
	return &tools.Result{Content: info}, nil
}
func RegisterDebug(registry *tools.Registry) { registry.Register(&DebugTool{}) }
