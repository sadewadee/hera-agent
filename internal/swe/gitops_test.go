package swe

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	// Create an initial commit so HEAD exists.
	f := filepath.Join(dir, "README.md")
	if err := os.WriteFile(f, []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")
	return dir
}

func TestExecGitOperator_IsGitRepo(t *testing.T) {
	g := &ExecGitOperator{}
	repo := initTestRepo(t)
	if !g.IsGitRepo(repo) {
		t.Errorf("IsGitRepo(%q) = false, want true", repo)
	}
	if g.IsGitRepo(t.TempDir()) {
		t.Error("IsGitRepo(tmpDir) = true for non-repo, want false")
	}
}

func TestExecGitOperator_CreateBranch(t *testing.T) {
	g := &ExecGitOperator{}
	repo := initTestRepo(t)
	ctx := context.Background()

	if err := g.CreateBranch(ctx, repo, "feature/test", "main"); err != nil {
		t.Fatalf("CreateBranch error: %v", err)
	}
	branch, err := g.CurrentBranch(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature/test" {
		t.Errorf("branch = %q, want %q", branch, "feature/test")
	}

	// Creating same branch again should succeed (checkout existing).
	if err := g.CreateBranch(ctx, repo, "feature/test", "main"); err != nil {
		t.Fatalf("CreateBranch (existing) error: %v", err)
	}
}

func TestExecGitOperator_CommitChanges(t *testing.T) {
	g := &ExecGitOperator{}
	repo := initTestRepo(t)
	ctx := context.Background()

	// Clean repo — commit should be no-op.
	if err := g.CommitChanges(ctx, repo, "should not commit"); err != nil {
		t.Fatalf("CommitChanges on clean repo error: %v", err)
	}

	// Write a file and commit.
	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := g.CommitChanges(ctx, repo, "add new.txt"); err != nil {
		t.Fatalf("CommitChanges error: %v", err)
	}

	clean, err := g.WorkdirClean(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Error("repo should be clean after CommitChanges")
	}
}

func TestExecGitOperator_WorkdirClean(t *testing.T) {
	g := &ExecGitOperator{}
	repo := initTestRepo(t)
	ctx := context.Background()

	clean, err := g.WorkdirClean(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Error("fresh repo should be clean")
	}

	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clean, err = g.WorkdirClean(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if clean {
		t.Error("repo with untracked file should not be clean")
	}
}
