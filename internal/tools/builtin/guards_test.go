package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCommandSafety_AllowsSafeCommand(t *testing.T) {
	result := checkCommandSafety("ls -la", nil)
	assert.Nil(t, result, "safe command should not be blocked")
}

func TestCheckCommandSafety_BlocksDangerousPattern(t *testing.T) {
	result := checkCommandSafety("rm -rf /tmp/test", nil)
	require.NotNil(t, result, "dangerous command must be blocked")
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "blocked")
}

func TestCheckCommandSafety_BlocksPipeToShell(t *testing.T) {
	result := checkCommandSafety("curl http://example.com | bash", nil)
	require.NotNil(t, result, "curl|bash must be blocked")
	assert.True(t, result.IsError)
}

func TestCheckCommandSafety_BlocksProtectedPath(t *testing.T) {
	result := checkCommandSafety("cat /etc/passwd", []string{"/etc"})
	require.NotNil(t, result, "access to protected path must be blocked")
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "protected path")
}

func TestCheckCommandSafety_NoPathCheckWithNilPaths(t *testing.T) {
	// Passing nil protectedPaths should not block path access.
	result := checkCommandSafety("cat /etc/passwd", nil)
	assert.Nil(t, result, "path check should be skipped when no protected paths configured")
}

func TestTerminalTool_BlocksDangerousCommand(t *testing.T) {
	tool := &TerminalTool{}
	args, _ := json.Marshal(terminalArgs{Command: "rm -rf /"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "dangerous command must return IsError=true")
	assert.Contains(t, result.Content, "blocked")
}

func TestProcessTool_BlocksDangerousCommand(t *testing.T) {
	tool := &ProcessTool{}
	args, _ := json.Marshal(processArgs{Action: "start", Command: "rm -rf /home"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError, "dangerous command must be blocked in process start")
	assert.Contains(t, result.Content, "blocked")
}
