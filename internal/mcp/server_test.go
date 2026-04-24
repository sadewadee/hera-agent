package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// sendAndReceive sends a JSON-RPC request via the server's stdin pipe and reads
// the response from stdout. It runs the server in a goroutine and cancels after
// the first response line.
func sendAndReceive(t *testing.T, srv *Server, request any) JSONRPCResponse {
	t.Helper()

	reqBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	reqBytes = append(reqBytes, '\n')

	reader := bytes.NewReader(reqBytes)
	var writer bytes.Buffer
	srv.SetIO(reader, &writer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run blocks until reader is exhausted (one line).
	if err := srv.Run(ctx); err != nil && err != context.Canceled {
		t.Fatalf("Run() error = %v", err)
	}

	// Parse the first line of output as a response.
	output := writer.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("no response written by server")
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (raw: %s)", err, lines[0])
	}
	return resp
}

func TestServer_Initialize(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")

	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	})

	if resp.Error != nil {
		t.Fatalf("initialize returned error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want %q", result["protocolVersion"], "2024-11-05")
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("serverInfo is not a map: %T", result["serverInfo"])
	}
	if serverInfo["name"] != "test-server" {
		t.Errorf("serverInfo.name = %v, want %q", serverInfo["name"], "test-server")
	}
	if serverInfo["version"] != "1.0.0" {
		t.Errorf("serverInfo.version = %v, want %q", serverInfo["version"], "1.0.0")
	}
}

func TestServer_ToolsList(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	srv.RegisterTool(ToolSchema{
		Name:        "greet",
		Description: "Greet a person",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		return "hello", nil
	})

	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	})

	if resp.Error != nil {
		t.Fatalf("tools/list returned error: %v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	toolsList, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools is not a slice: %T", result["tools"])
	}
	if len(toolsList) != 1 {
		t.Fatalf("tools list length = %d, want 1", len(toolsList))
	}
	toolMap, ok := toolsList[0].(map[string]any)
	if !ok {
		t.Fatalf("tool entry is not a map: %T", toolsList[0])
	}
	if toolMap["name"] != "greet" {
		t.Errorf("tool name = %v, want %q", toolMap["name"], "greet")
	}
}

func TestServer_ToolsCall(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	srv.RegisterTool(ToolSchema{
		Name:        "echo",
		Description: "Echo back input",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`),
	}, func(ctx context.Context, params json.RawMessage) (any, error) {
		var args struct {
			Text string `json:"text"`
		}
		json.Unmarshal(params, &args)
		return map[string]string{"echoed": args.Text}, nil
	})

	callParams, _ := json.Marshal(map[string]any{
		"name":      "echo",
		"arguments": map[string]string{"text": "hello world"},
	})

	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params:  callParams,
	})

	if resp.Error != nil {
		t.Fatalf("tools/call returned error: %+v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	content, ok := result["content"].([]any)
	if !ok {
		t.Fatalf("content is not a slice: %T", result["content"])
	}
	if len(content) == 0 {
		t.Fatal("content is empty")
	}
	contentItem, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not a map: %T", content[0])
	}
	if contentItem["type"] != "text" {
		t.Errorf("content[0].type = %v, want %q", contentItem["type"], "text")
	}
}

func TestServer_ToolsCallUnknown(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")

	callParams, _ := json.Marshal(map[string]any{
		"name":      "nonexistent",
		"arguments": map[string]any{},
	})

	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "tools/call",
		Params:  callParams,
	})

	if resp.Error == nil {
		t.Fatal("tools/call for unknown tool should return error")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestServer_Ping(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")

	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`5`),
		Method:  "ping",
	})

	if resp.Error != nil {
		t.Fatalf("ping returned error: %v", resp.Error)
	}
	// Result should be an empty object.
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	if len(result) != 0 {
		t.Errorf("ping result should be empty map, got %v", result)
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")

	resp := sendAndReceive(t, srv, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`6`),
		Method:  "some/unknown/method",
	})

	if resp.Error == nil {
		t.Fatal("unknown method should return error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("error message = %q, want %q", resp.Error.Message, "Method not found")
	}
}
