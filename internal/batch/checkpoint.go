package batch

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// PromptStatus tracks the lifecycle of a single prompt.
type PromptStatus string

const (
	StatusPending   PromptStatus = "pending"
	StatusRunning   PromptStatus = "running"
	StatusCompleted PromptStatus = "completed"
	StatusFailed    PromptStatus = "failed"
)

// CheckpointStore persists per-prompt status so runs can be resumed.
type CheckpointStore interface {
	// SetStatus records the status (and optional error message) for prompt[index].
	SetStatus(runID string, index int, prompt string, status PromptStatus, errMsg string) error
	// GetStatus returns the current status for prompt[index], or StatusPending
	// if no record exists yet.
	GetStatus(runID string, index int) (PromptStatus, error)
	// Close releases the store's resources.
	Close() error
}

// SQLiteCheckpointStore is a SQLite-backed CheckpointStore.
// It uses WAL mode and busy_timeout=5000 to match the main memory store pattern.
type SQLiteCheckpointStore struct {
	db *sql.DB
}

// NewSQLiteCheckpointStore opens (or creates) a checkpoint DB at dbPath.
// Parent directories are created as needed.
func NewSQLiteCheckpointStore(dbPath string) (*SQLiteCheckpointStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create checkpoint dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open checkpoint db: %w", err)
	}

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	s := &SQLiteCheckpointStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate checkpoint db: %w", err)
	}
	return s, nil
}

func (s *SQLiteCheckpointStore) migrate() error {
	_, err := s.db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS checkpoints (
			run_id     TEXT    NOT NULL,
			idx        INTEGER NOT NULL,
			prompt     TEXT    NOT NULL DEFAULT '',
			status     TEXT    NOT NULL DEFAULT 'pending',
			error_msg  TEXT    NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (run_id, idx)
		)
	`)
	return err
}

// SetStatus upserts the checkpoint record for (runID, index).
func (s *SQLiteCheckpointStore) SetStatus(runID string, index int, prompt string, status PromptStatus, errMsg string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(context.Background(),
		`INSERT INTO checkpoints (run_id, idx, prompt, status, error_msg, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(run_id, idx) DO UPDATE SET
		   status     = excluded.status,
		   error_msg  = excluded.error_msg,
		   updated_at = excluded.updated_at`,
		runID, index, prompt, string(status), errMsg, now,
	)
	return err
}

// GetStatus returns the stored status for (runID, index) or StatusPending if absent.
func (s *SQLiteCheckpointStore) GetStatus(runID string, index int) (PromptStatus, error) {
	var status string
	err := s.db.QueryRowContext(context.Background(),
		`SELECT status FROM checkpoints WHERE run_id = ? AND idx = ?`,
		runID, index,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return StatusPending, nil
	}
	if err != nil {
		return StatusPending, fmt.Errorf("get checkpoint status: %w", err)
	}
	return PromptStatus(status), nil
}

// Close closes the underlying database connection.
func (s *SQLiteCheckpointStore) Close() error {
	return s.db.Close()
}

// NoopCheckpointStore is a CheckpointStore that does nothing.
// Use when persistence is not needed (e.g., estimate mode).
type NoopCheckpointStore struct{}

func (NoopCheckpointStore) SetStatus(_ string, _ int, _ string, _ PromptStatus, _ string) error {
	return nil
}

func (NoopCheckpointStore) GetStatus(_ string, _ int) (PromptStatus, error) {
	return StatusPending, nil
}

func (NoopCheckpointStore) Close() error { return nil }
