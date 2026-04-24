package environments

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewTrajectoryRecorder(t *testing.T) {
	dir := t.TempDir()
	tr := NewTrajectoryRecorder(dir)

	if tr.outputDir != dir {
		t.Errorf("outputDir = %q, want %q", tr.outputDir, dir)
	}
	if tr.active == nil {
		t.Error("active map should be initialized")
	}
}

func TestTrajectoryRecorder_StartEpisode(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())

	id := tr.StartEpisode("helpfulness", "session-1")
	if id == "" {
		t.Fatal("StartEpisode() returned empty ID")
	}

	active := tr.ListActive()
	if len(active) != 1 {
		t.Fatalf("ListActive() len = %d, want 1", len(active))
	}
	if active[0] != id {
		t.Errorf("active[0] = %q, want %q", active[0], id)
	}
}

func TestTrajectoryRecorder_RecordStep(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())
	id := tr.StartEpisode("test-env", "session-1")

	state := State{SessionID: "session-1", TurnCount: 1}
	action := Action{Type: "message", Content: "hello"}
	reward := 1.0

	tr.RecordStep(id, state, action, reward)

	// End and verify.
	traj, err := tr.EndEpisode(id)
	if err != nil {
		t.Fatalf("EndEpisode() error = %v", err)
	}

	if len(traj.Episodes) != 1 {
		t.Fatalf("Episodes len = %d, want 1", len(traj.Episodes))
	}

	ep := traj.Episodes[0]
	if ep.State.SessionID != "session-1" {
		t.Errorf("episode state SessionID = %q, want 'session-1'", ep.State.SessionID)
	}
	if ep.Action.Type != "message" {
		t.Errorf("episode action Type = %q, want 'message'", ep.Action.Type)
	}
	if ep.Reward != 1.0 {
		t.Errorf("episode reward = %f, want 1.0", ep.Reward)
	}
}

func TestTrajectoryRecorder_RecordStep_InvalidID(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())

	// Recording to a non-existent trajectory should be a no-op (no panic).
	tr.RecordStep("nonexistent", State{}, Action{}, 1.0)
}

func TestTrajectoryRecorder_RecordStep_MultipleSteps(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())
	id := tr.StartEpisode("test-env", "session-1")

	tr.RecordStep(id, State{TurnCount: 1}, Action{Type: "message", Content: "hello"}, 1.0)
	tr.RecordStep(id, State{TurnCount: 2}, Action{Type: "tool_call", ToolName: "search"}, 2.0)
	tr.RecordStep(id, State{TurnCount: 3}, Action{Type: "message", Content: "found it"}, 1.5)

	traj, err := tr.EndEpisode(id)
	if err != nil {
		t.Fatalf("EndEpisode() error = %v", err)
	}

	if len(traj.Episodes) != 3 {
		t.Fatalf("Episodes len = %d, want 3", len(traj.Episodes))
	}
	if traj.TotalReward != 4.5 {
		t.Errorf("TotalReward = %f, want 4.5", traj.TotalReward)
	}
}

func TestTrajectoryRecorder_EndEpisode(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())
	id := tr.StartEpisode("test-env", "session-1")

	tr.RecordStep(id, State{}, Action{Type: "message", Content: "hi"}, 1.0)

	traj, err := tr.EndEpisode(id)
	if err != nil {
		t.Fatalf("EndEpisode() error = %v", err)
	}

	if traj.ID != id {
		t.Errorf("ID = %q, want %q", traj.ID, id)
	}
	if traj.Environment != "test-env" {
		t.Errorf("Environment = %q, want 'test-env'", traj.Environment)
	}
	if traj.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want 'session-1'", traj.SessionID)
	}
	if traj.EndedAt.IsZero() {
		t.Error("EndedAt should be set after EndEpisode")
	}
	if traj.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}

	// Verify removed from active.
	active := tr.ListActive()
	if len(active) != 0 {
		t.Errorf("ListActive() len = %d, want 0 (should be removed)", len(active))
	}
}

func TestTrajectoryRecorder_EndEpisode_NotFound(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())

	_, err := tr.EndEpisode("nonexistent")
	if err == nil {
		t.Fatal("EndEpisode() should fail for nonexistent trajectory")
	}
}

func TestTrajectoryRecorder_EndEpisode_DoubleEnd(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())
	id := tr.StartEpisode("test-env", "session-1")

	_, err := tr.EndEpisode(id)
	if err != nil {
		t.Fatalf("first EndEpisode() error = %v", err)
	}

	_, err = tr.EndEpisode(id)
	if err == nil {
		t.Fatal("second EndEpisode() should fail (already ended)")
	}
}

func TestTrajectoryRecorder_SaveToFile(t *testing.T) {
	dir := t.TempDir()
	tr := NewTrajectoryRecorder(dir)
	id := tr.StartEpisode("helpfulness", "session-1")

	tr.RecordStep(id, State{SessionID: "session-1", TurnCount: 1},
		Action{Type: "message", Content: "A helpful response here."}, 1.0)

	traj, err := tr.EndEpisode(id)
	if err != nil {
		t.Fatalf("EndEpisode() error = %v", err)
	}

	err = tr.SaveToFile(traj)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Verify file exists.
	expectedFilename := "trajectory_helpfulness_" + traj.ID[:8] + ".jsonl"
	path := filepath.Join(dir, expectedFilename)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(data) == 0 {
		t.Fatal("saved file should not be empty")
	}

	// Verify it is valid JSON.
	var loaded Trajectory
	if err := json.Unmarshal(data[:len(data)-1], &loaded); err != nil { // strip trailing newline
		t.Fatalf("saved data is not valid JSON: %v", err)
	}

	if loaded.ID != traj.ID {
		t.Errorf("loaded ID = %q, want %q", loaded.ID, traj.ID)
	}
	if loaded.Environment != "helpfulness" {
		t.Errorf("loaded Environment = %q, want 'helpfulness'", loaded.Environment)
	}
	if len(loaded.Episodes) != 1 {
		t.Errorf("loaded Episodes len = %d, want 1", len(loaded.Episodes))
	}
}

func TestTrajectoryRecorder_SaveToFile_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "sub", "dir")
	tr := NewTrajectoryRecorder(nestedDir)

	id := tr.StartEpisode("test", "s1")
	tr.RecordStep(id, State{}, Action{Type: "message", Content: "test"}, 1.0)
	traj, _ := tr.EndEpisode(id)

	err := tr.SaveToFile(traj)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Verify nested directory was created.
	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("nested dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected nested path to be a directory")
	}
}

func TestTrajectoryRecorder_ListActive(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())

	// Start multiple episodes.
	id1 := tr.StartEpisode("env1", "s1")
	id2 := tr.StartEpisode("env2", "s2")
	id3 := tr.StartEpisode("env3", "s3")

	active := tr.ListActive()
	if len(active) != 3 {
		t.Fatalf("ListActive() len = %d, want 3", len(active))
	}

	// End one.
	tr.EndEpisode(id2)

	active = tr.ListActive()
	if len(active) != 2 {
		t.Fatalf("ListActive() after end len = %d, want 2", len(active))
	}

	// Verify id2 is not in active list.
	for _, a := range active {
		if a == id2 {
			t.Errorf("id2 should not be in active list after EndEpisode")
		}
	}

	_ = id1
	_ = id3
}

func TestTrajectory_TotalReward(t *testing.T) {
	tr := NewTrajectoryRecorder(t.TempDir())
	id := tr.StartEpisode("test", "s1")

	tr.RecordStep(id, State{}, Action{Type: "message", Content: "a"}, 1.0)
	tr.RecordStep(id, State{}, Action{Type: "message", Content: "b"}, 2.0)
	tr.RecordStep(id, State{}, Action{Type: "message", Content: "c"}, 0.5)

	traj, _ := tr.EndEpisode(id)

	expected := 3.5
	if traj.TotalReward != expected {
		t.Errorf("TotalReward = %f, want %f", traj.TotalReward, expected)
	}
}

func TestEpisode_Fields(t *testing.T) {
	ep := Episode{
		State:  State{SessionID: "s1", TurnCount: 3},
		Action: Action{Type: "tool_call", ToolName: "search"},
		Reward: 2.5,
	}

	if ep.State.SessionID != "s1" {
		t.Errorf("State.SessionID = %q, want 's1'", ep.State.SessionID)
	}
	if ep.Action.ToolName != "search" {
		t.Errorf("Action.ToolName = %q, want 'search'", ep.Action.ToolName)
	}
	if ep.Reward != 2.5 {
		t.Errorf("Reward = %f, want 2.5", ep.Reward)
	}
}
