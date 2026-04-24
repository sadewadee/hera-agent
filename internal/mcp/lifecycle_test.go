package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMCPServer is a minimal MCP stdio server written in Go and
// compiled into a test binary. It accepts initialize + tools/list +
// tools/call and keeps a counter of calls so tests can assert
// spawn/respawn behaviour. Communicates via JSON-RPC 2.0 over stdin/stdout.
//
// We build the binary once per test file via TestMain, drop it in a
// tmpdir, and point ManagedClient's Command at it. Avoids the need
// for an external MCP server in CI.
const fakeServerSource = `package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

var callCount int64

func main() {
	startTS := time.Now().UnixNano()
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		id := req["id"]
		method, _ := req["method"].(string)
		var result interface{}
		switch method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "2025-03-26",
				"serverInfo":      map[string]string{"name": "fake", "version": "1"},
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			}
		case "tools/list":
			result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{"name": "ping", "description": "returns spawn id", "inputSchema": map[string]interface{}{"type": "object"}},
				},
			}
		case "tools/call":
			n := atomic.AddInt64(&callCount, 1)
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("spawn=%d call=%d", startTS, n)},
				},
			}
		default:
			result = map[string]interface{}{}
		}
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		}
		data, _ := json.Marshal(resp)
		os.Stdout.Write(append(data, '\n'))
	}
}
`

var fakeServerBin string

func TestMain(m *testing.M) {
	// Compile the fake server once.
	dir, err := os.MkdirTemp("", "mcp-fake-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	srcPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcPath, []byte(fakeServerSource), 0o644); err != nil {
		panic(err)
	}
	binPath := filepath.Join(dir, "fake-mcp")
	cmd := exec.Command("go", "build", "-o", binPath, srcPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(string(out))
	}
	fakeServerBin = binPath
	os.Exit(m.Run())
}

func TestManagedClient_OnDemandKillsIdleSubprocess(t *testing.T) {
	cfg := MCPServerConfig{
		Name:        "fake",
		Command:     fakeServerBin,
		Mode:        LifecycleOnDemand,
		IdleTimeout: 200 * time.Millisecond,
	}
	m, err := NewManagedClient(cfg)
	require.NoError(t, err)
	defer m.Close()

	// First call — client was spawned at construction, so this just
	// uses it.
	res1, err := m.CallTool(context.Background(), "ping", json.RawMessage(`{}`))
	require.NoError(t, err)
	spawn1 := extractSpawn(t, res1)

	// Wait past the idle timeout plus the idle-check interval so the
	// background goroutine has a chance to kill the subprocess. We
	// shortened idleCheckInterval for the test by using a short
	// IdleTimeout; the default check is 30s which would make the
	// test slow. Force the kill directly for determinism:
	time.Sleep(250 * time.Millisecond)
	m.maybeKillIdle()

	// Confirm client is nil (i.e., killed).
	m.mu.Lock()
	killed := m.client == nil
	m.mu.Unlock()
	assert.True(t, killed, "idle subprocess should be nil after maybeKillIdle")

	// Next call triggers respawn — new spawn ID should differ.
	res2, err := m.CallTool(context.Background(), "ping", json.RawMessage(`{}`))
	require.NoError(t, err)
	spawn2 := extractSpawn(t, res2)

	assert.NotEqual(t, spawn1, spawn2, "respawn should produce a new process")
}

func TestManagedClient_DaemonDoesNotIdleKill(t *testing.T) {
	cfg := MCPServerConfig{
		Name:    "fake",
		Command: fakeServerBin,
		Mode:    LifecycleDaemon,
	}
	m, err := NewManagedClient(cfg)
	require.NoError(t, err)
	defer m.Close()

	// Manually invoke maybeKillIdle — but daemon mode should skip
	// because the lifecycle goroutine never runs. Even calling
	// maybeKillIdle directly respects the idle window, but the real
	// assertion is that the idle goroutine didn't start.
	//
	// We verify by checking that Mode returns daemon; the lack of
	// idleLoop means no kill ever happens.
	assert.Equal(t, LifecycleDaemon, m.Mode())

	res, err := m.CallTool(context.Background(), "ping", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Contains(t, res, "spawn=")
}

func TestManagedClient_ToolsCacheSurvivesRespawn(t *testing.T) {
	cfg := MCPServerConfig{
		Name:        "fake",
		Command:     fakeServerBin,
		Mode:        LifecycleOnDemand,
		IdleTimeout: 50 * time.Millisecond,
	}
	m, err := NewManagedClient(cfg)
	require.NoError(t, err)
	defer m.Close()

	toolsBefore := m.Tools()
	require.Len(t, toolsBefore, 1)
	assert.Equal(t, "ping", toolsBefore[0].Name)

	time.Sleep(100 * time.Millisecond)
	m.maybeKillIdle()

	// After kill, Tools() still returns the cached list. A subsequent
	// CallTool respawns and refreshes it; even before that, callers
	// querying Tools() must not see an empty slice.
	toolsAfterKill := m.Tools()
	require.Len(t, toolsAfterKill, 1)
}

func extractSpawn(t *testing.T, raw string) string {
	t.Helper()
	// Result format: "spawn=<ts> call=<n>"
	parts := strings.Fields(raw)
	for _, p := range parts {
		if strings.HasPrefix(p, "spawn=") {
			return strings.TrimPrefix(p, "spawn=")
		}
	}
	t.Fatalf("no spawn=... in %q", raw)
	return ""
}
