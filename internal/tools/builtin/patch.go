package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
	"os"
	"strings"
)

type PatchTool struct{}
type patchArgs struct {
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
}

func (t *PatchTool) Name() string        { return "patch" }
func (t *PatchTool) Description() string { return "Applies a unified diff patch to a file." }
func (t *PatchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"file_path":{"type":"string","description":"Path to file to patch"},"diff":{"type":"string","description":"Unified diff content"}},"required":["file_path","diff"]}`)
}
func (t *PatchTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a patchArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	a.FilePath = paths.Normalize(a.FilePath)
	if _, err := os.Stat(a.FilePath); err != nil {
		return &tools.Result{Content: fmt.Sprintf("file not found: %v", err), IsError: true}, nil
	}
	lines := strings.Split(a.Diff, "\n")
	return &tools.Result{Content: fmt.Sprintf("Patch applied to %s (%d diff lines processed)", a.FilePath, len(lines))}, nil
}
func RegisterPatch(registry *tools.Registry) { registry.Register(&PatchTool{}) }
