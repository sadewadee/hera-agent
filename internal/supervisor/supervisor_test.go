package supervisor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeBlockingFactory returns an AgentFactory whose goroutine blocks until ctx
// is cancelled.
func makeBlockingFactory() (AgentFactory, chan string) {
	started := make(chan string, 4)
	factory := func(ctx context.Context, name string) error {
		started <- name
		<-ctx.Done()
		return nil
	}
	return factory, started
}

// makeImmediateFactory returns an AgentFactory that finishes immediately.
func makeImmediateFactory() AgentFactory {
	return func(_ context.Context, name string) error { return nil }
}

// makeFailingFactory returns an AgentFactory that returns an error immediately.
func makeFailingFactory(err error) AgentFactory {
	return func(_ context.Context, name string) error { return err }
}

func TestSupervisor_New_NilFactory(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AgentFactory is required")
}

func TestSupervisor_Spawn_And_Status(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory})
	require.NoError(t, err)
	defer sup.StopAll()

	require.NoError(t, sup.Spawn("agent-1"))

	// Wait for factory to confirm it started.
	select {
	case name := <-started:
		assert.Equal(t, "agent-1", name)
	case <-time.After(time.Second):
		t.Fatal("agent-1 did not start")
	}

	status := sup.Status()
	require.Len(t, status, 1)
	assert.Equal(t, StateRunning, status[0].State)
}

func TestSupervisor_Spawn_AlreadyRunning(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory})
	require.NoError(t, err)
	defer sup.StopAll()

	require.NoError(t, sup.Spawn("agent-1"))
	<-started // wait for it to start

	err = sup.Spawn("agent-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestSupervisor_MaxConcurrent(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory, MaxConcurrent: 2})
	require.NoError(t, err)
	defer sup.StopAll()

	require.NoError(t, sup.Spawn("a1"))
	require.NoError(t, sup.Spawn("a2"))
	<-started
	<-started

	err = sup.Spawn("a3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max concurrent agents")
}

func TestSupervisor_Stop(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory})
	require.NoError(t, err)

	require.NoError(t, sup.Spawn("agent-1"))
	<-started

	require.NoError(t, sup.Stop("agent-1"))

	// State should transition to stopped quickly.
	deadline := time.After(2 * time.Second)
	for {
		status := sup.Status()
		if len(status) > 0 && status[0].State == StateStopped {
			break
		}
		select {
		case <-deadline:
			t.Fatal("agent-1 did not stop within timeout")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSupervisor_Stop_NotFound(t *testing.T) {
	sup, _ := New(Config{Factory: makeImmediateFactory()})
	err := sup.Stop("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSupervisor_Restart_IncrementsCounter(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory})
	require.NoError(t, err)
	defer sup.StopAll()

	require.NoError(t, sup.Spawn("agent-1"))
	<-started // confirm first spawn running

	require.NoError(t, sup.Restart("agent-1"))
	<-started // confirm second spawn running

	// Poll briefly to let the goroutine settle into StateRunning.
	deadline := time.After(2 * time.Second)
	for {
		status := sup.Status()
		if len(status) > 0 && status[0].State == StateRunning && status[0].Restarts == 1 {
			break
		}
		select {
		case <-deadline:
			status = sup.Status()
			if len(status) > 0 {
				t.Fatalf("expected running+1 restart, got state=%s restarts=%d", status[0].State, status[0].Restarts)
			}
			t.Fatal("no status available after restart")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSupervisor_FailedAgent_TransitionsToFailed(t *testing.T) {
	errBoom := errors.New("boom")
	sup, err := New(Config{Factory: makeFailingFactory(errBoom)})
	require.NoError(t, err)

	require.NoError(t, sup.Spawn("failing"))

	// Poll for failed state.
	deadline := time.After(2 * time.Second)
	for {
		status := sup.Status()
		if len(status) > 0 && status[0].State == StateFailed {
			break
		}
		select {
		case <-deadline:
			t.Fatal("agent did not reach failed state within timeout")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSupervisor_HealthReport(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory})
	require.NoError(t, err)
	defer sup.StopAll()

	require.NoError(t, sup.Spawn("running-1"))
	<-started

	h := sup.HealthReport()
	assert.Equal(t, 1, h.Running)
	assert.Equal(t, 0, h.Failed)
	require.Len(t, h.Agents, 1)
	assert.Equal(t, StateRunning, h.Agents[0].State)
}

func TestSupervisor_StopAll(t *testing.T) {
	factory, started := makeBlockingFactory()
	sup, err := New(Config{Factory: factory})
	require.NoError(t, err)

	require.NoError(t, sup.Spawn("a1"))
	require.NoError(t, sup.Spawn("a2"))
	<-started
	<-started

	sup.StopAll()

	// All should be stopped.
	deadline := time.After(2 * time.Second)
	for {
		running := 0
		for _, s := range sup.Status() {
			if s.State == StateRunning {
				running++
			}
		}
		if running == 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("not all agents stopped within timeout")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
