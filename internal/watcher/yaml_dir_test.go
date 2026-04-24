package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collect returns a thread-safe helper that appends events as the watcher
// fires them, plus a waitFor helper that blocks until N events are observed
// or the timeout elapses.
func collect() (*sync.Mutex, *[]Event, func(int, time.Duration) bool) {
	mu := &sync.Mutex{}
	events := make([]Event, 0)
	waitFor := func(n int, timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			mu.Lock()
			got := len(events)
			mu.Unlock()
			if got >= n {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		return false
	}
	return mu, &events, waitFor
}

func TestNew_Defaults(t *testing.T) {
	w := New(Config{Dir: "/nonexistent"})
	assert.Equal(t, 5*time.Second, w.cfg.PollInterval)
}

func TestStart_EmitsAddForExistingFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("name: a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.yml"), []byte("name: b"), 0o644))

	mu, events, waitFor := collect()
	w := New(Config{
		Dir:          dir,
		PollInterval: 50 * time.Millisecond,
		OnEvent: func(e Event) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, e)
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	require.True(t, waitFor(2, 500*time.Millisecond), "expected 2 Add events for seeded files")

	mu.Lock()
	defer mu.Unlock()
	names := []string{(*events)[0].Name, (*events)[1].Name}
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
	for _, e := range *events {
		assert.Equal(t, EventAdd, e.Kind)
	}
}

func TestStart_DetectsNewFile(t *testing.T) {
	dir := t.TempDir()

	mu, events, waitFor := collect()
	w := New(Config{
		Dir:          dir,
		PollInterval: 30 * time.Millisecond,
		OnEvent: func(e Event) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, e)
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	// Empty dir — give it one poll cycle to settle.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.yaml"), []byte("name: new"), 0o644))
	require.True(t, waitFor(1, 300*time.Millisecond), "expected Add event after new file")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "new", (*events)[0].Name)
	assert.Equal(t, EventAdd, (*events)[0].Kind)
}

func TestStart_DetectsModify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: original"), 0o644))

	mu, events, waitFor := collect()
	w := New(Config{
		Dir:          dir,
		PollInterval: 30 * time.Millisecond,
		OnEvent: func(e Event) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, e)
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	require.True(t, waitFor(1, 300*time.Millisecond), "expected initial Add")

	require.NoError(t, os.WriteFile(path, []byte("name: changed"), 0o644))
	require.True(t, waitFor(2, 300*time.Millisecond), "expected Modify after content change")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, EventModify, (*events)[1].Kind)
	assert.Equal(t, "name: changed", string((*events)[1].Content))
}

func TestStart_DetectsDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "d.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: d"), 0o644))

	mu, events, waitFor := collect()
	w := New(Config{
		Dir:          dir,
		PollInterval: 30 * time.Millisecond,
		OnEvent: func(e Event) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, e)
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	require.True(t, waitFor(1, 300*time.Millisecond), "expected initial Add")

	require.NoError(t, os.Remove(path))
	require.True(t, waitFor(2, 300*time.Millisecond), "expected Delete after file removal")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, EventDelete, (*events)[1].Kind)
	assert.Equal(t, "d", (*events)[1].Name)
}

func TestStart_IgnoresDotfilesAndNonYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden.yaml"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.yaml"), []byte("x"), 0o644))

	mu, events, waitFor := collect()
	w := New(Config{
		Dir:          dir,
		PollInterval: 30 * time.Millisecond,
		OnEvent: func(e Event) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, e)
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	require.True(t, waitFor(1, 300*time.Millisecond), "expected exactly one Add for ok.yaml")

	// Wait one more poll cycle to prove nothing else fires.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, *events, 1, "dotfiles and non-yaml must be ignored")
	assert.Equal(t, "ok", (*events)[0].Name)
}

func TestStart_MissingDirIsEmpty(t *testing.T) {
	w := New(Config{
		Dir:          "/absolutely-does-not-exist-i-hope",
		PollInterval: 30 * time.Millisecond,
		OnEvent:      func(Event) { t.Fatal("no events expected for missing dir") },
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestStop_Idempotent(t *testing.T) {
	w := New(Config{Dir: t.TempDir(), PollInterval: 30 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	w.Stop()
	w.Stop() // must not panic.
}
