package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsSyncTool_Name(t *testing.T) {
	tool := &SkillsSyncTool{}
	assert.Equal(t, "skills_sync", tool.Name())
}

func TestSkillsSyncTool_Description(t *testing.T) {
	tool := &SkillsSyncTool{}
	assert.Contains(t, tool.Description(), "Syncs")
}

func TestSkillsSyncTool_InvalidArgs(t *testing.T) {
	tool := &SkillsSyncTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSkillsSyncTool_Pull(t *testing.T) {
	tool := &SkillsSyncTool{}
	args, _ := json.Marshal(skillsSyncArgs{Action: "pull"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "pull")
}

func TestSkillsSyncTool_Push(t *testing.T) {
	tool := &SkillsSyncTool{}
	args, _ := json.Marshal(skillsSyncArgs{Action: "push"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "push")
}

func TestSkillsSyncTool_Status(t *testing.T) {
	tool := &SkillsSyncTool{}
	args, _ := json.Marshal(skillsSyncArgs{Action: "status"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "up to date")
}

func TestSkillsSyncTool_UnknownAction(t *testing.T) {
	tool := &SkillsSyncTool{}
	args, _ := json.Marshal(skillsSyncArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestRegisterSkillsSync(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterSkillsSync(registry)
	_, ok := registry.Get("skills_sync")
	assert.True(t, ok)
}
