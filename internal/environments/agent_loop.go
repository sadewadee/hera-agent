package environments

import "context"

// AgentLoop runs an agent in a loop environment for benchmarking and evaluation.
type AgentLoop struct {
	MaxSteps    int
	Environment string
	OnStep      func(step int, state State, action Action)
}

// Run executes the agent loop for the configured number of steps.
func (al *AgentLoop) Run(ctx context.Context, stepFn func(ctx context.Context, state State) (Action, float64, error)) ([]Trajectory, error) {
	var trajectories []Trajectory
	recorder := NewTrajectoryRecorder("")
	tid := recorder.StartEpisode(al.Environment, "loop")
	for i := 0; i < al.MaxSteps; i++ {
		select {
		case <-ctx.Done(): break
		default:
		}
		state := State{SessionID: "loop", TurnCount: i}
		action, reward, err := stepFn(ctx, state)
		if err != nil { break }
		recorder.RecordStep(tid, state, action, reward)
		if al.OnStep != nil { al.OnStep(i, state, action) }
	}
	if t, err := recorder.EndEpisode(tid); err == nil {
		trajectories = append(trajectories, *t)
	}
	return trajectories, nil
}
