package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sadewadee/hera/internal/tools"
)

type CodeExecTool struct{}

type codeExecArgs struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Timeout  int    `json:"timeout,omitempty"`
}

func (t *CodeExecTool) Name() string        { return "code_exec" }
func (t *CodeExecTool) Description() string  { return "Executes code snippets in supported languages (python, node, bash, go)." }
func (t *CodeExecTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"language":{"type":"string","enum":["python","node","bash","go"],"description":"Programming language"},"code":{"type":"string","description":"Code to execute"},"timeout":{"type":"integer","description":"Timeout in seconds (default 30)"}},"required":["language","code"]}`)
}

func (t *CodeExecTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a codeExecArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	tmpDir, _ := os.MkdirTemp("", "hera-exec-*")
	defer os.RemoveAll(tmpDir)
	var cmdName string; var cmdArgs []string; var ext string
	switch a.Language {
	case "python": cmdName = "python3"; cmdArgs = []string{}; ext = ".py"
	case "node": cmdName = "node"; cmdArgs = []string{}; ext = ".js"
	case "bash": cmdName = "bash"; cmdArgs = []string{}; ext = ".sh"
	case "go": cmdName = "go"; cmdArgs = []string{"run"}; ext = ".go"
	default:
		return &tools.Result{Content: "unsupported language: " + a.Language, IsError: true}, nil
	}
	fpath := filepath.Join(tmpDir, "script"+ext)
	if err := os.WriteFile(fpath, []byte(a.Code), 0644); err != nil {
		return &tools.Result{Content: fmt.Sprintf("write error: %v", err), IsError: true}, nil
	}
	cmdArgs = append(cmdArgs, fpath)
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout; cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 { output += "\nSTDERR: " + stderr.String() }
	if err != nil { output += fmt.Sprintf("\nError: %v", err) }
	return &tools.Result{Content: output}, nil
}

func RegisterCodeExec(registry *tools.Registry) { registry.Register(&CodeExecTool{}) }
