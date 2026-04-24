package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/tools"
)

func TestCustomToolWatcher_RegistersFromYAMLOnStart(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: ping_example
description: curl to example.com
type: command
command: curl -sS https://example.com
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ping.yaml"), []byte(yaml), 0o644))

	reg := tools.NewRegistry()
	w := NewCustomToolWatcher(reg, dir)
	w.Start(context.Background())
	defer w.Stop()

	got, ok := reg.Get("ping_example")
	require.True(t, ok, "tool should be registered after Start")
	assert.Equal(t, "ping_example", got.Name())
	assert.Contains(t, got.Description(), "curl")
}

func TestCustomToolWatcher_PicksUpNewFile(t *testing.T) {
	dir := t.TempDir()

	reg := tools.NewRegistry()
	w := NewCustomToolWatcher(reg, dir)
	// Tight interval so the test is quick.
	w.interval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// File does not exist yet — tool should NOT be registered.
	_, ok := reg.Get("added_later")
	assert.False(t, ok, "tool should not exist before file is created")

	yaml := `
name: added_later
description: added at runtime
type: command
command: echo hi
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "added_later.yaml"), []byte(yaml), 0o644))

	// Allow the watcher to tick.
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("added_later"); ok {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("watcher never picked up the new file")
}

func TestCustomToolWatcher_IgnoresNonYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a tool"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden.yaml"), []byte("name: x\ntype: command\ncommand: echo"), 0o644))

	reg := tools.NewRegistry()
	w := NewCustomToolWatcher(reg, dir)
	w.Start(context.Background())
	defer w.Stop()

	defs := reg.ToolDefs()
	for _, d := range defs {
		assert.NotEqual(t, "notes", d.Name)
		assert.NotEqual(t, ".hidden", d.Name)
		assert.NotEqual(t, "x", d.Name, "hidden files must be ignored")
	}
}

func TestCustomToolWatcher_UnregistersOnFileDeletion(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
name: temp_tool
description: temporary tool for deletion test
type: command
command: echo temporary
`
	yamlPath := filepath.Join(dir, "temp_tool.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o644))

	reg := tools.NewRegistry()
	w := NewCustomToolWatcher(reg, dir)
	w.interval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Wait for initial registration.
	deadline := time.Now().Add(1500 * time.Millisecond)
	registered := false
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("temp_tool"); ok {
			registered = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.True(t, registered, "tool should register after YAML is written")

	// Delete the YAML file.
	require.NoError(t, os.Remove(yamlPath))

	// Wait for watcher to detect deletion and unregister.
	deadline = time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := reg.Get("temp_tool"); !ok {
			return // tool is gone
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("watcher never unregistered the deleted tool")
}

func TestRegistryUnregister(t *testing.T) {
	reg := tools.NewRegistry()
	// Register a real built-in tool (datetime) and then unregister it.
	RegisterDatetime(reg)

	_, ok := reg.Get("datetime")
	require.True(t, ok, "datetime tool should exist before unregister")

	require.NoError(t, reg.Unregister("datetime"))

	_, ok = reg.Get("datetime")
	assert.False(t, ok, "datetime tool should not exist after unregister")

	// Unregistering again should return an error.
	err := reg.Unregister("datetime")
	assert.Error(t, err)
}

func TestRegisterCustomToolTool_WritesYAML(t *testing.T) {
	dir := t.TempDir()
	reg := tools.NewRegistry()
	RegisterCustomToolTool(reg, dir)

	tool, ok := reg.Get("register_custom_tool")
	require.True(t, ok)

	args, err := json.Marshal(map[string]any{
		"name":        "github_status",
		"description": "check GitHub status page",
		"type":        "command",
		"command":     "curl -sS https://www.githubstatus.com/api/v2/status.json",
	})
	require.NoError(t, err)

	res, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	require.False(t, res.IsError, "unexpected: %s", res.Content)

	path := filepath.Join(dir, "github_status.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: github_status")
	assert.Contains(t, string(data), "type: command")
}

func TestRegisterCustomToolTool_RejectsInvalidName(t *testing.T) {
	dir := t.TempDir()
	reg := tools.NewRegistry()
	RegisterCustomToolTool(reg, dir)

	tool, _ := reg.Get("register_custom_tool")
	args, _ := json.Marshal(map[string]any{
		"name":        "Has Spaces",
		"description": "bad",
		"type":        "command",
		"command":     "echo",
	})
	res, _ := tool.Execute(context.Background(), args)
	assert.True(t, res.IsError)
}

func TestRegisterCustomToolTool_RequiresFieldPerType(t *testing.T) {
	dir := t.TempDir()
	reg := tools.NewRegistry()
	RegisterCustomToolTool(reg, dir)
	tool, _ := reg.Get("register_custom_tool")

	// type=command without command field
	args, _ := json.Marshal(map[string]any{
		"name":        "no_body",
		"description": "bad",
		"type":        "command",
	})
	res, _ := tool.Execute(context.Background(), args)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "requires a command")

	// type=http without url field
	args, _ = json.Marshal(map[string]any{
		"name":        "no_url",
		"description": "bad",
		"type":        "http",
	})
	res, _ = tool.Execute(context.Background(), args)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "url")
}
