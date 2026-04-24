package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerTool_Name(t *testing.T) {
	tool := &DockerTool{}
	assert.Equal(t, "docker", tool.Name())
}

func TestDockerTool_Description(t *testing.T) {
	tool := &DockerTool{}
	assert.Contains(t, tool.Description(), "Docker")
}

func TestDockerTool_InvalidArgs(t *testing.T) {
	tool := &DockerTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestDockerTool_RunRequiresImage(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "run"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "image is required")
}

func TestDockerTool_StopRequiresContainer(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "stop"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "container is required")
}

func TestDockerTool_RmRequiresContainer(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "rm"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "container is required")
}

func TestDockerTool_LogsRequiresContainer(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "logs"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "container is required")
}

func TestDockerTool_ExecRequiresContainerAndCommand(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "exec", Container: "test"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "container and command are required")
}

func TestDockerTool_PullRequiresImage(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "pull"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "image is required")
}

func TestDockerTool_BuildDelegates(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "build"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "shell tool")
}

func TestDockerTool_UnknownAction(t *testing.T) {
	tool := &DockerTool{}
	args, _ := json.Marshal(dockerToolArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestRegisterDocker(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterDocker(registry)
	_, ok := registry.Get("docker")
	assert.True(t, ok)
}
