package agent

import (
	"context"
	"fmt"
	"sync"
)

// AgentRunner is the minimal interface for invoking an agent with a prompt.
// It mirrors the subset of Agent used by delegation — this allows test fakes.
type AgentRunner interface {
	HandleMessage(ctx context.Context, platform, chatID, userID, text string) (string, error)
}

// AgentRegistry holds a named set of AgentRunners. The main agent uses this to
// delegate tasks to specialised sub-agents.
//
// Thread-safe: Register and Get may be called concurrently.
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]AgentRunner
}

// NewAgentRegistry returns an empty, ready-to-use registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{agents: make(map[string]AgentRunner)}
}

// Register adds a named agent to the registry. If a registration already exists
// under the same name, it is replaced.
func (r *AgentRegistry) Register(name string, a AgentRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[name] = a
}

// Get returns the agent registered under name. ok is false if not found.
func (r *AgentRegistry) Get(name string) (AgentRunner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.agents[name]
	return a, ok
}

// Names returns a sorted snapshot of all registered agent names.
func (r *AgentRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.agents))
	for k := range r.agents {
		out = append(out, k)
	}
	return out
}

// DelegateTo invokes the agent registered under targetName with the given prompt.
// It uses a synthetic platform/chat/user identity so the sub-agent gets its own
// memory session isolated from the caller's session.
func (r *AgentRegistry) DelegateTo(ctx context.Context, targetName, prompt string) (string, error) {
	a, ok := r.Get(targetName)
	if !ok {
		return "", fmt.Errorf("agent registry: no agent registered as %q", targetName)
	}
	// Synthetic identity: delegation sessions are keyed by target name + "delegate".
	return a.HandleMessage(ctx, "delegate", targetName, "delegate", prompt)
}
