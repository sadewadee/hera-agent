package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsGuardTool_Name(t *testing.T) {
	tool := &SkillsGuardTool{}
	assert.Equal(t, "skills_guard", tool.Name())
}

func TestSkillsGuardTool_Description(t *testing.T) {
	tool := &SkillsGuardTool{}
	assert.Contains(t, tool.Description(), "safety")
}

func TestSkillsGuardTool_InvalidArgs(t *testing.T) {
	tool := &SkillsGuardTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSkillsGuardTool_SafeContent(t *testing.T) {
	tool := &SkillsGuardTool{}
	args, _ := json.Marshal(skillsGuardArgs{SkillName: "my-skill", Content: "echo hello world"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "passed safety check")
}

func TestSkillsGuardTool_DangerousPatterns(t *testing.T) {
	dangerousContents := []string{
		"rm -rf /",
		"eval(user_input)",
		"exec(malicious)",
		"sudo reboot",
		"> /dev/null",
		"chmod 777 /etc/passwd",
	}

	tool := &SkillsGuardTool{}
	for _, content := range dangerousContents {
		args, _ := json.Marshal(skillsGuardArgs{SkillName: "bad-skill", Content: content})
		result, err := tool.Execute(context.Background(), args)
		require.NoError(t, err, "content: %s", content)
		assert.True(t, result.IsError, "should block dangerous content: %s", content)
		assert.Contains(t, result.Content, "BLOCKED")
	}
}

func TestSkillsGuardTool_EmptyContent(t *testing.T) {
	tool := &SkillsGuardTool{}
	args, _ := json.Marshal(skillsGuardArgs{SkillName: "empty-skill", Content: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestRegisterSkillsGuard(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterSkillsGuard(registry)
	_, ok := registry.Get("skills_guard")
	assert.True(t, ok)
}
