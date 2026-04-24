package environments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Episode records a single step in a trajectory: state, action, and reward.
type Episode struct {
	State  State   `json:"state"`
	Action Action  `json:"action"`
	Reward float64 `json:"reward"`
}

// Trajectory records a sequence of state-action-reward triples.
type Trajectory struct {
	ID          string    `json:"id"`
	Environment string    `json:"environment"`
	SessionID   string    `json:"session_id"`
	Episodes    []Episode `json:"episodes"`
	TotalReward float64   `json:"total_reward"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at,omitempty"`
}

// TrajectoryRecorder records trajectories for RL training data generation.
type TrajectoryRecorder struct {
	mu        sync.Mutex
	outputDir string
	active    map[string]*Trajectory // trajectoryID -> trajectory
}

// NewTrajectoryRecorder creates a recorder that saves trajectories to outputDir.
func NewTrajectoryRecorder(outputDir string) *TrajectoryRecorder {
	return &TrajectoryRecorder{
		outputDir: outputDir,
		active:    make(map[string]*Trajectory),
	}
}

// StartEpisode begins a new trajectory for the given environment and session.
// Returns the trajectory ID.
func (tr *TrajectoryRecorder) StartEpisode(env, sessionID string) string {
	t := &Trajectory{
		ID:          uuid.New().String(),
		Environment: env,
		SessionID:   sessionID,
		Episodes:    []Episode{},
		StartedAt:   time.Now(),
	}

	tr.mu.Lock()
	tr.active[t.ID] = t
	tr.mu.Unlock()

	return t.ID
}

// RecordStep appends a state-action-reward step to an active trajectory.
func (tr *TrajectoryRecorder) RecordStep(trajectoryID string, state State, action Action, reward float64) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	t, ok := tr.active[trajectoryID]
	if !ok {
		return
	}

	t.Episodes = append(t.Episodes, Episode{
		State:  state,
		Action: action,
		Reward: reward,
	})
	t.TotalReward += reward
}

// EndEpisode finalizes a trajectory and removes it from the active set.
func (tr *TrajectoryRecorder) EndEpisode(trajectoryID string) (*Trajectory, error) {
	tr.mu.Lock()
	t, ok := tr.active[trajectoryID]
	if !ok {
		tr.mu.Unlock()
		return nil, fmt.Errorf("trajectory not found: %s", trajectoryID)
	}
	delete(tr.active, trajectoryID)
	tr.mu.Unlock()

	t.EndedAt = time.Now()
	return t, nil
}

// SaveToFile writes a trajectory as a single JSONL line to the output directory.
func (tr *TrajectoryRecorder) SaveToFile(t *Trajectory) error {
	if err := os.MkdirAll(tr.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	filename := fmt.Sprintf("trajectory_%s_%s.jsonl", t.Environment, t.ID[:8])
	path := filepath.Join(tr.outputDir, filename)

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal trajectory: %w", err)
	}

	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write trajectory: %w", err)
	}

	return nil
}

// ListActive returns IDs of all active (in-progress) trajectories.
func (tr *TrajectoryRecorder) ListActive() []string {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	ids := make([]string, 0, len(tr.active))
	for id := range tr.active {
		ids = append(ids, id)
	}
	return ids
}
