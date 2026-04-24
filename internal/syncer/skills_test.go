package syncer_test

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/syncer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupBundled creates a temporary bundled skills directory with some .md files.
func setupBundled(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	return dir
}

// readFile is a test helper to read a file and return its content.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

// md5sum returns the hex MD5 of a string.
func md5sum(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// TestSync_FreshInstall verifies a fresh sync copies all bundled skills.
func TestSync_FreshInstall(t *testing.T) {
	bundledDir := setupBundled(t, map[string]string{
		"general/hello.md":  "# Hello\nThis is a skill.",
		"general/world.md":  "# World\nAnother skill.",
		"optional/fancy.md": "# Fancy\nOptional skill.",
	})
	userDir := t.TempDir()

	s := syncer.New(bundledDir, userDir)
	stats, err := s.Sync()
	require.NoError(t, err)

	assert.Equal(t, 3, stats.Copied, "all 3 files should be copied")
	assert.Equal(t, 0, stats.Preserved, "nothing to preserve on fresh install")
	assert.Equal(t, 0, stats.Skipped, "no skips on fresh install")

	assert.Equal(t, "# Hello\nThis is a skill.", readFile(t, filepath.Join(userDir, "general/hello.md")))
	assert.Equal(t, "# World\nAnother skill.", readFile(t, filepath.Join(userDir, "general/world.md")))
	assert.Equal(t, "# Fancy\nOptional skill.", readFile(t, filepath.Join(userDir, "optional/fancy.md")))

	// Manifest must exist.
	manifestPath := filepath.Join(userDir, ".bundled_manifest")
	_, err = os.Stat(manifestPath)
	require.NoError(t, err, "manifest file must be created")
}

// TestSync_PreservesUserEdit verifies that a user-modified skill is preserved
// on subsequent syncs (no overwrite), and the stats.Preserved count is correct.
func TestSync_PreservesUserEdit(t *testing.T) {
	bundledContent := "# Hello\nOriginal content."
	bundledDir := setupBundled(t, map[string]string{
		"general/hello.md": bundledContent,
	})
	userDir := t.TempDir()

	// First sync: fresh install.
	s := syncer.New(bundledDir, userDir)
	_, err := s.Sync()
	require.NoError(t, err)

	// User edits the skill.
	userFile := filepath.Join(userDir, "general/hello.md")
	userEdit := "# Hello\nUser-modified content — do not overwrite!"
	require.NoError(t, os.WriteFile(userFile, []byte(userEdit), 0o644))

	// Second sync: same bundled content.
	stats, err := s.Sync()
	require.NoError(t, err)

	assert.Equal(t, 0, stats.Copied, "user-modified file must not be overwritten")
	assert.Equal(t, 1, stats.Preserved, "user edit must be counted as preserved")

	// File content must still be the user's version.
	assert.Equal(t, userEdit, readFile(t, userFile))
}

// TestSync_PropagatesUpstreamUpdate verifies that an unmodified skill is
// overwritten when the bundled version changes.
func TestSync_PropagatesUpstreamUpdate(t *testing.T) {
	originalContent := "# Hello\nOriginal bundled content."
	bundledDir := setupBundled(t, map[string]string{
		"general/hello.md": originalContent,
	})
	userDir := t.TempDir()

	s := syncer.New(bundledDir, userDir)
	_, err := s.Sync()
	require.NoError(t, err)

	// Bundled file gets a new version (upstream update).
	updatedContent := "# Hello\nUpdated bundled content."
	require.NoError(t, os.WriteFile(
		filepath.Join(bundledDir, "general/hello.md"),
		[]byte(updatedContent), 0o644))

	// Second sync: should copy the updated bundled content.
	stats, err := s.Sync()
	require.NoError(t, err)

	assert.Equal(t, 1, stats.Copied, "upstream update on unmodified file must be applied")
	assert.Equal(t, 0, stats.Preserved)

	userFile := filepath.Join(userDir, "general/hello.md")
	assert.Equal(t, updatedContent, readFile(t, userFile))
}

// TestSync_MissingManifest verifies that syncing without an existing manifest
// (migration from pre-v0.11.0) treats all existing user files as untracked
// and does NOT overwrite them (treat as user-modified to be safe).
func TestSync_MissingManifest(t *testing.T) {
	bundledContent := "# Hello\nBundled content."
	bundledDir := setupBundled(t, map[string]string{
		"general/hello.md": bundledContent,
	})
	userDir := t.TempDir()

	// Pre-populate user skill dir without using the syncer (simulates old install).
	userFile := filepath.Join(userDir, "general", "hello.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(userFile), 0o755))
	existingContent := "# Hello\nPre-existing user content."
	require.NoError(t, os.WriteFile(userFile, []byte(existingContent), 0o644))

	// Sync without manifest: pre-existing user file must be preserved.
	s := syncer.New(bundledDir, userDir)
	stats, err := s.Sync()
	require.NoError(t, err)

	assert.Equal(t, 0, stats.Copied, "pre-existing files with no manifest are treated as user-modified")
	assert.Equal(t, 1, stats.Preserved)
	assert.Equal(t, existingContent, readFile(t, userFile))
}

// TestSync_ManifestHashFormat verifies that the manifest stores MD5 hashes
// of the BUNDLED content (not user content).
func TestSync_ManifestHashFormat(t *testing.T) {
	content := "# Hello\nContent for hash test."
	bundledDir := setupBundled(t, map[string]string{
		"general/hello.md": content,
	})
	userDir := t.TempDir()

	s := syncer.New(bundledDir, userDir)
	_, err := s.Sync()
	require.NoError(t, err)

	manifestPath := filepath.Join(userDir, ".bundled_manifest")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	manifestContent := string(data)
	expectedHash := md5sum(content)
	assert.True(t, strings.Contains(manifestContent, expectedHash),
		"manifest must contain MD5 hash %q of bundled content; got:\n%s",
		expectedHash, manifestContent)
}

// TestSyncStats_ZeroOnEmptyBundled verifies that syncing an empty bundled dir
// returns zero stats and does not error.
func TestSyncStats_ZeroOnEmptyBundled(t *testing.T) {
	bundledDir := t.TempDir() // empty
	userDir := t.TempDir()

	s := syncer.New(bundledDir, userDir)
	stats, err := s.Sync()
	require.NoError(t, err)

	assert.Equal(t, 0, stats.Copied)
	assert.Equal(t, 0, stats.Preserved)
	assert.Equal(t, 0, stats.Skipped)
}
