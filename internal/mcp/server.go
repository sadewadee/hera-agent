package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ToolHandler is the function signature for MCP tool implementations.
type ToolHandler func(ctx context.Context, params json.RawMessage) (any, error)

// ToolSchema describes an MCP tool for the tools/list response.
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Server implements an MCP server using stdio (stdin/stdout) JSON-RPC 2.0.
type Server struct {
	mu      sync.RWMutex
	name    string
	version string
	tools   map[string]ToolHandler
	schemas []ToolSchema
	logger  *slog.Logger
	reader  io.Reader
	writer  io.Writer
	cancel  context.CancelFunc
}

// NewServer creates a new MCP stdio server.
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]ToolHandler),
		schemas: make([]ToolSchema, 0),
		logger:  slog.Default(),
		reader:  os.Stdin,
		writer:  os.Stdout,
	}
}

// SetIO overrides the default stdin/stdout for testing.
func (s *Server) SetIO(r io.Reader, w io.Writer) {
	s.reader = r
	s.writer = w
}

// RegisterTool adds a tool to the server.
func (s *Server) RegisterTool(schema ToolSchema, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[schema.Name] = handler
	s.schemas = append(s.schemas, schema)
}

// Run starts the server, reading JSON-RPC messages from stdin until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	defer cancel()

	scanner := bufio.NewScanner(s.reader)
	// Allow large messages (up to 1MB).
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(nil, -32700, "Parse error", err.Error())
			continue
		}

		resp := s.handleRequest(ctx, &req)
		if resp != nil {
			s.writeResponse(resp)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	return nil
}

// Stop signals the server to shut down.
func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Notification -- no response.
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "ping":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		}
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
				Data:    req.Method,
			},
		}
	}
}

func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    s.name,
				"version": s.version,
			},
		},
	}
}

func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	toolList := make([]map[string]any, 0, len(s.schemas))
	for _, schema := range s.schemas {
		toolList = append(toolList, map[string]any{
			"name":        schema.Name,
			"description": schema.Description,
			"inputSchema": json.RawMessage(schema.InputSchema),
		})
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": toolList,
		},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	s.mu.RLock()
	handler, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Unknown tool",
				Data:    params.Name,
			},
		}
	}

	result, err := handler(ctx, params.Arguments)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
				},
				"isError": true,
			},
		}
	}

	// Wrap result as MCP text content.
	text, _ := json.Marshal(result)
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(text)},
			},
		},
	}
}

func (s *Server) writeResponse(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("marshal response", "error", err)
		return
	}
	data = append(data, '\n')
	_, _ = s.writer.Write(data)
}

func (s *Server) writeError(id json.RawMessage, code int, message, data string) {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.writeResponse(resp)
}
