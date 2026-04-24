package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasTraversalComponent_WithDotDot(t *testing.T) {
	assert.True(t, HasTraversalComponent("../etc/passwd"))
	assert.True(t, HasTraversalComponent("/safe/../../etc/passwd"))
	assert.True(t, HasTraversalComponent("a/b/../../../secret"))
}

func TestHasTraversalComponent_Safe(t *testing.T) {
	assert.False(t, HasTraversalComponent("/safe/path/to/file.txt"))
	assert.False(t, HasTraversalComponent("relative/path/file.go"))
	assert.False(t, HasTraversalComponent(""))
}

func TestHasTraversalComponent_SingleDot(t *testing.T) {
	// Single dots are not traversal.
	assert.False(t, HasTraversalComponent("./path/to/file"))
}

func TestValidateWithinDir_SafePath(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child.txt")
	// Create the file so EvalSymlinks can resolve it.
	if err := os.WriteFile(child, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ValidateWithinDir(child, dir)
	assert.Empty(t, result, "expected no error for safe path")
}

func TestValidateWithinDir_RootItself(t *testing.T) {
	dir := t.TempDir()
	result := ValidateWithinDir(dir, dir)
	assert.Empty(t, result, "root itself should be valid")
}

func TestValidateWithinDir_EscapeAttempt(t *testing.T) {
	dir := t.TempDir()
	escapePath := filepath.Join(dir, "..", "outside")
	result := ValidateWithinDir(escapePath, dir)
	// Should return an error string because the path escapes root.
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "escapes allowed directory")
}

func TestValidateWithinDir_NestedSafe(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	result := ValidateWithinDir(nested, dir)
	assert.Empty(t, result)
}

func TestValidateWithinDir_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var) so both paths match.
	dir, _ = filepath.EvalSymlinks(dir)
	nonExistent := filepath.Join(dir, "does-not-exist.txt")
	result := ValidateWithinDir(nonExistent, dir)
	assert.Empty(t, result)
}
