package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTool_Name(t *testing.T) {
	tool := &ProcessTool{}
	assert.Equal(t, "process", tool.Name())
}

func TestProcessTool_InvalidArgs(t *testing.T) {
	tool := &ProcessTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestProcessTool_List_Empty(t *testing.T) {
	tool := &ProcessTool{}
	args, _ := json.Marshal(processArgs{Action: "list"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "0 active processes")
}

func TestProcessTool_Start(t *testing.T) {
	tool := &ProcessTool{}
	args, _ := json.Marshal(processArgs{Action: "start", Command: "sleep 1"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "started")
}

func TestProcessTool_Stop_NotFound(t *testing.T) {
	tool := &ProcessTool{}
	args, _ := json.Marshal(processArgs{Action: "stop", ID: "nonexistent"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestProcessTool_UnknownAction(t *testing.T) {
	tool := &ProcessTool{}
	args, _ := json.Marshal(processArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestProcessTool_WatchPatternsAndStatus(t *testing.T) {
	tool := &ProcessTool{}

	// Start a short process that emits a distinctive line.
	startArgs, _ := json.Marshal(processArgs{
		Action:        "start",
		Command:       `printf 'launching service\nlistening on port 8080\ndone\n'; sleep 0.1`,
		WatchPatterns: []string{"listening on port"},
		NotifyOnDone:  true,
	})
	startRes, err := tool.Execute(context.Background(), startArgs)
	require.NoError(t, err)
	require.False(t, startRes.IsError, "start failed: %s", startRes.Content)
	// Extract proc id from response text ("Process proc-1 started:").
	// We know it's proc-1 because the map was empty.
	id := "proc-1"

	// Poll status briefly until the process has finished.
	var statusContent string
	for i := 0; i < 40; i++ {
		args, _ := json.Marshal(processArgs{Action: "status", ID: id})
		res, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		require.False(t, res.IsError, "status failed: %s", res.Content)
		statusContent = res.Content
		if strings.Contains(statusContent, "finished:") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.Contains(t, statusContent, "pattern matches", "should report pattern matches: %s", statusContent)
	assert.Contains(t, statusContent, "listening on port 8080", "matched line should appear: %s", statusContent)
	assert.Contains(t, statusContent, "exit_code: 0", "should report exit code when notify_on_complete: %s", statusContent)
}

func TestProcessTool_List_ReportsCompletionState(t *testing.T) {
	tool := &ProcessTool{}

	startArgs, _ := json.Marshal(processArgs{
		Action:       "start",
		Command:      `true`,
		NotifyOnDone: true,
	})
	res, err := tool.Execute(context.Background(), startArgs)
	require.NoError(t, err)
	require.False(t, res.IsError)

	// Give the exit goroutine a moment to fire.
	time.Sleep(200 * time.Millisecond)

	listArgs, _ := json.Marshal(processArgs{Action: "list"})
	listRes, err := tool.Execute(context.Background(), listArgs)
	require.NoError(t, err)
	assert.Contains(t, listRes.Content, "[done]", "list should show done state: %s", listRes.Content)
}

func TestProcessTool_Status_NotFound(t *testing.T) {
	tool := &ProcessTool{}
	args, _ := json.Marshal(processArgs{Action: "status", ID: "nope"})
	res, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
