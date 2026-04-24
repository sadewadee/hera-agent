package swe

import (
	"context"
	"errors"
	"testing"
)

// mockAgent implements AgentRunner for tests.
type mockAgent struct {
	responses []string
	calls     int
	err       error
}

func (m *mockAgent) HandleMessage(_ context.Context, _, _, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.calls < len(m.responses) {
		r := m.responses[m.calls]
		m.calls++
		return r, nil
	}
	m.calls++
	return "ITERATION_COMPLETE", nil
}

// mockGit implements GitOperator for tests.
type mockGit struct {
	isRepo      bool
	commitCalls int
	createCalls int
}

func (g *mockGit) IsGitRepo(_ string) bool { return g.isRepo }
func (g *mockGit) CreateBranch(_ context.Context, _, _, _ string) error {
	g.createCalls++
	return nil
}
func (g *mockGit) CommitChanges(_ context.Context, _, _ string) error {
	g.commitCalls++
	return nil
}
func (g *mockGit) CurrentBranch(_ context.Context, _ string) (string, error) { return "main", nil }
func (g *mockGit) WorkdirClean(_ context.Context, _ string) (bool, error)    { return false, nil }

// mockTDD implements TDDRunner for tests.
type mockTDD struct {
	results []TestResult
	call    int
}

func (m *mockTDD) RunTests(_ context.Context) TestResult {
	if m.call < len(m.results) {
		r := m.results[m.call]
		m.call++
		return r
	}
	return TestResult{Passed: false, Output: "no more results"}
}

func makeEngine(t *testing.T, cfg Config, ag AgentRunner, git GitOperator, tdd TDDRunner) *Engine {
	t.Helper()
	e, err := NewEngine(cfg, ag, git, tdd)
	if err != nil {
		t.Fatalf("NewEngine error: %v", err)
	}
	return e
}

func TestNewEngine_RequiresTask(t *testing.T) {
	g := &mockGit{isRepo: true}
	_, err := NewEngine(Config{RepoPath: "/x", TestCmd: "true"}, &mockAgent{}, g, &mockTDD{})
	if err == nil {
		t.Fatal("expected error for empty task")
	}
}

func TestNewEngine_RequiresGitRepo(t *testing.T) {
	g := &mockGit{isRepo: false}
	_, err := NewEngine(
		Config{Task: "fix it", RepoPath: "/x", TestCmd: "true"},
		&mockAgent{}, g, &mockTDD{},
	)
	if err == nil {
		t.Fatal("expected error for non-git repo")
	}
}

func TestNewEngine_RequiresTestCmd(t *testing.T) {
	g := &mockGit{isRepo: true}
	_, err := NewEngine(
		Config{Task: "fix it", RepoPath: "/x"},
		&mockAgent{}, g, &mockTDD{},
	)
	if err == nil {
		t.Fatal("expected error for empty test command")
	}
}

func TestEngine_StopsOnTestPass(t *testing.T) {
	git := &mockGit{isRepo: true}
	tdd := &mockTDD{results: []TestResult{{Passed: true}}}
	e := makeEngine(t, Config{
		Task: "add a feature", RepoPath: "/repo", TestCmd: "go test ./...",
		Branch: "ai/feature", MaxIterations: 5,
	}, &mockAgent{}, git, tdd)

	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if git.commitCalls != 1 {
		t.Errorf("commitCalls = %d, want 1", git.commitCalls)
	}
}

func TestEngine_IterationLimitReached(t *testing.T) {
	git := &mockGit{isRepo: true}
	tdd := &mockTDD{results: []TestResult{
		{Passed: false, Output: "FAIL"},
		{Passed: false, Output: "FAIL"},
	}}
	e := makeEngine(t, Config{
		Task: "add a feature", RepoPath: "/repo", TestCmd: "go test ./...",
		MaxIterations: 2,
	}, &mockAgent{}, git, tdd)

	err := e.Run(context.Background())
	if !errors.Is(err, ErrIterationLimitReached) {
		t.Errorf("Run() error = %v, want ErrIterationLimitReached", err)
	}
	if git.commitCalls != 0 {
		t.Errorf("commitCalls = %d, want 0", git.commitCalls)
	}
}

func TestEngine_DryRun(t *testing.T) {
	git := &mockGit{isRepo: true}
	tdd := &mockTDD{results: []TestResult{{Passed: true}}}
	e := makeEngine(t, Config{
		Task: "add a feature", RepoPath: "/repo", TestCmd: "go test ./...",
		MaxIterations: 3, DryRun: true,
	}, &mockAgent{}, git, tdd)

	// DryRun should exhaust iterations without committing or running tests.
	err := e.Run(context.Background())
	if !errors.Is(err, ErrIterationLimitReached) {
		t.Errorf("DryRun: Run() error = %v, want ErrIterationLimitReached", err)
	}
	if git.commitCalls != 0 {
		t.Errorf("DryRun: commitCalls = %d, want 0", git.commitCalls)
	}
	if tdd.call != 0 {
		t.Errorf("DryRun: tdd.call = %d, want 0", tdd.call)
	}
}

func TestEngine_PassesOnSecondIteration(t *testing.T) {
	git := &mockGit{isRepo: true}
	tdd := &mockTDD{results: []TestResult{
		{Passed: false, Output: "expected foo but got bar"},
		{Passed: true},
	}}
	e := makeEngine(t, Config{
		Task: "fix foo", RepoPath: "/repo", TestCmd: "go test ./...",
		MaxIterations: 5,
	}, &mockAgent{}, git, tdd)

	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if git.commitCalls != 1 {
		t.Errorf("commitCalls = %d, want 1", git.commitCalls)
	}
}

func TestEngine_AgentError(t *testing.T) {
	git := &mockGit{isRepo: true}
	tdd := &mockTDD{}
	ag := &mockAgent{err: errors.New("LLM unavailable")}
	e := makeEngine(t, Config{
		Task: "fix it", RepoPath: "/repo", TestCmd: "go test ./...",
		MaxIterations: 3,
	}, ag, git, tdd)

	err := e.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when agent fails")
	}
}

func TestEngine_CreatesBranch(t *testing.T) {
	git := &mockGit{isRepo: true}
	tdd := &mockTDD{results: []TestResult{{Passed: true}}}
	e := makeEngine(t, Config{
		Task: "fix it", RepoPath: "/repo", TestCmd: "go test ./...",
		Branch: "feature/x", MaxIterations: 3,
	}, &mockAgent{}, git, tdd)

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if git.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", git.createCalls)
	}
}
