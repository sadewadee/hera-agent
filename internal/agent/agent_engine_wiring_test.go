package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/plugins"
)

// spyContextEngine records every lifecycle and usage callback so a test
// can assert that NewAgent wired the engine into SessionManager correctly.
type spyContextEngine struct {
	mu      sync.Mutex
	starts  []string
	ends    []string
	resets  int
	updates []llm.Usage
}

func (e *spyContextEngine) Name() string                                 { return "spy" }
func (e *spyContextEngine) IsAvailable() bool                            { return true }
func (e *spyContextEngine) Initialize(plugins.ContextEngineConfig) error { return nil }
func (e *spyContextEngine) Shutdown()                                    {}
func (e *spyContextEngine) UpdateFromResponse(u llm.Usage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.updates = append(e.updates, u)
}
func (e *spyContextEngine) ShouldCompress(int) bool { return false }
func (e *spyContextEngine) Compress(ctx context.Context, messages []llm.Message, _ int) ([]llm.Message, error) {
	return messages, nil
}
func (e *spyContextEngine) ShouldCompressPreflight([]llm.Message) bool { return false }
func (e *spyContextEngine) OnSessionStart(id, _, _ string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.starts = append(e.starts, id)
	return nil
}
func (e *spyContextEngine) OnSessionEnd(id string, _ []llm.Message) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ends = append(e.ends, id)
	return nil
}
func (e *spyContextEngine) OnSessionReset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.resets++
}
func (e *spyContextEngine) GetToolSchemas() []llm.ToolDef { return nil }
func (e *spyContextEngine) HandleToolCall(context.Context, string, json.RawMessage) (string, error) {
	return "", nil
}
func (e *spyContextEngine) Status() plugins.ContextEngineStatus {
	return plugins.ContextEngineStatus{Name: "spy"}
}
func (e *spyContextEngine) UpdateModel(string, int, string, string, string) error { return nil }

func TestNewAgent_WiresContextEngineLifecycle(t *testing.T) {
	t.Parallel()

	eng := &spyContextEngine{}
	sm := NewSessionManager(30 * time.Minute)
	cfg := &config.Config{}

	ag, err := NewAgent(AgentDeps{
		LLM:           &mockLLMProvider{},
		Sessions:      sm,
		Config:        cfg,
		ContextEngine: eng,
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if ag.ContextEngine() != eng {
		t.Errorf("Agent.ContextEngine() = %v, want spy", ag.ContextEngine())
	}

	// Creating a session should fire OnSessionStart exactly once.
	s := sm.GetOrCreate("cli", "alice")
	eng.mu.Lock()
	gotStarts := append([]string(nil), eng.starts...)
	eng.mu.Unlock()

	if len(gotStarts) != 1 || gotStarts[0] != s.ID {
		t.Errorf("OnSessionStart calls = %v, want [%s]", gotStarts, s.ID)
	}

	// Deleting the session should fire OnSessionEnd.
	sm.Delete(s.ID)
	eng.mu.Lock()
	gotEnds := append([]string(nil), eng.ends...)
	eng.mu.Unlock()

	if len(gotEnds) != 1 || gotEnds[0] != s.ID {
		t.Errorf("OnSessionEnd calls = %v, want [%s]", gotEnds, s.ID)
	}
}

// spySidecar counts Initialize + OnSessionEnd so a test can assert that
// NewAgent wires the memory sidecar into SessionManager lifecycle.
type spySidecar struct {
	mu            sync.Mutex
	initCalls     []string
	endCalls      int
	shutdownCalls int
}

func (p *spySidecar) Name() string      { return "spy-sidecar" }
func (p *spySidecar) IsAvailable() bool { return true }
func (p *spySidecar) Initialize(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.initCalls = append(p.initCalls, sessionID)
	return nil
}
func (p *spySidecar) SystemPromptBlock() string                     { return "" }
func (p *spySidecar) Prefetch(string, string) string                { return "" }
func (p *spySidecar) SyncTurn(string, string, string)               {}
func (p *spySidecar) OnMemoryWrite(string, string, string)          {}
func (p *spySidecar) OnPreCompress([]map[string]interface{}) string { return "" }
func (p *spySidecar) OnSessionEnd([]map[string]interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.endCalls++
}
func (p *spySidecar) GetToolSchemas() []plugins.ToolSchema { return nil }
func (p *spySidecar) HandleToolCall(string, map[string]interface{}) (string, error) {
	return "", nil
}
func (p *spySidecar) GetConfigSchema() []plugins.ConfigField { return nil }
func (p *spySidecar) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shutdownCalls++
}

func TestNewAgent_WiresMemorySidecarLifecycle(t *testing.T) {
	t.Parallel()

	sidecar := &spySidecar{}
	sm := NewSessionManager(30 * time.Minute)
	cfg := &config.Config{}

	_, err := NewAgent(AgentDeps{
		LLM:           &mockLLMProvider{},
		Sessions:      sm,
		Config:        cfg,
		MemorySidecar: sidecar,
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}

	s := sm.GetOrCreate("cli", "alice")

	sidecar.mu.Lock()
	gotInits := append([]string(nil), sidecar.initCalls...)
	sidecar.mu.Unlock()

	if len(gotInits) != 1 || gotInits[0] != s.ID {
		t.Errorf("Sidecar.Initialize calls = %v, want [%s]", gotInits, s.ID)
	}

	sm.Delete(s.ID)
	sidecar.mu.Lock()
	gotEnds := sidecar.endCalls
	sidecar.mu.Unlock()

	if gotEnds != 1 {
		t.Errorf("Sidecar.OnSessionEnd count = %d, want 1", gotEnds)
	}
}

func TestNewAgent_NoEngineLeavesLifecycleUnwired(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(30 * time.Minute)
	cfg := &config.Config{}

	_, err := NewAgent(AgentDeps{
		LLM:      &mockLLMProvider{},
		Sessions: sm,
		Config:   cfg,
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}

	// Without a context engine (and no compression enabled in cfg), lifecycle
	// should remain nil so plain session ops stay a no-op code path.
	sm.mu.RLock()
	lc := sm.lifecycle
	sm.mu.RUnlock()

	if lc != nil {
		t.Errorf("SessionManager.lifecycle = %v, want nil when no engine wired", lc)
	}
}
