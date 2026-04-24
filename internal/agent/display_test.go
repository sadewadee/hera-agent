package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolDisplay_FormatToolCall_Compact(t *testing.T) {
	d := &ToolDisplay{Compact: true}
	result := d.FormatToolCall("shell", map[string]interface{}{"command": "ls -la"})
	assert.Contains(t, result, "shell(")
	assert.Contains(t, result, "command=ls -la")
}

func TestToolDisplay_FormatToolCall_Verbose(t *testing.T) {
	d := &ToolDisplay{Compact: false}
	result := d.FormatToolCall("shell", map[string]interface{}{"command": "ls -la"})
	assert.Contains(t, result, "Tool: shell")
	assert.Contains(t, result, "command: ls -la")
}

func TestToolDisplay_FormatToolCall_EmptyArgs(t *testing.T) {
	d := &ToolDisplay{Compact: true}
	result := d.FormatToolCall("debug", map[string]interface{}{})
	assert.Contains(t, result, "debug(")
}

func TestToolDisplay_FormatToolResult_Success(t *testing.T) {
	d := &ToolDisplay{}
	result := d.FormatToolResult("shell", "output text", false)
	assert.Contains(t, result, "shell:")
	assert.Contains(t, result, "output text")
	assert.NotContains(t, result, "ERROR")
}

func TestToolDisplay_FormatToolResult_Error(t *testing.T) {
	d := &ToolDisplay{}
	result := d.FormatToolResult("shell", "command failed", true)
	assert.Contains(t, result, "ERROR")
	assert.Contains(t, result, "command failed")
}

func TestToolDisplay_FormatToolResult_Truncation(t *testing.T) {
	d := &ToolDisplay{}
	longOutput := make([]byte, 500)
	for i := range longOutput {
		longOutput[i] = 'x'
	}
	result := d.FormatToolResult("shell", string(longOutput), false)
	assert.Contains(t, result, "...")
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		max    int
		expect string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello wo..."},
		{"empty string", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.max)
			assert.Equal(t, tt.expect, result)
		})
	}
}
