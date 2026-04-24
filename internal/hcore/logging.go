package hcore

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// LogConfig configures structured logging destination and rotation.
type LogConfig struct {
	Level      string // "debug"|"info"|"warn"|"error"; default "info"
	JSONFormat bool
	LogDir     string // if non-empty, mirror output to <LogDir>/agent.log with rotation
	MaxBytes   int64  // rotate when file exceeds; default 10MiB
	MaxFiles   int    // keep current + MaxFiles-1 rotated; default 5
}

// SetupLogging installs a slog default that writes to stderr, and optionally
// mirrors to a size-rotated file in LogDir. Returns the configured logger
// and a cleanup function the caller should defer (closes the log file).
// Backward compatibility: if cfg is the legacy (level, jsonFormat) pair,
// callers can still build the same config manually.
func SetupLogging(cfg LogConfig) (*slog.Logger, func()) {
	var lvl slog.Level
	switch cfg.Level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	writers := []io.Writer{os.Stderr}
	var rotator *rotatingFile
	cleanup := func() {}

	if cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0o755); err == nil {
			maxBytes := cfg.MaxBytes
			if maxBytes <= 0 {
				maxBytes = 10 << 20
			}
			maxFiles := cfg.MaxFiles
			if maxFiles <= 0 {
				maxFiles = 5
			}
			r, err := openRotatingFile(filepath.Join(cfg.LogDir, "agent.log"), maxBytes, maxFiles)
			if err == nil {
				rotator = r
				writers = append(writers, r)
				cleanup = func() { _ = r.Close() }
			}
		}
	}
	_ = rotator

	w := io.MultiWriter(writers...)
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if cfg.JSONFormat {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, cleanup
}

// rotatingFile is an io.Writer that rotates the backing file once it grows
// past maxBytes. On rotation, agent.log becomes agent.log.1, .1 becomes .2,
// etc. up to maxFiles total files; the oldest is removed.
type rotatingFile struct {
	path     string
	maxBytes int64
	maxFiles int

	mu   sync.Mutex
	f    *os.File
	size int64
}

func openRotatingFile(path string, maxBytes int64, maxFiles int) (*rotatingFile, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat log: %w", err)
	}
	return &rotatingFile{
		path:     path,
		maxBytes: maxBytes,
		maxFiles: maxFiles,
		f:        f,
		size:     info.Size(),
	}, nil
}

func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size+int64(len(p)) > r.maxBytes {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}

// rotate closes the current file, shifts .N→.N+1, opens a fresh path.
// Must be called with r.mu held.
func (r *rotatingFile) rotate() error {
	if err := r.f.Close(); err != nil {
		return fmt.Errorf("close for rotate: %w", err)
	}
	// Delete oldest if it exists.
	oldest := fmt.Sprintf("%s.%d", r.path, r.maxFiles-1)
	_ = os.Remove(oldest)
	// Shift N→N+1 from N=maxFiles-2 down to N=1.
	for i := r.maxFiles - 2; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", r.path, i)
		dst := fmt.Sprintf("%s.%d", r.path, i+1)
		_ = os.Rename(src, dst)
	}
	// Move current to .1.
	_ = os.Rename(r.path, r.path+".1")

	f, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("reopen log after rotate: %w", err)
	}
	r.f = f
	r.size = 0
	return nil
}
