package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// FileReadTool reads the contents of a file.
type FileReadTool struct {
	protectedPaths []string
}

type fileReadArgs struct {
	Path string `json:"path"`
}

func (f *FileReadTool) Name() string {
	return "file_read"
}

func (f *FileReadTool) Description() string {
	return "Reads the contents of a file at the given path. Respects protected path restrictions."
}

func (f *FileReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Absolute or relative path to the file to read."
			}
		},
		"required": ["path"]
	}`)
}

func (f *FileReadTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params fileReadArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Path == "" {
		return &tools.Result{Content: "path is required", IsError: true}, nil
	}

	absPath, err := expandAndAbs(params.Path)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("resolve path: %v", err), IsError: true}, nil
	}

	if isProtected(absPath, f.protectedPaths) {
		return &tools.Result{Content: fmt.Sprintf("access denied: %s is a protected path", params.Path), IsError: true}, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("read file: %v", err), IsError: true}, nil
	}

	// Limit output to 100KB to avoid blowing up context
	const maxSize = 100 * 1024
	content := string(data)
	if len(content) > maxSize {
		content = content[:maxSize] + "\n... [truncated, file exceeds 100KB]"
	}

	return &tools.Result{Content: content}, nil
}

// FileWriteTool writes content to a file.
type FileWriteTool struct {
	protectedPaths []string
}

type fileWriteArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append,omitempty"`
}

func (f *FileWriteTool) Name() string {
	return "file_write"
}

func (f *FileWriteTool) Description() string {
	return "Writes content to a file. Creates the file and parent directories if they do not exist. Can optionally append."
}

func (f *FileWriteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Absolute or relative path to the file to write."
			},
			"content": {
				"type": "string",
				"description": "The content to write to the file."
			},
			"append": {
				"type": "boolean",
				"description": "If true, append to existing file instead of overwriting. Defaults to false."
			}
		},
		"required": ["path", "content"]
	}`)
}

func (f *FileWriteTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params fileWriteArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Path == "" {
		return &tools.Result{Content: "path is required", IsError: true}, nil
	}

	absPath, err := expandAndAbs(params.Path)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("resolve path: %v", err), IsError: true}, nil
	}

	if isProtected(absPath, f.protectedPaths) {
		return &tools.Result{Content: fmt.Sprintf("access denied: %s is a protected path", params.Path), IsError: true}, nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &tools.Result{Content: fmt.Sprintf("create directory: %v", err), IsError: true}, nil
	}

	flag := os.O_WRONLY | os.O_CREATE
	if params.Append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(absPath, flag, 0o644)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open file: %v", err), IsError: true}, nil
	}
	defer file.Close()

	n, err := file.WriteString(params.Content)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("write file: %v", err), IsError: true}, nil
	}

	action := "wrote"
	if params.Append {
		action = "appended"
	}

	return &tools.Result{Content: fmt.Sprintf("%s %d bytes to %s", action, n, absPath)}, nil
}

// RegisterFiles registers file_read and file_write tools with the given registry.
func RegisterFiles(registry *tools.Registry, protectedPaths []string) {
	registry.Register(&FileReadTool{protectedPaths: protectedPaths})
	registry.Register(&FileWriteTool{protectedPaths: protectedPaths})
}

// expandAndAbs resolves a user-supplied path via paths.Normalize, which
// handles ~, $HERA_HOME, ${HERA_HOME}, the .hera/… safety-net redirect,
// and CWD-relative fallback. Kept as a thin wrapper so existing callers
// that treat it as fallible keep their error path; paths.Normalize itself
// is non-failing, so the error is always nil today.
func expandAndAbs(path string) (string, error) {
	return paths.Normalize(path), nil
}

// isProtected checks if a path falls within any protected path.
func isProtected(absPath string, protectedPaths []string) bool {
	for _, pp := range protectedPaths {
		expanded, err := expandAndAbs(pp)
		if err != nil {
			continue
		}
		// Check if the path is the protected path or is inside it
		if absPath == expanded || strings.HasPrefix(absPath, expanded+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
