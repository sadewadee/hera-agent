package swe

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePRContent_Basic(t *testing.T) {
	task := "Fix the nil pointer dereference in foo.go"
	diffSum := "Modified foo.go: added nil guard in HandleRequest"
	title, body := GeneratePRContent(task, diffSum, 3)
	assert.Equal(t, "Fix the nil pointer dereference in foo.go", title)
	assert.Contains(t, body, task)
	assert.Contains(t, body, diffSum)
	assert.Contains(t, body, "TDD iterations: 3")
}

func TestGeneratePRContent_LongTask(t *testing.T) {
	task := "Implement a comprehensive caching layer with LRU eviction, TTL support, and optional Redis backend for the hot path queries in the user service"
	title, body := GeneratePRContent(task, "", 1)
	assert.LessOrEqual(t, len(title), 72, "title must be <= 72 chars")
	assert.Contains(t, body, "hera-swe")
}

func TestGeneratePRContent_MultiSentence(t *testing.T) {
	task := "Fix broken auth. Add more tests. Deploy."
	title, body := GeneratePRContent(task, "", 2)
	assert.Equal(t, "Fix broken auth", title, "title should be first sentence without period")
	assert.Contains(t, body, task)
}

func TestGeneratePRContent_NoDiffSummary(t *testing.T) {
	title, body := GeneratePRContent("Simple task", "", 0)
	assert.Equal(t, "Simple task", title)
	assert.NotContains(t, body, "## Changes")
}

func TestBuildTitle_StripsTrailingPunctuation(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Fix it.", "Fix it"},
		{"Fix it!", "Fix it"},
		{"Fix it?", "Fix it"},
		{"Fix it", "Fix it"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, buildTitle(c.in), "input: %q", c.in)
	}
}

func TestBuildTitle_NewlineSplit(t *testing.T) {
	task := "Fix auth\nMore context here"
	title := buildTitle(task)
	assert.Equal(t, "Fix auth", title)
}

// TestCreatePR_GhAbsent verifies an explicit error is returned when gh is missing.
func TestCreatePR_GhAbsent(t *testing.T) {
	// Manipulate PATH so gh cannot be found.
	t.Setenv("PATH", t.TempDir()) // empty dir, no binaries
	err := CreatePR("test title", "test body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gh CLI not found")
}

// TestCreatePR_GhFakeSuccess tests PR creation with a fake gh binary.
func TestCreatePR_GhFakeSuccess(t *testing.T) {
	binDir := t.TempDir()
	ghFake := filepath.Join(binDir, "gh")
	// Write a tiny shell script that simulates a successful gh run.
	script := "#!/bin/sh\necho 'https://github.com/owner/repo/pull/1'\nexit 0\n"
	require.NoError(t, os.WriteFile(ghFake, []byte(script), 0o755))
	// Prepend the fake binary dir to PATH.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)
	// Verify gh is found now.
	_, err := exec.LookPath("gh")
	require.NoError(t, err, "fake gh must be discoverable")
	err = CreatePR("feat: add widget", "## Summary\n\nAdded widget.")
	assert.NoError(t, err)
}

// TestCreatePR_GhFakeFailure tests that gh non-zero exit is reported.
func TestCreatePR_GhFakeFailure(t *testing.T) {
	binDir := t.TempDir()
	ghFake := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\necho 'fatal: not a git repository' >&2\nexit 128\n"
	require.NoError(t, os.WriteFile(ghFake, []byte(script), 0o755))
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)
	err := CreatePR("title", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gh pr create failed")
}
