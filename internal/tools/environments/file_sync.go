// Package environments provides execution environment implementations.
//
// file_sync.go implements a shared file sync manager for remote execution
// backends. Tracks local file changes via mtime+size, detects deletions,
// and syncs to remote environments transactionally. Used by SSH, Modal,
// and Daytona backends. Docker and Singularity use bind mounts and do
// not need this.
package environments

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultSyncInterval = 5 * time.Second
	forceSyncEnv        = "HERA_FORCE_FILE_SYNC"
)

// UploadFunc uploads a single file from hostPath to remotePath.
type UploadFunc func(hostPath, remotePath string) error

// BulkUploadFunc uploads multiple files in one operation.
type BulkUploadFunc func(pairs []FilePair) error

// DeleteFunc deletes remote files.
type DeleteFunc func(remotePaths []string) error

// GetFilesFunc returns the current set of files that should be synced.
type GetFilesFunc func() []FilePair

// FilePair represents a host-path to remote-path mapping.
type FilePair struct {
	HostPath   string
	RemotePath string
}

// fileMtimeKey returns an mtime+size fingerprint for change detection.
// Returns nil if the file cannot be stat'd (deleted or inaccessible).
func fileMtimeKey(hostPath string) *fileKey {
	info, err := os.Stat(hostPath)
	if err != nil {
		return nil
	}
	return &fileKey{
		mtime: info.ModTime().UnixNano(),
		size:  info.Size(),
	}
}

type fileKey struct {
	mtime int64
	size  int64
}

// FileSyncManager tracks local file changes and syncs to a remote
// environment. Backends instantiate this with transport callbacks
// (upload, delete) and a file-source callable. The manager handles
// mtime-based change detection, deletion tracking, rate limiting,
// and transactional state.
type FileSyncManager struct {
	mu           sync.Mutex
	getFilesFn   GetFilesFunc
	uploadFn     UploadFunc
	bulkUploadFn BulkUploadFunc
	deleteFn     DeleteFunc
	syncedFiles  map[string]fileKey // remotePath -> fileKey
	lastSyncTime time.Time
	syncInterval time.Duration
}

// NewFileSyncManager creates a new file sync manager.
func NewFileSyncManager(
	getFiles GetFilesFunc,
	upload UploadFunc,
	deleteFn DeleteFunc,
	opts ...FileSyncOption,
) *FileSyncManager {
	m := &FileSyncManager{
		getFilesFn:   getFiles,
		uploadFn:     upload,
		deleteFn:     deleteFn,
		syncedFiles:  make(map[string]fileKey),
		syncInterval: defaultSyncInterval,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// FileSyncOption configures a FileSyncManager.
type FileSyncOption func(*FileSyncManager)

// WithBulkUpload sets a bulk upload function for efficiency.
func WithBulkUpload(fn BulkUploadFunc) FileSyncOption {
	return func(m *FileSyncManager) { m.bulkUploadFn = fn }
}

// WithSyncInterval overrides the default sync interval.
func WithSyncInterval(d time.Duration) FileSyncOption {
	return func(m *FileSyncManager) { m.syncInterval = d }
}

// Sync runs a sync cycle: upload changed files, delete removed files.
// Rate-limited unless force is true or HERA_FORCE_FILE_SYNC=1.
// Transactional: state only committed if ALL operations succeed.
func (m *FileSyncManager) Sync(force bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !force && os.Getenv(forceSyncEnv) == "" {
		if time.Since(m.lastSyncTime) < m.syncInterval {
			return
		}
	}

	currentFiles := m.getFilesFn()
	currentRemotePaths := make(map[string]bool)
	for _, fp := range currentFiles {
		currentRemotePaths[fp.RemotePath] = true
	}

	// Detect uploads: new or changed files.
	var toUpload []FilePair
	newFiles := make(map[string]fileKey)
	for k, v := range m.syncedFiles {
		newFiles[k] = v
	}

	for _, fp := range currentFiles {
		fk := fileMtimeKey(fp.HostPath)
		if fk == nil {
			continue
		}
		existing, ok := m.syncedFiles[fp.RemotePath]
		if ok && existing == *fk {
			continue
		}
		toUpload = append(toUpload, fp)
		newFiles[fp.RemotePath] = *fk
	}

	// Detect deletions: synced paths no longer in current set.
	var toDelete []string
	for remotePath := range m.syncedFiles {
		if !currentRemotePaths[remotePath] {
			toDelete = append(toDelete, remotePath)
		}
	}

	if len(toUpload) == 0 && len(toDelete) == 0 {
		m.lastSyncTime = time.Now()
		return
	}

	// Snapshot for rollback.
	prevFiles := make(map[string]fileKey)
	for k, v := range m.syncedFiles {
		prevFiles[k] = v
	}

	if len(toUpload) > 0 {
		slog.Debug("file_sync: uploading files", "count", len(toUpload))
	}
	if len(toDelete) > 0 {
		slog.Debug("file_sync: deleting stale remote files", "count", len(toDelete))
	}

	// Execute uploads.
	var err error
	if len(toUpload) > 0 && m.bulkUploadFn != nil {
		err = m.bulkUploadFn(toUpload)
	} else {
		for _, fp := range toUpload {
			if uploadErr := m.uploadFn(fp.HostPath, fp.RemotePath); uploadErr != nil {
				err = uploadErr
				break
			}
		}
	}

	if err != nil {
		m.syncedFiles = prevFiles
		m.lastSyncTime = time.Now()
		slog.Warn("file_sync: sync failed, rolled back state", "error", err)
		return
	}

	// Execute deletions.
	if len(toDelete) > 0 {
		if deleteErr := m.deleteFn(toDelete); deleteErr != nil {
			m.syncedFiles = prevFiles
			m.lastSyncTime = time.Now()
			slog.Warn("file_sync: delete failed, rolled back state", "error", deleteErr)
			return
		}
	}

	// Commit: all succeeded.
	for _, p := range toDelete {
		delete(newFiles, p)
	}
	m.syncedFiles = newFiles
	m.lastSyncTime = time.Now()
}

// QuotedRmCommand builds a shell "rm -f" command for a batch of remote paths.
func QuotedRmCommand(remotePaths []string) string {
	quoted := make([]string, len(remotePaths))
	for i, p := range remotePaths {
		quoted[i] = shellQuote(p)
	}
	return "rm -f " + strings.Join(quoted, " ")
}

// QuotedMkdirCommand builds a shell "mkdir -p" command for a batch of directories.
func QuotedMkdirCommand(dirs []string) string {
	quoted := make([]string, len(dirs))
	for i, d := range dirs {
		quoted[i] = shellQuote(d)
	}
	return "mkdir -p " + strings.Join(quoted, " ")
}

// UniqueParentDirs extracts sorted unique parent directories from file pairs.
func UniqueParentDirs(files []FilePair) []string {
	seen := make(map[string]bool)
	for _, fp := range files {
		dir := filepath.Dir(fp.RemotePath)
		seen[dir] = true
	}
	dirs := make([]string, 0, len(seen))
	for d := range seen {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}

// shellQuote wraps a string in single quotes for shell safety.
func shellQuote(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "'\"'\"'"))
}
