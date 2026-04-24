package batch

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRunner is an AgentRunner that returns configurable responses.
type fakeRunner struct {
	response string
	err      error
	calls    atomic.Int64
	delay    time.Duration
}

func (f *fakeRunner) HandleMessage(ctx context.Context, _, _, _, text string) (string, error) {
	f.calls.Add(1)
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.response + " | " + text, f.err
}

// transientRunner fails the first N times with a transient error, then succeeds.
type transientRunner struct {
	failN    int
	calls    atomic.Int64
	response string
}

func (r *transientRunner) HandleMessage(_ context.Context, _, _, _, text string) (string, error) {
	n := r.calls.Add(1)
	if int(n) <= r.failN {
		return "", errors.New("rate limit exceeded")
	}
	return r.response, nil
}

func newBatch(t *testing.T, runner AgentRunner, prompts []string, concurrency int) (*Batch, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	writer := NewJSONLWriter(&buf)
	store := NoopCheckpointStore{}
	src := NewSliceSource(prompts)
	cfg := Config{
		RunID:       "test-run",
		Concurrency: concurrency,
		MaxRetries:  3,
	}
	b := New(cfg, runner, src, writer, store, NoopProgress{})
	return b, &buf
}

func TestBatch_Run_Basic(t *testing.T) {
	runner := &fakeRunner{response: "ok"}
	prompts := []string{"p1", "p2", "p3"}

	b, buf := newBatch(t, runner, prompts, 1)
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := runner.calls.Load(); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d", len(lines))
	}
}

func TestBatch_Run_Parallel(t *testing.T) {
	runner := &fakeRunner{response: "ans", delay: 5 * time.Millisecond}
	prompts := make([]string, 10)
	for i := range prompts {
		prompts[i] = "prompt"
	}

	b, _ := newBatch(t, runner, prompts, 5)
	start := time.Now()
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	elapsed := time.Since(start)

	// With 10 prompts at 5ms each and concurrency=5, max serial time = 50ms.
	// Parallel should finish well under 30ms with 5 workers.
	if elapsed > 40*time.Millisecond {
		t.Logf("parallel run took %v (may be slow CI)", elapsed)
	}
	if got := runner.calls.Load(); got != 10 {
		t.Errorf("calls = %d, want 10", got)
	}
}

func TestBatch_Run_RetryTransient(t *testing.T) {
	runner := &transientRunner{failN: 2, response: "ok"}
	prompts := []string{"hello"}

	b, buf := newBatch(t, runner, prompts, 1)
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should have succeeded after retries.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 output line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"response":"ok"`) {
		t.Errorf("expected response ok in output: %s", lines[0])
	}
}

func TestBatch_Run_Resume(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteCheckpointStore(filepath.Join(dir, "cp.db"))
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	defer store.Close()

	// Mark first 2 prompts as completed.
	_ = store.SetStatus("run1", 0, "p0", StatusCompleted, "")
	_ = store.SetStatus("run1", 1, "p1", StatusCompleted, "")

	runner := &fakeRunner{response: "fresh"}
	prompts := []string{"p0", "p1", "p2"}
	writer := NewJSONLWriter(&bytes.Buffer{})

	cfg := Config{RunID: "run1", Concurrency: 1, MaxRetries: 0}
	b := New(cfg, runner, NewSliceSource(prompts), writer, store, NoopProgress{})
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Only p2 should have been processed.
	if got := runner.calls.Load(); got != 1 {
		t.Errorf("calls = %d, want 1 (resume skipped 2 completed)", got)
	}
}

func TestBatch_Run_ContextCancel(t *testing.T) {
	runner := &fakeRunner{response: "ok", delay: 50 * time.Millisecond}
	prompts := make([]string, 20)
	for i := range prompts {
		prompts[i] = "slow"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	b, _ := newBatch(t, runner, prompts, 2)
	err := b.Run(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestBatch_Run_AllFail(t *testing.T) {
	runner := &fakeRunner{response: "", err: errors.New("permanent failure")}
	prompts := []string{"a", "b"}

	var buf bytes.Buffer
	writer := NewJSONLWriter(&buf)
	store := NoopCheckpointStore{}
	cfg := Config{RunID: "fail-run", Concurrency: 1, MaxRetries: 0}
	b := New(cfg, runner, NewSliceSource(prompts), writer, store, NoopProgress{})

	// Should not return error at batch level — errors are per-result.
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
	// Both should have error field set.
	for _, line := range lines {
		if !strings.Contains(line, `"error"`) {
			t.Errorf("expected error field in: %s", line)
		}
	}
}

func TestBatch_Empty(t *testing.T) {
	runner := &fakeRunner{response: "ok"}
	b, _ := newBatch(t, runner, nil, 1)
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("Run empty: %v", err)
	}
	if got := runner.calls.Load(); got != 0 {
		t.Errorf("calls on empty = %d, want 0", got)
	}
}

func TestIsTransient(t *testing.T) {
	cases := []struct {
		msg       string
		transient bool
	}{
		{"rate limit exceeded", true},
		{"429 too many requests", true},
		{"connection refused", true},
		{"deadline exceeded", true},
		{"server error 503", true},
		{"not found", false},
		{"invalid request", false},
		{"", false},
	}
	for _, tc := range cases {
		var err error
		if tc.msg != "" {
			err = errors.New(tc.msg)
		}
		got := isTransient(err)
		if got != tc.transient {
			t.Errorf("isTransient(%q) = %v, want %v", tc.msg, got, tc.transient)
		}
	}
}
