package builtin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/tools"
)

// fakeAgent is a minimal AgentRunner for integration tests.
type fakeAgent struct {
	response string
}

func (f *fakeAgent) HandleMessage(_ context.Context, _, _, _, text string) (string, error) {
	return f.response + " (input: " + text + ")", nil
}

// agentRegistryAdapter wraps a map to satisfy DelegateTaskRunner.
type agentRegistryAdapter struct {
	agents map[string]*fakeAgent
}

func newAgentRegistryAdapter() *agentRegistryAdapter {
	return &agentRegistryAdapter{agents: make(map[string]*fakeAgent)}
}

func (a *agentRegistryAdapter) register(name string, ag *fakeAgent) {
	a.agents[name] = ag
}

func (a *agentRegistryAdapter) DelegateTo(ctx context.Context, targetName, prompt string) (string, error) {
	ag, ok := a.agents[targetName]
	if !ok {
		return "", nil
	}
	return ag.HandleMessage(ctx, "delegate", targetName, "delegate", prompt)
}

// TestDelegateTask_IntegrationWithAgentBus verifies that:
//  1. delegate_task tool is registered and reachable via tools.Registry.
//  2. Executing it routes the task to the correct named sub-agent.
//  3. AgentBus subscriber on "delegation" topic receives a notification.
func TestDelegateTask_IntegrationWithAgentBus(t *testing.T) {
	// Set up fake sub-agent.
	sub := &fakeAgent{response: "analysis complete"}
	registry := newAgentRegistryAdapter()
	registry.register("analyst", sub)

	// Set up AgentBus and subscribe before the tool runs.
	bus := gateway.NewAgentBus()
	ch := bus.Subscribe("delegation", "test-observer")
	defer bus.Unsubscribe("delegation", "test-observer")

	// Build and register the tool.
	dt := NewDelegateTaskTool(registry).WithObserver(bus)
	toolReg := tools.NewRegistry()
	toolReg.Register(dt)

	// Verify tool is reachable by name (simulates LLM tool call lookup).
	found, ok := toolReg.Get("delegate_task")
	require.True(t, ok, "delegate_task must be reachable in tool registry")
	assert.Equal(t, "delegate_task", found.Name())

	// Simulate LLM invoking the tool.
	args, _ := json.Marshal(map[string]string{
		"agent": "analyst",
		"task":  "analyse market trends for Q1",
	})
	result, err := dt.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "execute must succeed: %s", result.Content)
	assert.Contains(t, result.Content, "analysis complete")
	assert.Contains(t, result.Content, "analyst")

	// Verify AgentBus subscriber received the delegation notification.
	select {
	case msg := <-ch:
		assert.Equal(t, "delegation", msg.Topic)
		assert.Contains(t, msg.Payload, "analyst")
	case <-time.After(2 * time.Second):
		t.Fatal("AgentBus subscriber did not receive delegation event within 2s")
	}
}

// TestDelegateTask_BusSubscriberReceivesOnDelegation verifies that
// PublishDelegation on AgentBus delivers to a subscriber on the "delegation"
// topic even without the full tool pipeline.
func TestDelegateTask_BusSubscriberReceivesOnDelegation(t *testing.T) {
	bus := gateway.NewAgentBus()
	ch := bus.Subscribe("delegation", "watcher")
	defer bus.Unsubscribe("delegation", "watcher")

	bus.PublishDelegation("main", "analyst", "delegated: check sales figures")

	select {
	case msg := <-ch:
		assert.Equal(t, "delegation", msg.Topic)
		assert.Equal(t, "main", msg.From)
		assert.Contains(t, msg.Payload, "analyst")
		assert.Contains(t, msg.Payload, "check sales figures")
	case <-time.After(2 * time.Second):
		t.Fatal("AgentBus subscriber did not receive event")
	}
}

// TestDelegateTask_CallerNameInBusEvent verifies that the From field in the
// AgentBus delegation event reflects the callerName set at construction, not
// the hardcoded "caller" string.
func TestDelegateTask_CallerNameInBusEvent(t *testing.T) {
	sub := &fakeAgent{response: "done"}
	registry := newAgentRegistryAdapter()
	registry.register("worker", sub)

	bus := gateway.NewAgentBus()
	ch := bus.Subscribe("delegation", "tester")
	defer bus.Unsubscribe("delegation", "tester")

	// Build tool with explicit caller name "orchestrator".
	dt := NewDelegateTaskTool(registry).WithCallerName("orchestrator").WithObserver(bus)

	args, _ := json.Marshal(map[string]string{"agent": "worker", "task": "do the thing"})
	result, err := dt.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError, "execute must succeed: %s", result.Content)

	select {
	case msg := <-ch:
		assert.Equal(t, "orchestrator", msg.From, "From must be the callerName, not hardcoded 'caller'")
	case <-time.After(2 * time.Second):
		t.Fatal("AgentBus subscriber did not receive delegation event within 2s")
	}
}
