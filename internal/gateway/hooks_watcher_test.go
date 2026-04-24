package gateway

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHooksWatcher_LoadsOnStart(t *testing.T) {
	dir := t.TempDir()
	writeHookFile(t, dir, "test.yaml", `
- name: my-hook
  event: before_message
  type: command
  command: echo hello
`)
	hm := NewHookManager()
	w := NewHooksWatcher(hm, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	assert.True(t, hookRegistered(hm, "my-hook"), "hook should be registered on start")
}

func TestHooksWatcher_PicksUpNewFile(t *testing.T) {
	dir := t.TempDir()
	hm := NewHookManager()
	w := NewHooksWatcher(hm, dir)
	w.interval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	// No hooks yet.
	assert.False(t, hookRegistered(hm, "new-hook"))

	// Drop a new file.
	writeHookFile(t, dir, "new.yaml", `
- name: new-hook
  event: after_message
  type: command
  command: echo after
`)

	require.Eventually(t, func() bool {
		return hookRegistered(hm, "new-hook")
	}, 500*time.Millisecond, 20*time.Millisecond, "new hook should be registered after file appears")
}

func TestHooksWatcher_UpdatesOnFileChange(t *testing.T) {
	dir := t.TempDir()
	writeHookFile(t, dir, "hooks.yaml", `
- name: hook-v1
  event: before_message
  type: command
  command: echo v1
`)
	hm := NewHookManager()
	w := NewHooksWatcher(hm, dir)
	w.interval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	require.True(t, hookRegistered(hm, "hook-v1"))

	// Overwrite with a different hook name.
	writeHookFile(t, dir, "hooks.yaml", `
- name: hook-v2
  event: before_message
  type: command
  command: echo v2
`)

	require.Eventually(t, func() bool {
		return hookRegistered(hm, "hook-v2")
	}, 500*time.Millisecond, 20*time.Millisecond, "updated hook should be registered")

	assert.False(t, hookRegistered(hm, "hook-v1"), "old hook name should be unregistered")
}

func TestHooksWatcher_RemovesHooksOnFileDeletion(t *testing.T) {
	dir := t.TempDir()
	path := writeHookFile(t, dir, "gone.yaml", `
- name: gone-hook
  event: before_message
  type: command
  command: echo bye
`)
	hm := NewHookManager()
	w := NewHooksWatcher(hm, dir)
	w.interval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	require.True(t, hookRegistered(hm, "gone-hook"))

	// Delete the file.
	require.NoError(t, os.Remove(path))

	require.Eventually(t, func() bool {
		return !hookRegistered(hm, "gone-hook")
	}, 500*time.Millisecond, 20*time.Millisecond, "hook should be unregistered after file is deleted")
}

func TestHooksWatcher_SkipsIncompleteEntries(t *testing.T) {
	dir := t.TempDir()
	writeHookFile(t, dir, "partial.yaml", `
- name: ""
  event: before_message
  type: command
  command: echo incomplete
- name: good-hook
  event: after_message
  type: command
  command: echo ok
`)
	hm := NewHookManager()
	w := NewHooksWatcher(hm, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	defer w.Stop()

	assert.True(t, hookRegistered(hm, "good-hook"), "valid hook should be registered")
}

// --- helpers ---

func writeHookFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func hookRegistered(hm *HookManager, name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	for _, h := range hm.hooks {
		if h.Name() == name {
			return true
		}
	}
	return false
}
