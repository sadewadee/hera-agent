package acp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetToolKind_KnownTools(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     ToolKind
	}{
		{"read_file", "read_file", ToolKindRead},
		{"write_file", "write_file", ToolKindEdit},
		{"patch", "patch", ToolKindEdit},
		{"search_files", "search_files", ToolKindSearch},
		{"terminal", "terminal", ToolKindExecute},
		{"process", "process", ToolKindExecute},
		{"execute_code", "execute_code", ToolKindExecute},
		{"web_search", "web_search", ToolKindFetch},
		{"web_extract", "web_extract", ToolKindFetch},
		{"browser_navigate", "browser_navigate", ToolKindFetch},
		{"browser_click", "browser_click", ToolKindExecute},
		{"browser_snapshot", "browser_snapshot", ToolKindRead},
		{"_thinking", "_thinking", ToolKindThink},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetToolKind(tt.toolName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetToolKind_Unknown(t *testing.T) {
	got := GetToolKind("some_unknown_tool")
	assert.Equal(t, ToolKindOther, got)
}

func TestBuildToolTitle(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]any
		contains string
	}{
		{
			name:     "terminal",
			toolName: "terminal",
			args:     map[string]any{"command": "ls -la"},
			contains: "ls -la",
		},
		{
			name:     "read_file",
			toolName: "read_file",
			args:     map[string]any{"path": "/tmp/foo.go"},
			contains: "/tmp/foo.go",
		},
		{
			name:     "write_file",
			toolName: "write_file",
			args:     map[string]any{"path": "/tmp/bar.go"},
			contains: "/tmp/bar.go",
		},
		{
			name:     "search_files",
			toolName: "search_files",
			args:     map[string]any{"pattern": "func main"},
			contains: "func main",
		},
		{
			name:     "web_search",
			toolName: "web_search",
			args:     map[string]any{"query": "golang testing"},
			contains: "golang testing",
		},
		{
			name:     "delegate_task",
			toolName: "delegate_task",
			args:     map[string]any{"goal": "fix the bug"},
			contains: "fix the bug",
		},
		{
			name:     "execute_code",
			toolName: "execute_code",
			args:     map[string]any{},
			contains: "execute code",
		},
		{
			name:     "vision_analyze",
			toolName: "vision_analyze",
			args:     map[string]any{"question": "what is this?"},
			contains: "what is this?",
		},
		{
			name:     "unknown_tool",
			toolName: "my_custom_tool",
			args:     map[string]any{},
			contains: "my_custom_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := BuildToolTitle(tt.toolName, tt.args)
			assert.Contains(t, title, tt.contains)
		})
	}
}

func TestBuildToolTitle_LongCommand(t *testing.T) {
	longCmd := "echo " + string(make([]byte, 100))
	title := BuildToolTitle("terminal", map[string]any{"command": longCmd})
	assert.LessOrEqual(t, len(title), 100) // truncated
}

func TestBuildToolTitle_WebExtractMultipleURLs(t *testing.T) {
	args := map[string]any{
		"urls": []any{"https://example.com", "https://other.com"},
	}
	title := BuildToolTitle("web_extract", args)
	assert.Contains(t, title, "example.com")
	assert.Contains(t, title, "+1")
}

func TestBuildToolTitle_WebExtractEmpty(t *testing.T) {
	args := map[string]any{}
	title := BuildToolTitle("web_extract", args)
	assert.Equal(t, "web extract", title)
}

func TestBuildToolStartContent(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]any
		contains string
	}{
		{
			name:     "terminal",
			toolName: "terminal",
			args:     map[string]any{"command": "ls"},
			contains: "$ ls",
		},
		{
			name:     "read_file",
			toolName: "read_file",
			args:     map[string]any{"path": "/tmp/file.go"},
			contains: "/tmp/file.go",
		},
		{
			name:     "search_files",
			toolName: "search_files",
			args:     map[string]any{"pattern": "TODO", "target": "content"},
			contains: "TODO",
		},
		{
			name:     "write_file",
			toolName: "write_file",
			args:     map[string]any{"path": "/tmp/out.go"},
			contains: "/tmp/out.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := BuildToolStartContent(tt.toolName, tt.args)
			assert.Contains(t, content, tt.contains)
		})
	}
}

func TestBuildToolStartContent_PatchReplace(t *testing.T) {
	args := map[string]any{"mode": "replace", "path": "/tmp/foo.go"}
	content := BuildToolStartContent("patch", args)
	assert.Contains(t, content, "/tmp/foo.go")
}

func TestBuildToolStartContent_PatchOther(t *testing.T) {
	args := map[string]any{"mode": "insert", "patch": "some patch content"}
	content := BuildToolStartContent("patch", args)
	assert.Equal(t, "some patch content", content)
}

func TestBuildToolStartContent_DefaultJSON(t *testing.T) {
	args := map[string]any{"key": "value"}
	content := BuildToolStartContent("unknown_tool", args)
	assert.Contains(t, content, "key")
}

func TestBuildToolCompleteContent_Short(t *testing.T) {
	result := "short result"
	got := BuildToolCompleteContent(result)
	assert.Equal(t, result, got)
}

func TestBuildToolCompleteContent_Long(t *testing.T) {
	long := string(make([]byte, 6000))
	got := BuildToolCompleteContent(long)
	assert.Less(t, len(got), 6000)
	assert.Contains(t, got, "truncated")
}

func TestExtractLocations_WithPath(t *testing.T) {
	args := map[string]any{"path": "/tmp/foo.go"}
	locs := ExtractLocations(args)
	assert.Len(t, locs, 1)
	assert.Equal(t, "/tmp/foo.go", locs[0].Path)
}

func TestExtractLocations_WithPathAndLine(t *testing.T) {
	args := map[string]any{"path": "/tmp/foo.go", "line": 42}
	locs := ExtractLocations(args)
	assert.Len(t, locs, 1)
	assert.Equal(t, "/tmp/foo.go", locs[0].Path)
	assert.NotNil(t, locs[0].Line)
	assert.Equal(t, 42, *locs[0].Line)
}

func TestExtractLocations_WithOffset(t *testing.T) {
	args := map[string]any{"path": "/tmp/foo.go", "offset": float64(10)}
	locs := ExtractLocations(args)
	assert.Len(t, locs, 1)
	assert.NotNil(t, locs[0].Line)
	assert.Equal(t, 10, *locs[0].Line)
}

func TestExtractLocations_NoPath(t *testing.T) {
	args := map[string]any{"command": "ls"}
	locs := ExtractLocations(args)
	assert.Empty(t, locs)
}

func TestToolKindConstants(t *testing.T) {
	assert.Equal(t, ToolKind("read"), ToolKindRead)
	assert.Equal(t, ToolKind("edit"), ToolKindEdit)
	assert.Equal(t, ToolKind("search"), ToolKindSearch)
	assert.Equal(t, ToolKind("execute"), ToolKindExecute)
	assert.Equal(t, ToolKind("fetch"), ToolKindFetch)
	assert.Equal(t, ToolKind("think"), ToolKindThink)
	assert.Equal(t, ToolKind("other"), ToolKindOther)
}
