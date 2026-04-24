package swe

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// TestResult holds the outcome of running the test command.
type TestResult struct {
	Passed   bool
	Output   string // combined stdout+stderr
	Duration time.Duration
	Err      error // set if the command itself failed to run (e.g. binary not found)
}

// TDDRunner is the interface for running tests.
// Defined as an interface so tests can inject a mock without a real LLM or shell.
type TDDRunner interface {
	RunTests(ctx context.Context) TestResult
}

// ShellTDDRunner runs the test command via os/exec sh -c.
type ShellTDDRunner struct {
	cmd     string        // e.g. "go test ./..."
	dir     string        // working directory (repo root)
	timeout time.Duration // deadline per test run; defaults to 5 minutes
}

// NewShellTDDRunner creates a TDD runner with the given command and directory.
// timeout defaults to 5 minutes if zero.
func NewShellTDDRunner(cmd, dir string, timeout time.Duration) *ShellTDDRunner {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &ShellTDDRunner{cmd: cmd, dir: dir, timeout: timeout}
}

// RunTests implements TDDRunner. It runs the configured command, captures
// combined output, and returns Passed=true on exit code 0.
func (r *ShellTDDRunner) RunTests(ctx context.Context) TestResult {
	tctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(tctx, "sh", "-c", r.cmd)
	cmd.Dir = r.dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	var out strings.Builder
	if stdout.Len() > 0 {
		out.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString(stderr.String())
	}

	// Limit output to 4KB to avoid blowing up LLM context.
	const maxOutput = 4 * 1024
	output := out.String()
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... [truncated]"
	}

	passed := runErr == nil
	return TestResult{
		Passed:   passed,
		Output:   output,
		Duration: time.Since(start),
		Err:      runErr,
	}
}
