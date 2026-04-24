// Package watcher provides a shared directory watcher primitive for YAML
// files. It polls a directory at a fixed interval, diffs observed file
// contents by SHA-256, and emits add/modify/delete callbacks for every
// `.yaml` or `.yml` file.
//
// It is the underlying primitive for:
//   - internal/tools/builtin/custom_watcher.go (tools.d/)
//   - internal/gateway/hooks_watcher.go        (hooks.d/)
//
// Poll interval is a deliberate choice over fsnotify: dep-free,
// platform-agnostic (container overlay FS, macOS FSEvents, Linux inotify
// limits all work the same way), and a 5-second delay is fine for human-
// or agent-driven config file edits.
package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Event describes a single change observed by the poller.
type Event struct {
	Path    string
	Name    string // basename without .yaml/.yml extension
	Content []byte
	Kind    EventKind
}

// EventKind discriminates add/modify/delete events.
type EventKind int

const (
	// EventAdd fires the first time a YAML file is observed in the directory.
	EventAdd EventKind = iota
	// EventModify fires when an existing YAML file's content hash changes.
	EventModify
	// EventDelete fires when a previously observed YAML file disappears.
	EventDelete
)

// Config configures a Dir poller.
type Config struct {
	// Dir is the absolute path watched. Missing directory is treated as empty.
	Dir string
	// PollInterval between scans. Zero defaults to 5s.
	PollInterval time.Duration
	// OnEvent is called for every Add/Modify/Delete event. Must be safe for
	// concurrent use — the poller calls it in a single goroutine but the
	// callback may block future polls, so keep it fast or dispatch.
	OnEvent func(Event)
}

// Dir watches a single directory for YAML file changes.
type Dir struct {
	cfg    Config
	mu     sync.Mutex
	hashes map[string]string // path -> sha256 hex
	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a watcher. The caller invokes Start to begin polling.
func New(cfg Config) *Dir {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	return &Dir{
		cfg:    cfg,
		hashes: make(map[string]string),
	}
}

// Start begins polling in the background. Safe to call once; subsequent
// calls are no-ops. Stop via ctx cancellation or explicit Stop.
func (d *Dir) Start(ctx context.Context) {
	d.mu.Lock()
	if d.cancel != nil {
		d.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.done = make(chan struct{})
	d.mu.Unlock()

	// Initial scan to populate baseline state. All existing files emit
	// EventAdd so the caller can seed their registry.
	d.scan()

	go func() {
		defer close(d.done)
		ticker := time.NewTicker(d.cfg.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.scan()
			}
		}
	}()
}

// Stop halts polling and waits for the background goroutine to exit.
// Safe to call multiple times.
func (d *Dir) Stop() {
	d.mu.Lock()
	cancel := d.cancel
	done := d.done
	d.cancel = nil
	d.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

// scan is one poll cycle. It is safe for concurrent callers but the
// background goroutine holds exclusive access during normal operation.
func (d *Dir) scan() {
	entries, err := os.ReadDir(d.cfg.Dir)
	if err != nil {
		// Missing directory = no YAML files. Any previously-known files
		// should be treated as deletions.
		entries = nil
	}

	observed := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(d.cfg.Dir, name)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		sum := sha256.Sum256(data)
		observed[path] = hex.EncodeToString(sum[:])
	}

	d.mu.Lock()
	prev := d.hashes
	d.hashes = observed
	d.mu.Unlock()

	if d.cfg.OnEvent == nil {
		return
	}

	// Deletions: previously known paths missing from observed.
	for path := range prev {
		if _, ok := observed[path]; !ok {
			d.cfg.OnEvent(Event{
				Path: path,
				Name: baseName(path),
				Kind: EventDelete,
			})
		}
	}

	// Adds and modifies.
	for path, hash := range observed {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		old, known := prev[path]
		switch {
		case !known:
			d.cfg.OnEvent(Event{
				Path:    path,
				Name:    baseName(path),
				Content: data,
				Kind:    EventAdd,
			})
		case old != hash:
			d.cfg.OnEvent(Event{
				Path:    path,
				Name:    baseName(path),
				Content: data,
				Kind:    EventModify,
			})
		}
	}
}

func baseName(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	return base
}
