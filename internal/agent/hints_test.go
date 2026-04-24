package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSubdirectoryHints_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Empty(t, hints)
}

func TestLoadSubdirectoryHints_FindsHeraFile(t *testing.T) {
	dir := t.TempDir()
	content := "# Project Hints\nUse Go 1.22"
	err := os.WriteFile(filepath.Join(dir, ".hera.md"), []byte(content), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Contains(t, hints, "Project Hints")
	assert.Contains(t, hints, "Use Go 1.22")
}

func TestLoadSubdirectoryHints_FindsClaudeMD(t *testing.T) {
	dir := t.TempDir()
	content := "# CLAUDE.md conventions"
	err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(content), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Contains(t, hints, "CLAUDE.md conventions")
}

func TestLoadSubdirectoryHints_FindsAgentsMD(t *testing.T) {
	dir := t.TempDir()
	content := "# Agent instructions"
	err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(content), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Contains(t, hints, "Agent instructions")
}

func TestLoadSubdirectoryHints_SkipsEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".hera.md"), []byte("   \n  "), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Empty(t, hints)
}

func TestLoadSubdirectoryHints_CombinesMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".hera.md"), []byte("Hera hints"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("Claude hints"), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Contains(t, hints, "Hera hints")
	assert.Contains(t, hints, "Claude hints")
}

func TestLoadSubdirectoryHints_WalksParentDirectories(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.MkdirAll(child, 0755))

	err := os.WriteFile(filepath.Join(parent, ".hera.md"), []byte("Parent hints"), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(child)
	require.NoError(t, err)
	assert.Contains(t, hints, "Parent hints")
}

func TestLoadSubdirectoryHints_IncludesSourcePath(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".hera.md"), []byte("Some hints"), 0644)
	require.NoError(t, err)

	hints, err := LoadSubdirectoryHints(dir)
	require.NoError(t, err)
	assert.Contains(t, hints, "<!-- from ")
}

func TestLoadSubdirectoryHints_DefaultsToWorkingDir(t *testing.T) {
	// When empty string is passed, it uses os.Getwd()
	hints, err := LoadSubdirectoryHints("")
	require.NoError(t, err)
	// We cannot assert content since it depends on the test environment,
	// but it should not error.
	_ = hints
}
