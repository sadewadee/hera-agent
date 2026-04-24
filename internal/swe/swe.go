package swe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// ErrIterationLimitReached is returned when the engine exhausts all iterations
// without the test suite passing.
var ErrIterationLimitReached = errors.New("swe: iteration limit reached without passing tests")

// AgentRunner is the interface for LLM reasoning.
// It wraps agent.Agent.HandleMessage so Engine is testable without a real LLM.
type AgentRunner interface {
	HandleMessage(ctx context.Context, platform, chatID, userID, text string) (string, error)
}

// Config holds all configuration for the SWE engine.
type Config struct {
	// Task is the natural-language description of what to implement.
	Task string
	// RepoPath is the absolute path to the git repository root.
	RepoPath string
	// Branch is the git branch to create/checkout for changes.
	Branch string
	// BaseBranch is the base branch to branch from (default: "main").
	BaseBranch string
	// TestCmd is the shell command used to verify correctness (e.g. "go test ./...").
	TestCmd string
	// MaxIterations is the maximum number of LLM+tool iterations (default: 10).
	MaxIterations int
	// DryRun, if true, prints proposed actions without applying patches.
	DryRun bool
}

// Engine is the SWE agent engine. It owns the iteration loop, tool registry,
// test runner, and git operations. Construct via NewEngine.
type Engine struct {
	cfg        Config
	agent      AgentRunner
	controller *IterationController
	gitOps     GitOperator
	tddRunner  TDDRunner
	// lastTestOutput is carried forward to the next iteration's prompt.
	lastTestOutput string
}

// NewEngine constructs a new Engine.
// Returns an error if cfg.Task is empty, cfg.RepoPath is not a git repo,
// or cfg.TestCmd is empty.
func NewEngine(cfg Config, ar AgentRunner, gitOps GitOperator, tdd TDDRunner) (*Engine, error) {
	if strings.TrimSpace(cfg.Task) == "" {
		return nil, fmt.Errorf("swe: task description is required")
	}
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return nil, fmt.Errorf("swe: repo path is required")
	}
	if !gitOps.IsGitRepo(cfg.RepoPath) {
		return nil, fmt.Errorf("swe: %q is not a git repository", cfg.RepoPath)
	}
	if strings.TrimSpace(cfg.TestCmd) == "" {
		return nil, fmt.Errorf("swe: test command is required")
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = "main"
	}
	return &Engine{
		cfg:        cfg,
		agent:      ar,
		controller: NewIterationController(cfg.MaxIterations),
		gitOps:     gitOps,
		tddRunner:  tdd,
	}, nil
}

// Run executes the SWE loop.
//
// It:
//  1. Creates/checks out cfg.Branch from cfg.BaseBranch.
//  2. Iterates up to MaxIterations times:
//     a. Builds a prompt describing the task + current iteration + last test output.
//     b. Calls AgentRunner.HandleMessage so the LLM can use tools to make changes.
//     c. Runs the test suite via TDDRunner.
//     d. On PASS: commits changes (unless DryRun) and returns nil.
//     e. On FAIL: carries the failure output to the next iteration.
//  3. Returns ErrIterationLimitReached if tests never passed.
func (e *Engine) Run(ctx context.Context) error {
	// Setup: create or checkout the branch.
	if e.cfg.Branch != "" {
		if err := e.gitOps.CreateBranch(ctx, e.cfg.RepoPath, e.cfg.Branch, e.cfg.BaseBranch); err != nil {
			return fmt.Errorf("swe: setup branch: %w", err)
		}
		slog.Info("swe: on branch", "branch", e.cfg.Branch)
	}

	for e.controller.Next() {
		iter := e.controller.Current()
		max := e.cfg.MaxIterations
		if max < 1 {
			max = 10
		}

		slog.Info("swe: iteration", "n", iter, "max", max)

		prompt := e.buildPrompt(iter, max)

		if e.cfg.DryRun {
			fmt.Printf("[dry-run] iteration %d/%d prompt:\n%s\n", iter, max, prompt)
		}

		// Ask the LLM to make changes via tools.
		resp, err := e.agent.HandleMessage(ctx, "swe", e.cfg.RepoPath, "swe", prompt)
		if err != nil {
			return fmt.Errorf("swe: iteration %d: agent: %w", iter, err)
		}
		slog.Debug("swe: agent response", "iteration", iter, "response_len", len(resp))

		if e.cfg.DryRun {
			fmt.Printf("[dry-run] iteration %d agent response:\n%s\n", iter, resp)
			// In dry-run mode, don't run tests or commit; just continue to show all iterations.
			continue
		}

		// Run tests to check if the changes are correct.
		result := e.tddRunner.RunTests(ctx)
		slog.Info("swe: test run", "iteration", iter, "passed", result.Passed, "duration", result.Duration)

		if result.Passed {
			// Commit the passing changes.
			msg := fmt.Sprintf("swe: iter %d: %s", iter, truncate(e.cfg.Task, 60))
			if commitErr := e.gitOps.CommitChanges(ctx, e.cfg.RepoPath, msg); commitErr != nil {
				slog.Warn("swe: commit failed (changes not committed)", "error", commitErr)
			}
			slog.Info("swe: tests passed", "iteration", iter)
			return nil
		}

		// Tests failed — feed the output into the next iteration's prompt.
		e.lastTestOutput = result.Output
		slog.Info("swe: tests failed, continuing", "iteration", iter, "output_len", len(result.Output))
	}

	return ErrIterationLimitReached
}

// buildPrompt constructs the per-iteration LLM prompt.
func (e *Engine) buildPrompt(iter, max int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are a software engineering agent. Iteration %d of %d.\n\n", iter, max))
	sb.WriteString("## Task\n\n")
	sb.WriteString(e.cfg.Task)
	sb.WriteString("\n\n")
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("Use the available tools (file_read, file_write, patch, run_command, code_exec, git) ")
	sb.WriteString("to implement the task. Make targeted, minimal changes. ")
	sb.WriteString("After making changes, DO NOT run the test command yourself — the test runner will run automatically.\n\n")

	if e.lastTestOutput != "" {
		sb.WriteString("## Previous test output (FAILED — fix these issues)\n\n```\n")
		sb.WriteString(e.lastTestOutput)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("## Repository\n\n")
	sb.WriteString(fmt.Sprintf("Working directory: %s\n", e.cfg.RepoPath))
	sb.WriteString(fmt.Sprintf("Test command: %s\n\n", e.cfg.TestCmd))
	sb.WriteString("When you have finished making changes for this iteration, respond with: ITERATION_COMPLETE\n")
	return sb.String()
}

// truncate returns s truncated to n runes with an ellipsis if needed.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
