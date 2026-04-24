package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRLTrainingTool_Name(t *testing.T) {
	tool := &RLTrainingTool{}
	assert.Equal(t, "rl_training", tool.Name())
}

func TestRLTrainingTool_Description(t *testing.T) {
	tool := &RLTrainingTool{}
	assert.Contains(t, tool.Description(), "reinforcement learning")
}

func TestRLTrainingTool_InvalidArgs(t *testing.T) {
	tool := &RLTrainingTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRLTrainingTool_StartEpisode(t *testing.T) {
	tool := &RLTrainingTool{}
	args, _ := json.Marshal(rlTrainingArgs{Action: "start_episode"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "episode started")
}

func TestRLTrainingTool_RecordReward(t *testing.T) {
	tool := &RLTrainingTool{}
	args, _ := json.Marshal(rlTrainingArgs{Action: "record_reward", EpisodeID: "ep1", Reward: 0.95})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "0.95")
	assert.Contains(t, result.Content, "ep1")
}

func TestRLTrainingTool_EndEpisode(t *testing.T) {
	tool := &RLTrainingTool{}
	args, _ := json.Marshal(rlTrainingArgs{Action: "end_episode", EpisodeID: "ep1"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "ep1")
	assert.Contains(t, result.Content, "ended")
}

func TestRLTrainingTool_Export(t *testing.T) {
	tool := &RLTrainingTool{}
	args, _ := json.Marshal(rlTrainingArgs{Action: "export"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "exported")
}

func TestRLTrainingTool_UnknownAction(t *testing.T) {
	tool := &RLTrainingTool{}
	args, _ := json.Marshal(rlTrainingArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRegisterRLTraining(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterRLTraining(registry)
	_, ok := registry.Get("rl_training")
	assert.True(t, ok)
}
