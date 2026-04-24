package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitTool_Name(t *testing.T) {
	tool := &GitTool{}
	assert.Equal(t, "git", tool.Name())
}

func TestGitTool_Description(t *testing.T) {
	tool := &GitTool{}
	assert.Contains(t, tool.Description(), "Git")
}

func TestGitTool_InvalidArgs(t *testing.T) {
	tool := &GitTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGitTool_CommitRequiresMessage(t *testing.T) {
	tool := &GitTool{}
	args, _ := json.Marshal(gitToolArgs{Action: "commit"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "commit message is required")
}

func TestGitTool_CheckoutRequiresBranch(t *testing.T) {
	tool := &GitTool{}
	args, _ := json.Marshal(gitToolArgs{Action: "checkout"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "branch name is required")
}

func TestGitTool_UnknownAction(t *testing.T) {
	tool := &GitTool{}
	args, _ := json.Marshal(gitToolArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestGitTool_StatusInTempDir(t *testing.T) {
	dir := t.TempDir()
	tool := &GitTool{}
	// Init a git repo in temp dir
	args, _ := json.Marshal(gitToolArgs{Action: "status", Path: dir})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	// Will error since temp dir is not a git repo
	assert.True(t, result.IsError)
}

func TestRegisterGit(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterGit(registry)
	_, ok := registry.Get("git")
	assert.True(t, ok)
}
