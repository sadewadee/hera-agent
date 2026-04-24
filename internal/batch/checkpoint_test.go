package batch

import (
	"path/filepath"
	"testing"
)

func TestSQLiteCheckpointStore_SetGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteCheckpointStore(filepath.Join(dir, "checkpoint.db"))
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}
	defer store.Close()

	// Unknown key returns StatusPending.
	status, err := store.GetStatus("run1", 0)
	if err != nil {
		t.Fatalf("GetStatus missing key: %v", err)
	}
	if status != StatusPending {
		t.Errorf("want StatusPending for new key, got %v", status)
	}

	// Set to running.
	if err := store.SetStatus("run1", 0, "hello", StatusRunning, ""); err != nil {
		t.Fatalf("SetStatus running: %v", err)
	}
	status, err = store.GetStatus("run1", 0)
	if err != nil {
		t.Fatalf("GetStatus after set: %v", err)
	}
	if status != StatusRunning {
		t.Errorf("want StatusRunning, got %v", status)
	}

	// Upsert to completed.
	if err := store.SetStatus("run1", 0, "hello", StatusCompleted, ""); err != nil {
		t.Fatalf("SetStatus completed: %v", err)
	}
	status, err = store.GetStatus("run1", 0)
	if err != nil {
		t.Fatalf("GetStatus after update: %v", err)
	}
	if status != StatusCompleted {
		t.Errorf("want StatusCompleted, got %v", status)
	}
}

func TestSQLiteCheckpointStore_MultipleRuns(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteCheckpointStore(filepath.Join(dir, "checkpoint.db"))
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}
	defer store.Close()

	for _, run := range []string{"runA", "runB"} {
		for i := 0; i < 3; i++ {
			if err := store.SetStatus(run, i, "p", StatusCompleted, ""); err != nil {
				t.Fatalf("SetStatus %s[%d]: %v", run, i, err)
			}
		}
	}

	// Cross-run isolation.
	status, _ := store.GetStatus("runA", 0)
	if status != StatusCompleted {
		t.Errorf("runA[0]: want completed, got %v", status)
	}
	status, _ = store.GetStatus("runB", 2)
	if status != StatusCompleted {
		t.Errorf("runB[2]: want completed, got %v", status)
	}
	// runC doesn't exist.
	status, _ = store.GetStatus("runC", 0)
	if status != StatusPending {
		t.Errorf("runC[0]: want pending, got %v", status)
	}
}

func TestSQLiteCheckpointStore_WithError(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteCheckpointStore(filepath.Join(dir, "checkpoint.db"))
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}
	defer store.Close()

	if err := store.SetStatus("run1", 5, "prompt", StatusFailed, "deadline exceeded"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	status, err := store.GetStatus("run1", 5)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != StatusFailed {
		t.Errorf("want StatusFailed, got %v", status)
	}
}

func TestNoopCheckpointStore(t *testing.T) {
	var s NoopCheckpointStore
	if err := s.SetStatus("r", 0, "p", StatusCompleted, ""); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	status, err := s.GetStatus("r", 0)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status != StatusPending {
		t.Errorf("want StatusPending, got %v", status)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
