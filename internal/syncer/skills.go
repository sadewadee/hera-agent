// Package syncer implements the bundled-skills sync mechanism for Hera.
//
// On first install (or after hera init) it copies all .md files from
// $HERA_BUNDLED/skills/ into $HERA_HOME/skills/, recording the MD5 hash
// of each bundled file in a manifest at $HERA_HOME/skills/.bundled_manifest.
//
// On subsequent runs it re-syncs with these rules:
//   - File absent in user dir   → copy (new file from upstream).
//   - User hash == manifest hash → file unmodified → copy (propagates upstream update).
//   - User hash != manifest hash → user-modified  → skip + log "preserved".
//   - File absent in bundled dir → removed upstream → no action (user keeps their copy).
package syncer

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// SyncStats reports what happened during a Sync call.
type SyncStats struct {
	// Copied is the number of files written to the user dir (new or updated).
	Copied int
	// Preserved is the number of user-modified files that were left untouched.
	Preserved int
	// Skipped is reserved for future use (e.g. non-.md files).
	Skipped int
}

const manifestFilename = ".bundled_manifest"

// Syncer copies bundled skills into the user skills directory, preserving
// user edits.
type Syncer struct {
	bundledDir string // read-only source (e.g. $HERA_BUNDLED/skills)
	userDir    string // writable destination (e.g. $HERA_HOME/skills)
}

// New creates a Syncer.
//   - bundledDir: path to $HERA_BUNDLED/skills (may not exist — Sync returns early).
//   - userDir:    path to $HERA_HOME/skills (created if absent).
func New(bundledDir, userDir string) *Syncer {
	return &Syncer{bundledDir: bundledDir, userDir: userDir}
}

// Sync performs the copy-on-modify sync.
// It returns zero-value SyncStats (no error) when bundledDir does not exist,
// so callers don't need to special-case the `go install` path where there is
// no bundled dir.
func (s *Syncer) Sync() (SyncStats, error) {
	var stats SyncStats

	// If the bundled directory doesn't exist, there is nothing to sync.
	if _, err := os.Stat(s.bundledDir); os.IsNotExist(err) {
		return stats, nil
	}

	// Ensure user skills dir exists.
	if err := os.MkdirAll(s.userDir, 0o755); err != nil {
		return stats, fmt.Errorf("create user skills dir: %w", err)
	}

	// Load existing manifest (maps relative path → md5 of bundled content).
	manifest, err := loadManifest(filepath.Join(s.userDir, manifestFilename))
	if err != nil {
		return stats, fmt.Errorf("load manifest: %w", err)
	}

	// Walk bundled skills, sync each .md file.
	newManifest := make(map[string]string)

	walkErr := filepath.WalkDir(s.bundledDir, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			stats.Skipped++
			return nil
		}

		rel, err := filepath.Rel(s.bundledDir, srcPath)
		if err != nil {
			return fmt.Errorf("rel path for %s: %w", srcPath, err)
		}

		bundledContent, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read bundled skill %s: %w", rel, err)
		}
		bundledHash := hashBytes(bundledContent)
		newManifest[rel] = bundledHash

		dstPath := filepath.Join(s.userDir, rel)

		// Does the user file exist?
		dstContent, readErr := os.ReadFile(dstPath)
		if os.IsNotExist(readErr) {
			// File does not exist in user dir — copy it.
			if err := writeFile(dstPath, bundledContent); err != nil {
				return err
			}
			stats.Copied++
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("read user skill %s: %w", rel, readErr)
		}

		// File exists. Determine if the user modified it.
		userHash := hashBytes(dstContent)
		manifestHash, inManifest := manifest[rel]

		if !inManifest {
			// No manifest entry → pre-existing file from old install.
			// Treat as user-modified to be safe.
			slog.Info("syncer: no manifest entry, preserving pre-existing file", "path", rel)
			stats.Preserved++
			return nil
		}

		if userHash != manifestHash {
			// User modified the file after the last sync.
			slog.Info("syncer: user-modified, preserved", "path", rel)
			stats.Preserved++
			return nil
		}

		// File is unmodified. Check if bundled version changed.
		if bundledHash == manifestHash {
			// No change upstream, no change by user — nothing to do.
			return nil
		}

		// Upstream updated the file; user didn't touch it — overwrite.
		if err := writeFile(dstPath, bundledContent); err != nil {
			return err
		}
		slog.Info("syncer: applied upstream update", "path", rel)
		stats.Copied++
		return nil
	})
	if walkErr != nil {
		return stats, walkErr
	}

	// Write updated manifest.
	if err := writeManifest(filepath.Join(s.userDir, manifestFilename), newManifest); err != nil {
		return stats, fmt.Errorf("write manifest: %w", err)
	}

	return stats, nil
}

// hashBytes returns the hex-encoded MD5 of b.
func hashBytes(b []byte) string {
	h := md5.Sum(b)
	return hex.EncodeToString(h[:])
}

// writeFile writes content to path, creating parent directories as needed.
func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// loadManifest reads the manifest file and returns a map of relative path → MD5 hash.
// Returns an empty map (no error) if the file does not exist.
func loadManifest(manifestPath string) (map[string]string, error) {
	m := make(map[string]string)
	f, err := os.Open(manifestPath)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open manifest: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		hash, rel := parts[0], parts[1]
		m[rel] = hash
	}
	return m, scanner.Err()
}

// writeManifest writes the manifest as "hash  relative/path" lines,
// one per file (md5sum-compatible format).
func writeManifest(manifestPath string, m map[string]string) error {
	f, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("create manifest: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for rel, hash := range m {
		if _, err := fmt.Fprintf(w, "%s  %s\n", hash, rel); err != nil {
			return err
		}
	}
	return w.Flush()
}

// ManifestReader exposes the raw manifest for external tooling (e.g. hera init --reset).
func ManifestReader(userDir string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(userDir, manifestFilename))
}
