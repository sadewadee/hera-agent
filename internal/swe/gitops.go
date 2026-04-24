package swe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitOperator is the interface for git operations needed by the SWE engine.
// Defined as an interface so tests can inject a mock.
type GitOperator interface {
	// IsGitRepo returns true if path is inside a git repository.
	IsGitRepo(path string) bool
	// CreateBranch creates or resets branch from base in repoPath.
	// If branch already exists it is checked out (not reset).
	CreateBranch(ctx context.Context, repoPath, branch, base string) error
	// CommitChanges stages all modified/added files and creates a commit.
	// Does nothing (returns nil) if the working tree is clean.
	CommitChanges(ctx context.Context, repoPath, message string) error
	// CurrentBranch returns the name of the currently checked-out branch.
	CurrentBranch(ctx context.Context, repoPath string) (string, error)
	// WorkdirClean returns true if there are no uncommitted changes.
	WorkdirClean(ctx context.Context, repoPath string) (bool, error)
}

// ExecGitOperator implements GitOperator via os/exec calls to the git binary.
// No git push is ever issued — only local operations.
type ExecGitOperator struct{}

func (g *ExecGitOperator) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsGitRepo returns true if path is within a git repository.
func (g *ExecGitOperator) IsGitRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	return cmd.Run() == nil
}

// CreateBranch checks out an existing branch or creates a new one from base.
// If branch already exists locally it is simply checked out.
func (g *ExecGitOperator) CreateBranch(ctx context.Context, repoPath, branch, base string) error {
	// Try to check out an existing branch first.
	_, err := g.run(ctx, repoPath, "checkout", branch)
	if err == nil {
		return nil
	}
	// Branch doesn't exist — create it from base.
	if base == "" {
		base = "HEAD"
	}
	if _, err := g.run(ctx, repoPath, "checkout", "-b", branch, base); err != nil {
		return fmt.Errorf("create branch %q from %q: %w", branch, base, err)
	}
	return nil
}

// CommitChanges stages all changes under repoPath and commits with message.
// Returns nil without committing if the working directory is already clean.
func (g *ExecGitOperator) CommitChanges(ctx context.Context, repoPath, message string) error {
	clean, err := g.WorkdirClean(ctx, repoPath)
	if err != nil {
		return err
	}
	if clean {
		return nil
	}
	if _, err := g.run(ctx, repoPath, "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if _, err := g.run(ctx, repoPath, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// CurrentBranch returns the active branch name.
func (g *ExecGitOperator) CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	out, err := g.run(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

// WorkdirClean returns true when there are no staged or unstaged changes.
func (g *ExecGitOperator) WorkdirClean(ctx context.Context, repoPath string) (bool, error) {
	out, err := g.run(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}
