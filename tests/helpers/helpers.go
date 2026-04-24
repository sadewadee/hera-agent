// Package testhelpers provides shared test utilities, mock objects,
// and configuration builders for integration and e2e tests.
package testhelpers

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// --- Test Configuration Builders ---

// TestConfig returns a minimal Config suitable for testing.
func TestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Agent: config.AgentConfig{
			DefaultProvider:     "test",
			DefaultModel:        "test-model",
			Personality:         "helpful test assistant",
			MaxToolCalls:        5,
			MemoryNudgeInterval: 10,
			SkillNudgeInterval:  15,
			Compression: config.CompressionConfig{
				Enabled:        false,
				Threshold:      0.5,
				TargetRatio:    0.2,
				ProtectedTurns: 3,
			},
		},
		Memory: config.MemoryConfig{
			Provider:   "sqlite",
			DBPath:     filepath.Join(t.TempDir(), "test-memory.db"),
			MaxResults: 10,
		},
		Gateway: config.GatewayConfig{
			SessionTimeout: 30,
			HumanDelay:     false,
			DelayMsPerChar: 0,
			AllowAll:       true,
		},
		Security: config.SecurityConfig{
			RedactPII:      false,
			ProtectedPaths: []string{},
		},
		Cron: config.CronConfig{
			Enabled: false,
		},
	}
}

// TestConfigWithCompression returns a Config with compression enabled.
func TestConfigWithCompression(t *testing.T) *config.Config {
	t.Helper()
	cfg := TestConfig(t)
	cfg.Agent.Compression.Enabled = true
	cfg.Agent.Compression.Threshold = 0.5
	cfg.Agent.Compression.ProtectedTurns = 2
	return cfg
}

// --- SQLite Memory Helpers ---

// NewTestMemory creates a SQLite memory provider in a temp directory.
// Returns the provider and a cleanup function.
func NewTestMemory(t *testing.T) *memory.SQLiteProvider {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test-memory.db")
	provider, err := memory.NewSQLiteProvider(dbPath)
	if err != nil {
		t.Fatalf("failed to create test memory: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })
	return provider
}

// NewTestMemoryManager creates a memory.Manager with a fake summarizer.
func NewTestMemoryManager(t *testing.T) *memory.Manager {
	t.Helper()
	provider := NewTestMemory(t)
	summarizer := &FakeSummarizer{Result: "Test summary of the conversation."}
	return memory.NewManager(provider, summarizer)
}

// --- Gateway Helpers ---

// NewTestGateway creates a Gateway with short timeouts for testing.
func NewTestGateway(t *testing.T) *gateway.Gateway {
	t.Helper()
	gw := gateway.NewGateway(gateway.GatewayOptions{
		SessionTimeout:    5 * time.Minute,
		HealthInterval:    1 * time.Second,
		ReconnectBaseWait: 100 * time.Millisecond,
		ReconnectMaxWait:  500 * time.Millisecond,
		MaxReconnects:     3,
	})
	return gw
}

// --- Tool Helpers ---

// EchoTool is a simple tool that echoes its input for testing.
type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string  { return "Echoes the input back" }
func (t *EchoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`)
}
func (t *EchoTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var input struct{ Text string `json:"text"` }
	if err := json.Unmarshal(args, &input); err != nil {
		return &tools.Result{Content: "invalid input", IsError: true}, nil
	}
	return &tools.Result{Content: input.Text}, nil
}

// FailTool is a tool that always returns an error for testing.
type FailTool struct{}

func (t *FailTool) Name() string        { return "fail" }
func (t *FailTool) Description() string  { return "Always fails" }
func (t *FailTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}
func (t *FailTool) Execute(_ context.Context, _ json.RawMessage) (*tools.Result, error) {
	return &tools.Result{Content: "intentional error", IsError: true}, nil
}

// NewTestToolRegistry creates a registry with echo and fail tools.
func NewTestToolRegistry(t *testing.T) *tools.Registry {
	t.Helper()
	reg := tools.NewRegistry()
	reg.Register(&EchoTool{})
	reg.Register(&FailTool{})
	return reg
}

// --- LLM Helpers ---

// MakeMessages builds a conversation slice from role/content pairs.
// Usage: MakeMessages("user", "hello", "assistant", "hi there")
func MakeMessages(pairs ...string) []llm.Message {
	if len(pairs)%2 != 0 {
		panic("MakeMessages requires even number of args (role, content pairs)")
	}
	msgs := make([]llm.Message, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		msgs = append(msgs, llm.Message{
			Role:    llm.Role(pairs[i]),
			Content: pairs[i+1],
		})
	}
	return msgs
}

// --- Context Helpers ---

// TestContext returns a context with a 10-second timeout.
func TestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}
