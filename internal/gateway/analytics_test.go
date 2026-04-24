package gateway

import (
	"testing"
	"time"
)

func TestAnalytics_RecordMessage(t *testing.T) {
	a := NewAnalytics()

	a.RecordMessage("telegram", "user1")
	a.RecordMessage("telegram", "user1")
	a.RecordMessage("discord", "user2")

	snap := a.Snapshot()
	if snap.TotalMessages != 3 {
		t.Errorf("TotalMessages = %d, want 3", snap.TotalMessages)
	}
	if snap.ActivePlatforms != 2 {
		t.Errorf("ActivePlatforms = %d, want 2", snap.ActivePlatforms)
	}
	if snap.ActiveUsers != 2 {
		t.Errorf("ActiveUsers = %d, want 2", snap.ActiveUsers)
	}
}

func TestAnalytics_RecordResponse(t *testing.T) {
	a := NewAnalytics()

	a.RecordMessage("telegram", "user1")
	a.RecordResponse("telegram", "user1", 100*time.Millisecond)
	a.RecordResponse("telegram", "user1", 200*time.Millisecond)

	snap := a.Snapshot()
	if snap.TotalResponses != 2 {
		t.Errorf("TotalResponses = %d, want 2", snap.TotalResponses)
	}
	if snap.AvgResponseMs == 0 {
		t.Error("AvgResponseMs should not be zero")
	}
}

func TestAnalytics_PlatformStats(t *testing.T) {
	a := NewAnalytics()

	a.RecordMessage("telegram", "user1")
	a.RecordMessage("telegram", "user2")
	a.RecordMessage("discord", "user3")

	snap := a.Snapshot()
	telegramStats, ok := snap.PlatformStats["telegram"]
	if !ok {
		t.Fatal("expected telegram platform stats")
	}
	if telegramStats.TotalMessages != 2 {
		t.Errorf("telegram messages = %d, want 2", telegramStats.TotalMessages)
	}
}

func TestAnalytics_SnapshotIsolation(t *testing.T) {
	a := NewAnalytics()

	a.RecordMessage("test", "user1")
	snap1 := a.Snapshot()

	a.RecordMessage("test", "user2")
	snap2 := a.Snapshot()

	if snap1.TotalMessages != 1 {
		t.Errorf("snap1.TotalMessages = %d, want 1", snap1.TotalMessages)
	}
	if snap2.TotalMessages != 2 {
		t.Errorf("snap2.TotalMessages = %d, want 2", snap2.TotalMessages)
	}
}
