package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// GitTool provides Git repository operations.
type GitTool struct{}

type gitToolArgs struct {
	Action  string   `json:"action"`
	Path    string   `json:"path,omitempty"`
	Message string   `json:"message,omitempty"`
	Branch  string   `json:"branch,omitempty"`
	Files   []string `json:"files,omitempty"`
	Count   int      `json:"count,omitempty"`
}

func (t *GitTool) Name() string { return "git" }

func (t *GitTool) Description() string {
	return "Performs Git operations: status, log, diff, add, commit, branch, checkout."
}

func (t *GitTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["status", "log", "diff", "add", "commit", "branch_list", "checkout", "current_branch"],
				"description": "Git action to perform."
			},
			"path": {
				"type": "string",
				"description": "Repository path. Defaults to current directory."
			},
			"message": {
				"type": "string",
				"description": "Commit message (for commit action)."
			},
			"branch": {
				"type": "string",
				"description": "Branch name (for checkout action)."
			},
			"files": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Files to stage (for add action). Empty means all."
			},
			"count": {
				"type": "integer",
				"description": "Number of log entries. Defaults to 10."
			}
		},
		"required": ["action"]
	}`)
}

func (t *GitTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a gitToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	dir := a.Path
	if dir == "" {
		dir = "."
	}

	runGit := func(gitArgs ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", gitArgs...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	switch a.Action {
	case "status":
		out, err := runGit("status", "--porcelain")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git status: %v", err), IsError: true}, nil
		}
		if out == "" {
			out = "Working tree clean"
		}
		return &tools.Result{Content: out}, nil

	case "log":
		count := a.Count
		if count <= 0 {
			count = 10
		}
		out, err := runGit("log", fmt.Sprintf("--max-count=%d", count), "--oneline", "--graph")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git log: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "diff":
		out, err := runGit("diff")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git diff: %v", err), IsError: true}, nil
		}
		if out == "" {
			out = "No unstaged changes"
		}
		// Truncate large diffs
		if len(out) > 50*1024 {
			out = out[:50*1024] + "\n...[truncated]"
		}
		return &tools.Result{Content: out}, nil

	case "add":
		gitArgs := []string{"add"}
		if len(a.Files) > 0 {
			gitArgs = append(gitArgs, a.Files...)
		} else {
			gitArgs = append(gitArgs, ".")
		}
		out, err := runGit(gitArgs...)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git add: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: "Files staged successfully"}, nil

	case "commit":
		if a.Message == "" {
			return &tools.Result{Content: "commit message is required", IsError: true}, nil
		}
		out, err := runGit("commit", "-m", a.Message)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git commit: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "branch_list":
		out, err := runGit("branch", "-a")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git branch: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	case "checkout":
		if a.Branch == "" {
			return &tools.Result{Content: "branch name is required", IsError: true}, nil
		}
		out, err := runGit("checkout", a.Branch)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git checkout: %s %v", out, err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("Switched to branch '%s'", a.Branch)}, nil

	case "current_branch":
		out, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("git current-branch: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: out}, nil

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

// RegisterGit registers the git tool with the given registry.
func RegisterGit(registry *tools.Registry) {
	registry.Register(&GitTool{})
}
