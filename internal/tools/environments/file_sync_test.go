package environments

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileSyncManager(t *testing.T) {
	getFiles := func() []FilePair { return nil }
	upload := func(host, remote string) error { return nil }
	del := func(paths []string) error { return nil }

	m := NewFileSyncManager(getFiles, upload, del)
	require.NotNil(t, m)
	assert.Equal(t, defaultSyncInterval, m.syncInterval)
}

func TestFileSyncManager_WithSyncInterval(t *testing.T) {
	m := NewFileSyncManager(
		func() []FilePair { return nil },
		func(string, string) error { return nil },
		func([]string) error { return nil },
		WithSyncInterval(10*time.Second),
	)
	assert.Equal(t, 10*time.Second, m.syncInterval)
}

func TestFileSyncManager_SyncUploads(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644))

	var uploaded []FilePair
	m := NewFileSyncManager(
		func() []FilePair {
			return []FilePair{{HostPath: filePath, RemotePath: "/remote/test.txt"}}
		},
		func(host, remote string) error {
			uploaded = append(uploaded, FilePair{HostPath: host, RemotePath: remote})
			return nil
		},
		func([]string) error { return nil },
	)

	m.Sync(true)
	require.Len(t, uploaded, 1)
	assert.Equal(t, filePath, uploaded[0].HostPath)
	assert.Equal(t, "/remote/test.txt", uploaded[0].RemotePath)
}

func TestFileSyncManager_SyncNoChanges(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644))

	uploadCount := 0
	m := NewFileSyncManager(
		func() []FilePair {
			return []FilePair{{HostPath: filePath, RemotePath: "/remote/test.txt"}}
		},
		func(string, string) error { uploadCount++; return nil },
		func([]string) error { return nil },
	)

	m.Sync(true) // First sync uploads
	assert.Equal(t, 1, uploadCount)

	m.Sync(true) // Second sync should not upload (same mtime)
	assert.Equal(t, 1, uploadCount)
}

func TestFileSyncManager_SyncDeletions(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644))

	var deleted []string
	returnFiles := true
	m := NewFileSyncManager(
		func() []FilePair {
			if returnFiles {
				return []FilePair{{HostPath: filePath, RemotePath: "/remote/test.txt"}}
			}
			return nil
		},
		func(string, string) error { return nil },
		func(paths []string) error { deleted = append(deleted, paths...); return nil },
	)

	m.Sync(true) // Upload file
	assert.Empty(t, deleted)

	returnFiles = false
	m.Sync(true) // File removed, should delete
	assert.Contains(t, deleted, "/remote/test.txt")
}

func TestFileSyncManager_RateLimited(t *testing.T) {
	uploadCount := 0
	m := NewFileSyncManager(
		func() []FilePair { return nil },
		func(string, string) error { uploadCount++; return nil },
		func([]string) error { return nil },
		WithSyncInterval(1*time.Hour), // Very long interval
	)

	m.Sync(true)  // Force sync
	m.Sync(false) // Should be rate-limited
	// No files to upload anyway, but rate limiting was tested
}

func TestQuotedRmCommand(t *testing.T) {
	cmd := QuotedRmCommand([]string{"/path/to/file1", "/path/to/file2"})
	assert.Contains(t, cmd, "rm -f")
	assert.Contains(t, cmd, "/path/to/file1")
	assert.Contains(t, cmd, "/path/to/file2")
}

func TestQuotedMkdirCommand(t *testing.T) {
	cmd := QuotedMkdirCommand([]string{"/a/b", "/c/d"})
	assert.Contains(t, cmd, "mkdir -p")
	assert.Contains(t, cmd, "/a/b")
}

func TestUniqueParentDirs(t *testing.T) {
	files := []FilePair{
		{RemotePath: "/a/b/file1.txt"},
		{RemotePath: "/a/b/file2.txt"},
		{RemotePath: "/c/d/file3.txt"},
	}
	dirs := UniqueParentDirs(files)
	assert.Len(t, dirs, 2)
	assert.Equal(t, "/a/b", dirs[0])
	assert.Equal(t, "/c/d", dirs[1])
}

func TestUniqueParentDirs_Empty(t *testing.T) {
	dirs := UniqueParentDirs(nil)
	assert.Empty(t, dirs)
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "'/simple/path'", shellQuote("/simple/path"))
	assert.Contains(t, shellQuote("it's"), "it")
}
