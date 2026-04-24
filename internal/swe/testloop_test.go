package swe

import (
	"context"
	"testing"
	"time"
)

func TestShellTDDRunner_Pass(t *testing.T) {
	runner := NewShellTDDRunner("true", t.TempDir(), 5*time.Second)
	res := runner.RunTests(context.Background())
	if !res.Passed {
		t.Errorf("Passed = false for 'true', want true. Output: %s Err: %v", res.Output, res.Err)
	}
}

func TestShellTDDRunner_Fail(t *testing.T) {
	runner := NewShellTDDRunner("false", t.TempDir(), 5*time.Second)
	res := runner.RunTests(context.Background())
	if res.Passed {
		t.Errorf("Passed = true for 'false', want false")
	}
}

func TestShellTDDRunner_CapturesOutput(t *testing.T) {
	runner := NewShellTDDRunner("echo hello", t.TempDir(), 5*time.Second)
	res := runner.RunTests(context.Background())
	if !res.Passed {
		t.Fatalf("expected pass, got fail: %v", res.Err)
	}
	if res.Output == "" {
		t.Error("Output should not be empty for echo command")
	}
}

func TestShellTDDRunner_DefaultTimeout(t *testing.T) {
	runner := NewShellTDDRunner("true", t.TempDir(), 0)
	if runner.timeout != 5*time.Minute {
		t.Errorf("default timeout = %v, want 5m", runner.timeout)
	}
}

func TestShellTDDRunner_ContextCancel(t *testing.T) {
	runner := NewShellTDDRunner("sleep 60", t.TempDir(), 30*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	res := runner.RunTests(ctx)
	if res.Passed {
		t.Error("expected fail after context cancel, got pass")
	}
}
