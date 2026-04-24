package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
)

// SkillGenerator is satisfied by skills.Generator. Declared here to avoid an
// import cycle: builtin -> skills is safe; skills must not import builtin.
type SkillGenerator interface {
	GenerateFromDescription(ctx context.Context, description string) (*skills.Skill, error)
	SaveSkill(skill *skills.Skill, category string) (string, error)
}

// SkillCreateTool creates a new skill markdown file.
// When content is omitted and a generator is wired, the skill body is
// produced by the LLM via skills.Generator.GenerateFromDescription.
type SkillCreateTool struct {
	skillsDir string
	generator SkillGenerator // optional; nil = manual content only
}

type skillCreateArgs struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Triggers      []string `json:"triggers,omitempty"`
	Platforms     []string `json:"platforms,omitempty"`
	RequiresTools []string `json:"requires_tools,omitempty"`
	Content       string   `json:"content"`
}

func (s *SkillCreateTool) Name() string {
	return "skill_create"
}

func (s *SkillCreateTool) Description() string {
	return "Creates a new skill as a markdown file with YAML frontmatter. Skills are reusable instruction sets the agent can invoke."
}

func (s *SkillCreateTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The skill name (used as identifier and filename)."
			},
			"description": {
				"type": "string",
				"description": "A brief description of what the skill does."
			},
			"triggers": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Keywords that trigger this skill."
			},
			"platforms": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Platforms where this skill is available. Empty means all platforms."
			},
			"requires_tools": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Tools required by this skill."
			},
			"content": {
				"type": "string",
				"description": "The skill instructions in markdown format."
			}
		},
		"required": ["name", "description"]
	}`)
}

func (s *SkillCreateTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params skillCreateArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Name == "" || params.Description == "" {
		return &tools.Result{Content: "name and description are required", IsError: true}, nil
	}

	// If content is not provided and a generator is wired, produce the skill
	// body via LLM. This is the auto-generation path (5C wiring).
	if params.Content == "" {
		if s.generator == nil {
			return &tools.Result{Content: "content is required when no skill generator is configured", IsError: true}, nil
		}
		generated, err := s.generator.GenerateFromDescription(ctx, params.Description)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("skill generation failed: %v", err), IsError: true}, nil
		}
		// Merge generated metadata with caller-supplied overrides.
		params.Content = generated.Content
		if len(params.Triggers) == 0 {
			params.Triggers = generated.Triggers
		}
	}

	// Sanitize the filename
	safeName := sanitizeFilename(params.Name)
	filePath := filepath.Join(s.skillsDir, safeName+".md")

	// Check if skill already exists
	if _, err := os.Stat(filePath); err == nil {
		return &tools.Result{
			Content: fmt.Sprintf("skill %q already exists at %s", params.Name, filePath),
			IsError: true,
		}, nil
	}

	// Build the markdown file with YAML frontmatter
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", params.Name))
	sb.WriteString(fmt.Sprintf("description: %s\n", params.Description))

	if len(params.Triggers) > 0 {
		sb.WriteString("triggers:\n")
		for _, t := range params.Triggers {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}

	if len(params.Platforms) > 0 {
		sb.WriteString("platforms:\n")
		for _, p := range params.Platforms {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}

	if len(params.RequiresTools) > 0 {
		sb.WriteString("requires_tools:\n")
		for _, t := range params.RequiresTools {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString(params.Content)
	sb.WriteString("\n")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return &tools.Result{Content: fmt.Sprintf("create directory: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(filePath, []byte(sb.String()), 0o644); err != nil {
		return &tools.Result{Content: fmt.Sprintf("write skill file: %v", err), IsError: true}, nil
	}

	return &tools.Result{Content: fmt.Sprintf("created skill %q at %s", params.Name, filePath)}, nil
}

// sanitizeFilename converts a name to a safe filename.
func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	// Remove characters that are unsafe for filenames
	var safe strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			safe.WriteRune(r)
		}
	}
	return safe.String()
}

// RegisterSkills registers the skill_create tool with the given registry.
// gen may be nil; when non-nil, skill_create calls without a content field
// route to the generator to produce the skill body via LLM.
func RegisterSkills(registry *tools.Registry, skillsDir string, gen SkillGenerator) {
	registry.Register(&SkillCreateTool{skillsDir: skillsDir, generator: gen})
}
