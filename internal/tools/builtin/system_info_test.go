package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemInfoTool_Name(t *testing.T) {
	tool := &SystemInfoTool{}
	assert.Equal(t, "system_info", tool.Name())
}

func TestSystemInfoTool_Description(t *testing.T) {
	tool := &SystemInfoTool{}
	assert.Contains(t, tool.Description(), "system information")
}

func TestSystemInfoTool_AllCategories(t *testing.T) {
	tool := &SystemInfoTool{}
	args, _ := json.Marshal(systemInfoArgs{Category: "all"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Operating System")
	assert.Contains(t, result.Content, "CPU")
	assert.Contains(t, result.Content, "Memory")
	assert.Contains(t, result.Content, "Go Runtime")
}

func TestSystemInfoTool_OSCategory(t *testing.T) {
	tool := &SystemInfoTool{}
	args, _ := json.Marshal(systemInfoArgs{Category: "os"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Operating System")
	assert.NotContains(t, result.Content, "=== CPU ===")
}

func TestSystemInfoTool_CPUCategory(t *testing.T) {
	tool := &SystemInfoTool{}
	args, _ := json.Marshal(systemInfoArgs{Category: "cpu"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Logical CPUs")
}

func TestSystemInfoTool_MemoryCategory(t *testing.T) {
	tool := &SystemInfoTool{}
	args, _ := json.Marshal(systemInfoArgs{Category: "memory"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "HeapAlloc")
}

func TestSystemInfoTool_RuntimeCategory(t *testing.T) {
	tool := &SystemInfoTool{}
	args, _ := json.Marshal(systemInfoArgs{Category: "runtime"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Go Runtime")
	assert.Contains(t, result.Content, "Goroutines")
}

func TestSystemInfoTool_DefaultCategory(t *testing.T) {
	tool := &SystemInfoTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Operating System")
}

func TestSystemInfoTool_EmptyArgs(t *testing.T) {
	tool := &SystemInfoTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
	assert.Equal(t, "512 B", formatBytes(512))
	assert.Contains(t, formatBytes(1024), "KB")
	assert.Contains(t, formatBytes(1048576), "MB")
}

func TestRegisterSystemInfo(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterSystemInfo(registry)
	_, ok := registry.Get("system_info")
	assert.True(t, ok)
}
