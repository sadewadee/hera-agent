package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Default behaviour for an MCP server that doesn't specify a lifecycle.
// on_demand keeps RAM low by killing idle subprocesses, and 5 minutes
// is long enough that a typical conversation won't suffer a respawn
// mid-turn but short enough that a dormant Python sidecar doesn't
// linger for hours.
const (
	LifecycleDaemon   = "daemon"
	LifecycleOnDemand = "on_demand"

	defaultIdleTimeout = 5 * time.Minute
	idleCheckInterval  = 30 * time.Second
)

// ManagedClient wraps the stdio *Client and adds a lifecycle so the
// backing subprocess can be killed when idle and respawned on demand.
// A ManagedClient is safe for concurrent callers; the mutex protects
// spawn / kill transitions.
//
// Tool discovery still happens once at construction so Hera can
// register proxy tools into its registry up front (callers don't need
// to know MCP internals). The discovered tool list is cached so
// respawning doesn't require callers to re-query names.
type ManagedClient struct {
	cfg         MCPServerConfig
	mode        string
	idleTimeout time.Duration

	mu       sync.Mutex
	client   *Client
	tools    []MCPToolDef
	lastUsed time.Time

	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewManagedClient spawns the backing server once, discovers tools,
// and — when mode == on_demand — starts a background goroutine that
// kills the subprocess after idleTimeout with no activity. The first
// subsequent tool call respawns it.
func NewManagedClient(cfg MCPServerConfig) (*ManagedClient, error) {
	mode := cfg.Mode
	if mode == "" {
		mode = LifecycleOnDemand
	}
	timeout := cfg.IdleTimeout
	if timeout <= 0 {
		timeout = defaultIdleTimeout
	}

	m := &ManagedClient{
		cfg:         cfg,
		mode:        mode,
		idleTimeout: timeout,
		lastUsed:    time.Now(),
		stopCh:      make(chan struct{}),
	}

	c, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	m.client = c
	m.tools = c.Tools

	if mode == LifecycleOnDemand {
		go m.idleLoop()
	}
	return m, nil
}

// Tools returns the cached tool definitions from the most recent
// successful spawn. Survives idle-kill cycles.
func (m *ManagedClient) Tools() []MCPToolDef {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MCPToolDef, len(m.tools))
	copy(out, m.tools)
	return out
}

// Name returns the configured server name.
func (m *ManagedClient) Name() string { return m.cfg.Name }

// Mode returns the lifecycle mode in effect.
func (m *ManagedClient) Mode() string { return m.mode }

// CallTool ensures the backing client is alive (respawn if needed),
// updates the last-used timestamp, and delegates to the underlying
// stdio client. Lock is held only around the respawn path so calls
// overlap cleanly once the client is live.
func (m *ManagedClient) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	c, err := m.ensure()
	if err != nil {
		return "", err
	}
	m.touch()
	return c.CallTool(ctx, name, args)
}

// ensure returns a live Client, spawning a new one if the previous
// subprocess was killed by the idle loop.
func (m *ManagedClient) ensure() (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return m.client, nil
	}
	slog.Info("mcp: respawning on demand", "server", m.cfg.Name)
	c, err := NewClient(m.cfg)
	if err != nil {
		return nil, fmt.Errorf("respawn %s: %w", m.cfg.Name, err)
	}
	m.client = c
	// Refresh the tool cache in case the server's tool set changed
	// (e.g. Python MCP may have newly-registered py_<name> tools).
	m.tools = c.Tools
	return c, nil
}

func (m *ManagedClient) touch() {
	m.mu.Lock()
	m.lastUsed = time.Now()
	m.mu.Unlock()
}

// idleLoop runs only in on_demand mode. Periodically checks whether
// the backing process has been unused for idleTimeout and kills it if
// so. The next CallTool will respawn.
func (m *ManagedClient) idleLoop() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.maybeKillIdle()
		}
	}
}

func (m *ManagedClient) maybeKillIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return
	}
	if time.Since(m.lastUsed) < m.idleTimeout {
		return
	}
	slog.Info("mcp: killing idle subprocess",
		"server", m.cfg.Name,
		"idle_for", time.Since(m.lastUsed).Round(time.Second),
	)
	_ = m.client.Close()
	m.client = nil
}

// Close stops the idle loop and kills any live subprocess. Safe to
// call multiple times.
func (m *ManagedClient) Close() error {
	m.stopOnce.Do(func() { close(m.stopCh) })
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil {
		err := m.client.Close()
		m.client = nil
		return err
	}
	return nil
}
