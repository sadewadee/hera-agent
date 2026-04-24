package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchTool_Name(t *testing.T) {
	tool := &PatchTool{}
	assert.Equal(t, "patch", tool.Name())
}

func TestPatchTool_Description(t *testing.T) {
	tool := &PatchTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestPatchTool_InvalidArgs(t *testing.T) {
	tool := &PatchTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestPatchTool_FileNotFound(t *testing.T) {
	tool := &PatchTool{}
	args, _ := json.Marshal(patchArgs{FilePath: "/nonexistent/file.go", Diff: "--- a\n+++ b"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "file not found")
}

func TestPatchTool_ApplyPatch(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.go")
	os.WriteFile(fpath, []byte("original content"), 0644)

	tool := &PatchTool{}
	diff := "--- a/test.go\n+++ b/test.go\n-original\n+modified"
	args, _ := json.Marshal(patchArgs{FilePath: fpath, Diff: diff})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Patch applied")
	assert.Contains(t, result.Content, "4 diff lines")
}
