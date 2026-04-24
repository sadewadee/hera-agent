package agent

import (
	"fmt"
	"testing"
	"time"
)

func TestMetrics_RecordRequest(t *testing.T) {
	m := NewMetrics()

	m.RecordRequest(100*time.Millisecond, nil)
	m.RecordRequest(200*time.Millisecond, nil)

	snap := m.Snapshot()
	if snap.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", snap.TotalRequests)
	}
	if snap.TotalErrors != 0 {
		t.Errorf("TotalErrors = %d, want 0", snap.TotalErrors)
	}
}

func TestMetrics_RecordErrors(t *testing.T) {
	m := NewMetrics()

	m.RecordRequest(100*time.Millisecond, nil)
	m.RecordRequest(100*time.Millisecond, fmt.Errorf("timeout"))

	snap := m.Snapshot()
	if snap.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", snap.TotalErrors)
	}
	if snap.ErrorRate != 0.5 {
		t.Errorf("ErrorRate = %f, want 0.5", snap.ErrorRate)
	}
}

func TestMetrics_TokenTracking(t *testing.T) {
	m := NewMetrics()

	m.RecordTokens(100, 50)
	m.RecordTokens(200, 100)

	snap := m.Snapshot()
	if snap.TotalTokensIn != 300 {
		t.Errorf("TotalTokensIn = %d, want 300", snap.TotalTokensIn)
	}
	if snap.TotalTokensOut != 150 {
		t.Errorf("TotalTokensOut = %d, want 150", snap.TotalTokensOut)
	}
}

func TestMetrics_Sessions(t *testing.T) {
	m := NewMetrics()

	m.SessionStarted()
	m.SessionStarted()
	m.SessionStarted()
	m.SessionEnded()

	snap := m.Snapshot()
	if snap.ActiveSessions != 2 {
		t.Errorf("ActiveSessions = %d, want 2", snap.ActiveSessions)
	}
	if snap.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", snap.TotalSessions)
	}
}

func TestMetrics_ToolCalls(t *testing.T) {
	m := NewMetrics()

	m.RecordToolCall()
	m.RecordToolCall()
	m.RecordToolCall()

	snap := m.Snapshot()
	if snap.TotalToolCalls != 3 {
		t.Errorf("TotalToolCalls = %d, want 3", snap.TotalToolCalls)
	}
}

func TestMetrics_LatencyStats(t *testing.T) {
	m := NewMetrics()

	for i := 0; i < 100; i++ {
		m.RecordRequest(time.Duration(i+1)*time.Millisecond, nil)
	}

	snap := m.Snapshot()
	if snap.AvgLatencyMs == 0 {
		t.Error("AvgLatencyMs should not be zero")
	}
	if snap.MaxLatencyMs == 0 {
		t.Error("MaxLatencyMs should not be zero")
	}
	if snap.P95LatencyMs == 0 {
		t.Error("P95LatencyMs should not be zero")
	}
}
