package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/llm"
)

func llmUserMessage(text string) llm.Message {
	return llm.Message{Role: llm.RoleUser, Content: text}
}

// eventKind records what fired so tests can assert ordering.
type eventKind int

const (
	evtStart eventKind = iota
	evtEnd
)

type lifecycleRecorder struct {
	mu     sync.Mutex
	events []struct {
		kind eventKind
		id   string
	}
}

func (r *lifecycleRecorder) hook() *SessionLifecycle {
	return &SessionLifecycle{
		OnStart: func(s *Session) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.events = append(r.events, struct {
				kind eventKind
				id   string
			}{evtStart, s.ID})
		},
		OnEnd: func(s *Session) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.events = append(r.events, struct {
				kind eventKind
				id   string
			}{evtEnd, s.ID})
		},
	}
}

func (r *lifecycleRecorder) snapshot() []struct {
	kind eventKind
	id   string
} {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]struct {
		kind eventKind
		id   string
	}, len(r.events))
	copy(out, r.events)
	return out
}

func TestSessionLifecycle_OnStartFiresOnCreate(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	s := sm.Create("cli", "alice")

	events := rec.snapshot()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].kind != evtStart || events[0].id != s.ID {
		t.Errorf("event = %+v, want OnStart for %s", events[0], s.ID)
	}
}

func TestSessionLifecycle_OnStartFiresOnGetOrCreateNew(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	s := sm.GetOrCreate("cli", "bob")

	events := rec.snapshot()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].kind != evtStart || events[0].id != s.ID {
		t.Errorf("event = %+v, want OnStart for %s", events[0], s.ID)
	}
}

func TestSessionLifecycle_OnStartSilentOnReuse(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	sm.Create("cli", "carol") // before hook installed
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	// Reusing existing active session should NOT fire OnStart.
	sm.GetOrCreate("cli", "carol")

	if events := rec.snapshot(); len(events) != 0 {
		t.Errorf("events = %+v, want none on reuse", events)
	}
}

func TestSessionLifecycle_OnEndFiresOnDelete(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	s := sm.Create("cli", "dan")
	sm.Delete(s.ID)

	events := rec.snapshot()
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[1].kind != evtEnd || events[1].id != s.ID {
		t.Errorf("second event = %+v, want OnEnd for %s", events[1], s.ID)
	}
}

func TestSessionLifecycle_OnEndFiresOnCleanExpired(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(1 * time.Millisecond)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	s1 := sm.Create("cli", "eve")
	s2 := sm.Create("cli", "frank")

	time.Sleep(5 * time.Millisecond)
	sm.CleanExpired()

	events := rec.snapshot()
	// Expect 2 OnStart + 2 OnEnd
	if len(events) != 4 {
		t.Fatalf("events = %d (%+v), want 4", len(events), events)
	}
	endIDs := map[string]bool{}
	for _, ev := range events {
		if ev.kind == evtEnd {
			endIDs[ev.id] = true
		}
	}
	if !endIDs[s1.ID] || !endIDs[s2.ID] {
		t.Errorf("OnEnd not fired for both sessions: got %+v", endIDs)
	}
}

func TestSessionLifecycle_OnEndFiresOnGetOrCreateExpiry(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(1 * time.Millisecond)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	first := sm.GetOrCreate("cli", "gina")
	time.Sleep(5 * time.Millisecond)
	second := sm.GetOrCreate("cli", "gina")

	if first.ID == second.ID {
		t.Fatal("expected new session ID after expiry")
	}

	events := rec.snapshot()
	// Expected sequence: Start(first), End(first), Start(second).
	if len(events) != 3 {
		t.Fatalf("events = %+v, want 3", events)
	}
	if events[0].kind != evtStart || events[0].id != first.ID {
		t.Errorf("events[0] = %+v, want OnStart for %s", events[0], first.ID)
	}
	if events[1].kind != evtEnd || events[1].id != first.ID {
		t.Errorf("events[1] = %+v, want OnEnd for %s", events[1], first.ID)
	}
	if events[2].kind != evtStart || events[2].id != second.ID {
		t.Errorf("events[2] = %+v, want OnStart for %s", events[2], second.ID)
	}
}

func TestSessionLifecycle_NilHookIsNoOp(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	// no SetLifecycle call

	s := sm.Create("cli", "harry")
	sm.Delete(s.ID)
	// No panic = pass.
}

func TestSessionLifecycle_OnStartFiresOnBranch(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	parent := sm.Create("cli", "karl")
	branched, err := sm.Branch(parent.ID)
	if err != nil {
		t.Fatalf("Branch: %v", err)
	}

	events := rec.snapshot()
	if len(events) != 2 {
		t.Fatalf("events = %+v, want 2 (parent Create + branch)", events)
	}
	if events[1].kind != evtStart || events[1].id != branched.ID {
		t.Errorf("events[1] = %+v, want OnStart for branched %s", events[1], branched.ID)
	}
}

func TestSessionLifecycle_OnStartFiresOnFork(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	parent := sm.Create("cli", "lena")
	parent.AppendMessage(llmUserMessage("hi"))
	parent.AppendMessage(llmUserMessage("again"))

	forked, err := sm.Fork(parent.ID, 0)
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}

	events := rec.snapshot()
	if len(events) != 2 {
		t.Fatalf("events = %+v, want 2 (parent Create + fork)", events)
	}
	if events[1].kind != evtStart || events[1].id != forked.ID {
		t.Errorf("events[1] = %+v, want OnStart for forked %s", events[1], forked.ID)
	}
}

func TestSessionLifecycle_ForkOutOfRangeDoesNotFire(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	parent := sm.Create("cli", "mira")
	// No messages yet, index 0 is out of range.
	_, err := sm.Fork(parent.ID, 0)
	if err == nil {
		t.Fatal("Fork: expected out-of-range error")
	}

	events := rec.snapshot()
	// Only the parent Create fired; no spurious OnStart for an aborted fork.
	if len(events) != 1 {
		t.Errorf("events = %+v, want 1 (only parent Create)", events)
	}
}

func TestSessionLifecycle_SetLifecycleClearable(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager(30 * time.Minute)
	rec := &lifecycleRecorder{}
	sm.SetLifecycle(rec.hook())

	sm.Create("cli", "ivan")
	sm.SetLifecycle(nil)
	s := sm.Create("cli", "jane")
	sm.Delete(s.ID)

	events := rec.snapshot()
	if len(events) != 1 {
		t.Errorf("events = %+v, want 1 (only the first Create before clearing)", events)
	}
}
