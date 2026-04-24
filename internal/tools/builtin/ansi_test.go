package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestANSITool_Name(t *testing.T) {
	tool := &ANSITool{}
	assert.Equal(t, "ansi", tool.Name())
}

func TestANSITool_Description(t *testing.T) {
	tool := &ANSITool{}
	assert.Contains(t, tool.Description(), "ANSI")
}

func TestANSITool_InvalidArgs(t *testing.T) {
	tool := &ANSITool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestANSITool_BoldStyle(t *testing.T) {
	tool := &ANSITool{}
	args, _ := json.Marshal(ansiArgs{Text: "hello", Style: "bold"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "\033[1m")
	assert.Contains(t, result.Content, "hello")
	assert.Contains(t, result.Content, "\033[0m")
}

func TestANSITool_UnderlineStyle(t *testing.T) {
	tool := &ANSITool{}
	args, _ := json.Marshal(ansiArgs{Text: "test", Style: "underline"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "\033[4m")
}

func TestANSITool_NoStyle(t *testing.T) {
	tool := &ANSITool{}
	args, _ := json.Marshal(ansiArgs{Text: "plain"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "\033[0m")
	assert.Contains(t, result.Content, "plain")
}

func TestANSITool_AllStyles(t *testing.T) {
	styles := []string{"bold", "dim", "italic", "underline", "blink", "reverse", "hidden", "strikethrough"}
	expectedCodes := []string{"1", "2", "3", "4", "5", "7", "8", "9"}

	tool := &ANSITool{}
	for i, style := range styles {
		args, _ := json.Marshal(ansiArgs{Text: "x", Style: style})
		result, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		assert.Contains(t, result.Content, "\033["+expectedCodes[i]+"m", "style: %s", style)
	}
}

func TestRegisterANSI(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterANSI(registry)
	_, ok := registry.Get("ansi")
	assert.True(t, ok)
}
