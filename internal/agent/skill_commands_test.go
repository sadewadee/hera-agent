package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewSkillCommandRegistry ---

func TestNewSkillCommandRegistry(t *testing.T) {
	reg := NewSkillCommandRegistry()
	require.NotNil(t, reg)
	assert.NotNil(t, reg.commands)
}

// --- BuildPlanPath ---

func TestBuildPlanPath_EmptyInstruction(t *testing.T) {
	now := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	path := BuildPlanPath("", now)
	assert.Contains(t, path, "conversation-plan")
	assert.Contains(t, path, "2024-03-15_103000")
	assert.True(t, strings.HasPrefix(path, filepath.Join(".hera", "plans")))
}

func TestBuildPlanPath_WithInstruction(t *testing.T) {
	now := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	path := BuildPlanPath("Add user authentication", now)
	assert.Contains(t, path, "add-user-authentication")
}

func TestBuildPlanPath_LongInstruction_Truncated(t *testing.T) {
	now := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	long := "word-" + strings.Repeat("a-b-c-d-e-f-g-h-i-j-k-", 10)
	path := BuildPlanPath(long, now)
	// Slug should be truncated.
	parts := strings.Split(filepath.Base(path), "-")
	assert.LessOrEqual(t, len(parts), 20) // timestamp + up to 8 slug parts
}

func TestBuildPlanPath_MultilineInstruction(t *testing.T) {
	now := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	path := BuildPlanPath("First line\nSecond line", now)
	assert.Contains(t, path, "first-line")
	assert.NotContains(t, path, "second")
}

// --- SkillCommandRegistry.ResolveCommandKey ---

func TestResolveCommandKey_Empty(t *testing.T) {
	reg := NewSkillCommandRegistry()
	assert.Equal(t, "", reg.ResolveCommandKey(""))
}

func TestResolveCommandKey_Found(t *testing.T) {
	reg := NewSkillCommandRegistry()
	reg.commands["/my-skill"] = SkillCommandInfo{Name: "my-skill"}
	assert.Equal(t, "/my-skill", reg.ResolveCommandKey("my-skill"))
}

func TestResolveCommandKey_UnderscoreToHyphen(t *testing.T) {
	reg := NewSkillCommandRegistry()
	reg.commands["/my-skill"] = SkillCommandInfo{Name: "my-skill"}
	assert.Equal(t, "/my-skill", reg.ResolveCommandKey("my_skill"))
}

func TestResolveCommandKey_NotFound(t *testing.T) {
	reg := NewSkillCommandRegistry()
	assert.Equal(t, "", reg.ResolveCommandKey("nonexistent"))
}

// --- GetCommands ---

func TestGetCommands_Empty(t *testing.T) {
	reg := NewSkillCommandRegistry()
	cmds := reg.GetCommands()
	assert.Empty(t, cmds)
}

// --- BuildSkillMessage ---

func TestBuildSkillMessage_NoInstruction(t *testing.T) {
	msg := BuildSkillMessage("tdd", "Always write tests first", "")
	assert.Contains(t, msg, `"tdd" skill`)
	assert.Contains(t, msg, "Always write tests first")
	assert.NotContains(t, msg, "instruction alongside")
}

func TestBuildSkillMessage_WithInstruction(t *testing.T) {
	msg := BuildSkillMessage("tdd", "Always write tests first", "for the auth module")
	assert.Contains(t, msg, `"tdd" skill`)
	assert.Contains(t, msg, "Always write tests first")
	assert.Contains(t, msg, "for the auth module")
}

// --- BuildPreloadedSkillsPrompt ---

func TestBuildPreloadedSkillsPrompt_Empty(t *testing.T) {
	prompt, loaded, missing := BuildPreloadedSkillsPrompt(nil, nil)
	assert.Empty(t, prompt)
	assert.Empty(t, loaded)
	assert.Empty(t, missing)
}

func TestBuildPreloadedSkillsPrompt_Success(t *testing.T) {
	loader := func(id string) (string, string, error) {
		return "skill content for " + id, id, nil
	}
	prompt, loaded, missing := BuildPreloadedSkillsPrompt([]string{"tdd", "debug"}, loader)
	assert.Contains(t, prompt, "skill content for tdd")
	assert.Contains(t, prompt, "skill content for debug")
	assert.Equal(t, []string{"tdd", "debug"}, loaded)
	assert.Empty(t, missing)
}

func TestBuildPreloadedSkillsPrompt_Missing(t *testing.T) {
	loader := func(id string) (string, string, error) {
		return "", "", fmt.Errorf("not found")
	}
	_, loaded, missing := BuildPreloadedSkillsPrompt([]string{"nope"}, loader)
	assert.Empty(t, loaded)
	assert.Equal(t, []string{"nope"}, missing)
}

func TestBuildPreloadedSkillsPrompt_Dedup(t *testing.T) {
	calls := 0
	loader := func(id string) (string, string, error) {
		calls++
		return "content", id, nil
	}
	_, loaded, _ := BuildPreloadedSkillsPrompt([]string{"tdd", "tdd", "tdd"}, loader)
	assert.Equal(t, 1, calls)
	assert.Len(t, loaded, 1)
}

// --- ScanSkillCommands ---

func TestScanSkillCommands_EmptyDir(t *testing.T) {
	reg := NewSkillCommandRegistry()
	tmpDir := t.TempDir()
	cmds := reg.ScanSkillCommands([]string{tmpDir}, nil)
	assert.Empty(t, cmds)
}

func TestScanSkillCommands_FindsSkillMD(t *testing.T) {
	reg := NewSkillCommandRegistry()
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillContent := "---\nname: test-skill\ndescription: A test skill\n---\nSkill body"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))

	cmds := reg.ScanSkillCommands([]string{tmpDir}, nil)
	assert.Contains(t, cmds, "/test-skill")
	assert.Equal(t, "A test skill", cmds["/test-skill"].Description)
}

func TestScanSkillCommands_DisabledSkills(t *testing.T) {
	reg := NewSkillCommandRegistry()
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "disabled-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillContent := "---\nname: disabled-skill\n---\nBody"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))

	disabled := map[string]bool{"disabled-skill": true}
	cmds := reg.ScanSkillCommands([]string{tmpDir}, disabled)
	assert.NotContains(t, cmds, "/disabled-skill")
}

func TestScanSkillCommands_SkipsGitDir(t *testing.T) {
	reg := NewSkillCommandRegistry()
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git", "skill")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "SKILL.md"), []byte("---\nname: hidden\n---\n"), 0644))

	cmds := reg.ScanSkillCommands([]string{tmpDir}, nil)
	assert.NotContains(t, cmds, "/hidden")
}
