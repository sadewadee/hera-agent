package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sadewadee/hera/internal/tools"
)

// fakeRegistry is a test DelegateTaskRunner.
type fakeRegistry struct {
	resp        string
	err         error
	capturedTo  string
	capturedMsg string
}

func (f *fakeRegistry) DelegateTo(_ context.Context, targetName, prompt string) (string, error) {
	f.capturedTo = targetName
	f.capturedMsg = prompt
	return f.resp, f.err
}

// buildDelegateArgs marshals delegateTaskArgs for test use.
func buildDelegateArgs(agent, task, ctx string) json.RawMessage {
	m := map[string]string{"agent": agent, "task": task}
	if ctx != "" {
		m["context"] = ctx
	}
	b, _ := json.Marshal(m)
	return b
}

func TestDelegateTaskTool_Name(t *testing.T) {
	tool := NewDelegateTaskTool(nil)
	assert.Equal(t, "delegate_task", tool.Name())
}

func TestDelegateTaskTool_Parameters_ValidJSON(t *testing.T) {
	tool := NewDelegateTaskTool(nil)
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(tool.Parameters(), &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestDelegateTaskTool_Execute_Success(t *testing.T) {
	fake := &fakeRegistry{resp: "done building feature"}
	tool := NewDelegateTaskTool(fake)

	args := buildDelegateArgs("coder", "build authentication module", "")
	result, err := tool.Execute(context.Background(), args)

	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "done building feature")
	assert.Equal(t, "coder", fake.capturedTo)
	assert.Equal(t, "build authentication module", fake.capturedMsg)
}

func TestDelegateTaskTool_Execute_WithContext(t *testing.T) {
	fake := &fakeRegistry{resp: "ok"}
	tool := NewDelegateTaskTool(fake)

	args := buildDelegateArgs("coder", "fix the bug", "file: main.go line 42")
	result, err := tool.Execute(context.Background(), args)

	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, fake.capturedMsg, "fix the bug")
	assert.Contains(t, fake.capturedMsg, "Context: file: main.go line 42")
}

func TestDelegateTaskTool_Execute_AgentError(t *testing.T) {
	fake := &fakeRegistry{err: errors.New("target agent unavailable")}
	tool := NewDelegateTaskTool(fake)

	args := buildDelegateArgs("coder", "some task", "")
	result, err := tool.Execute(context.Background(), args)

	require.NoError(t, err, "Execute should not return Go error — errors are in Result")
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "target agent unavailable")
}

func TestDelegateTaskTool_Execute_NilRegistry(t *testing.T) {
	tool := NewDelegateTaskTool(nil)
	args := buildDelegateArgs("coder", "task", "")
	result, err := tool.Execute(context.Background(), args)

	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "no agent registry configured")
}

func TestDelegateTaskTool_Execute_MissingAgentField(t *testing.T) {
	fake := &fakeRegistry{resp: "ok"}
	tool := NewDelegateTaskTool(fake)

	args := json.RawMessage(`{"task":"do something"}`)
	result, err := tool.Execute(context.Background(), args)

	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "agent name is required")
}

func TestDelegateTaskTool_Execute_MissingTaskField(t *testing.T) {
	fake := &fakeRegistry{resp: "ok"}
	tool := NewDelegateTaskTool(fake)

	args := json.RawMessage(`{"agent":"coder"}`)
	result, err := tool.Execute(context.Background(), args)

	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "task is required")
}

func TestDelegateTaskTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewDelegateTaskTool(&fakeRegistry{})
	result, err := tool.Execute(context.Background(), json.RawMessage(`not-json`))

	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid args")
}

func TestRegisterDelegateTask(t *testing.T) {
	reg := tools.NewRegistry()
	fake := &fakeRegistry{resp: "hello"}
	RegisterDelegateTask(reg, fake)

	tool, ok := reg.Get("delegate_task")
	require.True(t, ok)
	assert.Equal(t, "delegate_task", tool.Name())
}
