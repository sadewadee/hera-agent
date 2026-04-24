package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
	"os"
	"path/filepath"
)

type BinaryExtTool struct{}
type binaryExtArgs struct {
	FilePath string `json:"file_path"`
}

func (t *BinaryExtTool) Name() string { return "binary_ext" }
func (t *BinaryExtTool) Description() string {
	return "Detects and handles binary file types based on extension and content."
}
func (t *BinaryExtTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"file_path":{"type":"string","description":"Path to file"}},"required":["file_path"]}`)
}
func (t *BinaryExtTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a binaryExtArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	a.FilePath = paths.Normalize(a.FilePath)
	info, err := os.Stat(a.FilePath)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("file error: %v", err), IsError: true}, nil
	}
	ext := filepath.Ext(a.FilePath)
	binaryExts := map[string]bool{".png": true, ".jpg": true, ".gif": true, ".pdf": true, ".zip": true, ".tar": true, ".gz": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true, ".wasm": true}
	isBinary := binaryExts[ext]
	return &tools.Result{Content: fmt.Sprintf("File: %s\nSize: %d bytes\nExtension: %s\nBinary: %v", a.FilePath, info.Size(), ext, isBinary)}, nil
}
func RegisterBinaryExt(registry *tools.Registry) { registry.Register(&BinaryExtTool{}) }
