// Package supervisor provides lifecycle management for a fleet of named agents.
//
// A Supervisor tracks active ManagedAgents, enforces a max-concurrency limit,
// and provides health reporting. It does NOT own goroutines — callers start
// agents via Spawn and stop them via Stop/StopAll. The supervisor tracks state
// transitions (stopped → running → stopped) and prevents over-spawning.
package supervisor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// AgentState describes the lifecycle state of a managed agent.
type AgentState string

const (
	StateStopped AgentState = "stopped"
	StateRunning AgentState = "running"
	StateRestart AgentState = "restarting"
	StateFailed  AgentState = "failed"
)

// ManagedAgent holds the configuration and runtime state of one agent under
// supervisor control.
type ManagedAgent struct {
	// Name is the unique identifier for this agent.
	Name string
	// State is the current lifecycle state.
	State AgentState
	// StartedAt is the time the agent last transitioned to StateRunning.
	StartedAt time.Time
	// StoppedAt is the time the agent last transitioned to StateStopped/StateFailed.
	StoppedAt time.Time
	// Restarts is the cumulative restart count.
	Restarts int
	// cancelFn cancels the agent's context when Stop is called.
	cancelFn context.CancelFunc
}

// AgentFactory is a function that runs an agent within a context. It should
// block until the agent is done or ctx is cancelled.
type AgentFactory func(ctx context.Context, name string) error

// Supervisor manages a named fleet of agents.
type Supervisor struct {
	mu            sync.Mutex
	agents        map[string]*ManagedAgent
	maxConcurrent int
	factory       AgentFactory
	logger        *slog.Logger
}

// Config holds Supervisor construction parameters.
type Config struct {
	// MaxConcurrent is the maximum number of agents that may be in StateRunning
	// at the same time. 0 means unlimited.
	MaxConcurrent int
	// Factory is called to start an agent. Must be non-nil.
	Factory AgentFactory
}

// New creates a Supervisor with the given configuration.
func New(cfg Config) (*Supervisor, error) {
	if cfg.Factory == nil {
		return nil, fmt.Errorf("supervisor: AgentFactory is required")
	}
	return &Supervisor{
		agents:        make(map[string]*ManagedAgent),
		maxConcurrent: cfg.MaxConcurrent,
		factory:       cfg.Factory,
		logger:        slog.Default().With("component", "supervisor"),
	}, nil
}

// Spawn starts an agent named name. Returns an error if:
//   - an agent with that name is already running
//   - the max-concurrent limit would be exceeded
func (s *Supervisor) Spawn(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already running.
	if existing, ok := s.agents[name]; ok && existing.State == StateRunning {
		return fmt.Errorf("supervisor: agent %q is already running", name)
	}

	// Enforce max-concurrent limit.
	if s.maxConcurrent > 0 && s.runningCount() >= s.maxConcurrent {
		return fmt.Errorf("supervisor: max concurrent agents (%d) reached", s.maxConcurrent)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ma := &ManagedAgent{
		Name:      name,
		State:     StateRunning,
		StartedAt: time.Now(),
		cancelFn:  cancel,
	}
	if prev, ok := s.agents[name]; ok {
		ma.Restarts = prev.Restarts
	}
	s.agents[name] = ma

	s.logger.Info("supervisor: spawning agent", "name", name)

	// Run the factory in a separate goroutine; transition state on exit.
	// Capture ma pointer so only this generation's goroutine updates state.
	captured := ma
	go func() {
		err := s.factory(ctx, name)
		s.mu.Lock()
		defer s.mu.Unlock()
		// Only update state if this goroutine's ManagedAgent is still current.
		if current, ok := s.agents[name]; ok && current == captured {
			if err != nil && ctx.Err() == nil {
				current.State = StateFailed
				s.logger.Error("supervisor: agent failed", "name", name, "error", err)
			} else {
				current.State = StateStopped
			}
			current.StoppedAt = time.Now()
		}
	}()

	return nil
}

// Stop cancels the named agent's context. It does not wait for the goroutine
// to finish — callers may poll State() or call Wait() if needed.
func (s *Supervisor) Stop(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ma, ok := s.agents[name]
	if !ok {
		return fmt.Errorf("supervisor: agent %q not found", name)
	}
	if ma.State != StateRunning {
		return fmt.Errorf("supervisor: agent %q is not running (state=%s)", name, ma.State)
	}
	s.logger.Info("supervisor: stopping agent", "name", name)
	ma.cancelFn()
	ma.State = StateStopped
	ma.StoppedAt = time.Now()
	return nil
}

// Restart stops and then immediately respawns the named agent. The restart
// counter is incremented.
func (s *Supervisor) Restart(name string) error {
	s.mu.Lock()
	ma, ok := s.agents[name]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("supervisor: agent %q not found", name)
	}
	if ma.State == StateRunning && ma.cancelFn != nil {
		ma.cancelFn()
	}
	ma.State = StateStopped
	ma.Restarts++
	s.mu.Unlock()

	return s.Spawn(name)
}

// StopAll stops all running agents. It iterates a snapshot to avoid deadlock.
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	names := make([]string, 0, len(s.agents))
	for name, ma := range s.agents {
		if ma.State == StateRunning {
			names = append(names, name)
		}
	}
	s.mu.Unlock()

	for _, name := range names {
		if err := s.Stop(name); err != nil {
			s.logger.Warn("supervisor: stop error", "name", name, "error", err)
		}
	}
}

// Status returns a snapshot of all managed agents.
func (s *Supervisor) Status() []ManagedAgent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ManagedAgent, 0, len(s.agents))
	for _, ma := range s.agents {
		out = append(out, *ma)
	}
	return out
}

// Health returns a summary suitable for the /supervisor/status HTTP endpoint.
type Health struct {
	Running int           `json:"running"`
	Stopped int           `json:"stopped"`
	Failed  int           `json:"failed"`
	Agents  []AgentHealth `json:"agents"`
}

// AgentHealth is the per-agent summary in the Health report.
type AgentHealth struct {
	Name      string     `json:"name"`
	State     AgentState `json:"state"`
	Restarts  int        `json:"restarts"`
	StartedAt time.Time  `json:"started_at,omitempty"`
}

// HealthReport builds a Health snapshot from current state.
func (s *Supervisor) HealthReport() Health {
	s.mu.Lock()
	defer s.mu.Unlock()

	h := Health{Agents: make([]AgentHealth, 0, len(s.agents))}
	for _, ma := range s.agents {
		switch ma.State {
		case StateRunning:
			h.Running++
		case StateFailed:
			h.Failed++
		default:
			h.Stopped++
		}
		h.Agents = append(h.Agents, AgentHealth{
			Name:      ma.Name,
			State:     ma.State,
			Restarts:  ma.Restarts,
			StartedAt: ma.StartedAt,
		})
	}
	return h
}

// runningCount returns the number of agents in StateRunning. Must be called
// with s.mu held.
func (s *Supervisor) runningCount() int {
	count := 0
	for _, ma := range s.agents {
		if ma.State == StateRunning {
			count++
		}
	}
	return count
}
