package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/llm"
)

// Generator creates new skill files automatically using an LLM.
// It analyzes conversation patterns, user requests, and domain context
// to generate reusable skill markdown files with YAML frontmatter.
type Generator struct {
	llm      llm.Provider
	skillDir string
}

// NewGenerator creates a skill generator that writes to the given directory.
func NewGenerator(provider llm.Provider, skillDir string) *Generator {
	return &Generator{
		llm:      provider,
		skillDir: skillDir,
	}
}

// GenerateFromConversation analyzes a conversation and generates a skill if a
// reusable pattern is detected. Returns the generated skill or nil if no pattern found.
func (g *Generator) GenerateFromConversation(ctx context.Context, messages []llm.Message, category string) (*Skill, error) {
	if len(messages) < 4 {
		return nil, nil // Too few messages to detect a pattern.
	}

	prompt := buildSkillGenPrompt(messages, category)

	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: skillGenSystemPrompt},
			{Role: llm.RoleUser, Content: prompt},
		},
	}

	resp, err := g.llm.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate skill: %w", err)
	}

	content := resp.Message.Content
	if !strings.Contains(content, "---") {
		return nil, nil // LLM decided no skill pattern exists.
	}

	// Parse the generated skill content.
	skill, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse generated skill: %w", err)
	}
	skill.Content = strings.TrimSpace(body)

	if skill.Name == "" {
		return nil, nil
	}

	return skill, nil
}

// GenerateFromDescription creates a skill from a natural language description.
func (g *Generator) GenerateFromDescription(ctx context.Context, description string) (*Skill, error) {
	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: skillGenSystemPrompt},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Create a skill for: %s\n\nGenerate a complete skill markdown file with YAML frontmatter.", description)},
		},
	}

	resp, err := g.llm.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate skill: %w", err)
	}

	content := resp.Message.Content

	// Strip markdown code fences if present.
	content = strings.TrimPrefix(content, "```markdown\n")
	content = strings.TrimPrefix(content, "```yaml\n")
	content = strings.TrimPrefix(content, "```\n")
	content = strings.TrimSuffix(content, "\n```")

	skill, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse generated skill: %w", err)
	}
	skill.Content = strings.TrimSpace(body)

	return skill, nil
}

// SaveSkill writes a skill to disk in the configured skill directory.
func (g *Generator) SaveSkill(skill *Skill, category string) (string, error) {
	dir := filepath.Join(g.skillDir, category)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create skill dir: %w", err)
	}

	filename := strings.ReplaceAll(strings.ToLower(skill.Name), " ", "-")
	filename = strings.ReplaceAll(filename, "_", "-")
	path := filepath.Join(dir, filename+".md")

	// Build file content.
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", skill.Name))
	sb.WriteString(fmt.Sprintf("description: %q\n", skill.Description))
	if len(skill.Triggers) > 0 {
		sb.WriteString("triggers:\n")
		for _, t := range skill.Triggers {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if len(skill.Platforms) > 0 {
		sb.WriteString(fmt.Sprintf("platforms: [%s]\n", strings.Join(skill.Platforms, ", ")))
	} else {
		sb.WriteString("platforms: []\n")
	}
	if len(skill.RequiresTools) > 0 {
		sb.WriteString(fmt.Sprintf("requires_tools: [%s]\n", strings.Join(skill.RequiresTools, ", ")))
	} else {
		sb.WriteString("requires_tools: []\n")
	}
	sb.WriteString(fmt.Sprintf("generated_at: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(skill.Content)
	sb.WriteString("\n")

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write skill: %w", err)
	}

	skill.FilePath = path
	return path, nil
}

func buildSkillGenPrompt(messages []llm.Message, category string) string {
	var sb strings.Builder
	sb.WriteString("Analyze this conversation and determine if there's a reusable pattern worth turning into a skill.\n\n")
	sb.WriteString("Conversation:\n")

	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", m.Role, truncateStr(m.Content, 500)))
	}

	if category != "" {
		sb.WriteString(fmt.Sprintf("\nSuggested category: %s\n", category))
	}

	sb.WriteString("\nIf a reusable pattern exists, generate a complete skill markdown file with YAML frontmatter.")
	sb.WriteString("\nIf no clear pattern exists, respond with just: NO_PATTERN")

	return sb.String()
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

const skillGenSystemPrompt = `You are a skill generator for Hera, a multi-platform AI agent.

Your job is to create reusable skill files in markdown format with YAML frontmatter.

A skill file looks like this:
---
name: skill-name
description: "What this skill does"
triggers:
  - keyword1
  - keyword2
platforms: []
requires_tools: []
---

# Skill Name

## Purpose
What this skill helps with.

## Instructions
Step-by-step instructions for the agent when this skill is activated.

## Examples
Example inputs and expected behaviors.

Rules:
- Name should be lowercase with hyphens (e.g., "web-research", "code-review")
- Description should be one clear sentence
- Triggers should be 2-5 keywords that would activate this skill
- Content should be detailed enough for an AI agent to follow
- Focus on the PATTERN, not the specific conversation details
- If no clear reusable pattern exists, respond with just: NO_PATTERN`
