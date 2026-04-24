package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillCreateTool_Name(t *testing.T) {
	tool := &SkillCreateTool{}
	if got := tool.Name(); got != "skill_create" {
		t.Errorf("Name() = %q, want %q", got, "skill_create")
	}
}

func TestSkillCreateTool_Execute_CreatesSkillFile(t *testing.T) {
	dir := t.TempDir()
	tool := &SkillCreateTool{skillsDir: dir}

	args, _ := json.Marshal(skillCreateArgs{
		Name:        "test_skill",
		Description: "A test skill",
		Triggers:    []string{"test", "trial"},
		Content:     "# Test Skill\n\nThis is a test skill.",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}

	// Verify the file was created
	expectedPath := filepath.Join(dir, "test_skill.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "---") {
		t.Error("skill file should contain YAML frontmatter")
	}
	if !strings.Contains(content, "name: test_skill") {
		t.Error("skill file should contain name field")
	}
	if !strings.Contains(content, "description: A test skill") {
		t.Error("skill file should contain description field")
	}
	if !strings.Contains(content, "- test") {
		t.Error("skill file should contain trigger 'test'")
	}
	if !strings.Contains(content, "# Test Skill") {
		t.Error("skill file should contain the skill content")
	}
}

func TestSkillCreateTool_Execute_RejectsExistingSkill(t *testing.T) {
	dir := t.TempDir()
	// Create an existing skill file
	existingPath := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &SkillCreateTool{skillsDir: dir}
	args, _ := json.Marshal(skillCreateArgs{
		Name:        "existing",
		Description: "Should fail",
		Content:     "content",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for existing skill")
	}
	if !strings.Contains(result.Content, "already exists") {
		t.Errorf("error should mention already exists, got: %q", result.Content)
	}
}

func TestSkillCreateTool_Execute_MissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	tool := &SkillCreateTool{skillsDir: dir}

	tests := []struct {
		name string
		args skillCreateArgs
	}{
		{"missing name", skillCreateArgs{Description: "desc", Content: "content"}},
		{"missing description", skillCreateArgs{Name: "name", Content: "content"}},
		{"missing content", skillCreateArgs{Name: "name", Description: "desc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(tt.args)
			result, err := tool.Execute(context.Background(), args)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !result.IsError {
				t.Error("Execute() should return error for missing required fields")
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello_world"},
		{"My Cool Skill!", "my_cool_skill"},
		{"test-skill", "test-skill"},
		{"UPPER_CASE", "upper_case"},
		{"special@#$chars", "specialchars"},
		{"123_numbers", "123_numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
