package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client connects to an external MCP server via stdio (JSON-RPC 2.0).
type Client struct {
	name    string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  atomic.Int64
	pending sync.Map // id -> chan json.RawMessage

	// Discovered tools from the server.
	Tools []MCPToolDef
}

// MCPToolDef describes a tool exposed by an MCP server.
type MCPToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// jsonrpcRequest is a JSON-RPC 2.0 request.
type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPServerConfig describes how to launch an MCP server.
type MCPServerConfig struct {
	Name    string            `json:"name" yaml:"name"`
	Command string            `json:"command" yaml:"command"`
	Args    []string          `json:"args,omitempty" yaml:"args"`
	Env     map[string]string `json:"env,omitempty" yaml:"env"`
	// Mode controls subprocess lifetime when managed via ManagedClient:
	//   "daemon"    — spawn at startup, keep running until shutdown
	//   "on_demand" — kill after IdleTimeout of inactivity, respawn on next call (default)
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`
	// IdleTimeout (only meaningful when Mode == on_demand) is how long
	// the subprocess may sit unused before the lifecycle goroutine
	// kills it. Default 5 minutes.
	IdleTimeout time.Duration `json:"idle_timeout,omitempty" yaml:"idle_timeout,omitempty"`
}

// NewClient spawns an MCP server process and connects via stdio.
func NewClient(cfg MCPServerConfig) (*Client, error) {
	parts := strings.Fields(cfg.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("mcp client: empty command")
	}

	cmdName := parts[0]
	cmdArgs := append(parts[1:], cfg.Args...)

	cmd := exec.Command(cmdName, cmdArgs...)

	// Set environment variables.
	if len(cfg.Env) > 0 {
		env := cmd.Environ()
		for k, v := range cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp client stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp client stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp client start %q: %w", cfg.Command, err)
	}

	c := &Client{
		name:   cfg.Name,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReaderSize(stdout, 256*1024),
	}

	// Read responses in background.
	go c.readLoop()

	// Initialize the MCP session.
	if err := c.initialize(); err != nil {
		c.Close()
		return nil, fmt.Errorf("mcp client initialize: %w", err)
	}

	// Discover tools.
	if err := c.discoverTools(); err != nil {
		slog.Warn("mcp client: could not discover tools", "server", cfg.Name, "error", err)
	}

	return c, nil
}

func (c *Client) readLoop() {
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}

		if resp.ID != nil {
			if ch, ok := c.pending.Load(*resp.ID); ok {
				resultCh := ch.(chan jsonrpcResponse)
				resultCh <- resp
			}
		}
	}
}

func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Register pending response channel.
	resultCh := make(chan jsonrpcResponse, 1)
	c.pending.Store(id, resultCh)
	defer c.pending.Delete(id)

	// Send request.
	c.mu.Lock()
	_, err = c.stdin.Write(append(data, '\n'))
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Wait for response with timeout.
	select {
	case resp := <-resultCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("MCP call timeout: %s", method)
	}
}

func (c *Client) initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.call(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "hera",
			"version": "1.0.0",
		},
	})
	if err != nil {
		return err
	}

	// Send initialized notification (no response expected).
	notif := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "notifications/initialized",
	}
	data, _ := json.Marshal(notif)
	c.mu.Lock()
	_, _ = c.stdin.Write(append(data, '\n'))
	c.mu.Unlock()

	return nil
}

func (c *Client) discoverTools() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return err
	}

	var toolsResp struct {
		Tools []MCPToolDef `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsResp); err != nil {
		return fmt.Errorf("parse tools: %w", err)
	}

	c.Tools = toolsResp.Tools
	slog.Info("mcp tools discovered", "server", c.name, "count", len(c.Tools))
	return nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	var argsMap map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &argsMap); err != nil {
			argsMap = map[string]any{"input": string(args)}
		}
	}

	result, err := c.call(ctx, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": argsMap,
	})
	if err != nil {
		return "", err
	}

	// Parse MCP tool result: {"content": [{"type": "text", "text": "..."}]}
	var toolResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &toolResult); err != nil {
		return string(result), nil
	}

	var texts []string
	for _, c := range toolResult.Content {
		if c.Text != "" {
			texts = append(texts, c.Text)
		}
	}

	return strings.Join(texts, "\n"), nil
}

// Name returns the server name.
func (c *Client) Name() string { return c.name }

// Close stops the MCP server process.
func (c *Client) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	return nil
}
