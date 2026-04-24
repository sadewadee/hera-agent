package agent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	planSlugRe      = regexp.MustCompile(`[^a-z0-9]+`)
	skillInvalidRe  = regexp.MustCompile(`[^a-z0-9-]`)
	skillMultiHypRe = regexp.MustCompile(`-{2,}`)
)

// SkillCommandInfo holds metadata for a skill-based slash command.
type SkillCommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SkillPath   string `json:"skill_md_path"`
	SkillDir    string `json:"skill_dir"`
}

// SkillCommandRegistry manages skill slash commands.
type SkillCommandRegistry struct {
	commands map[string]SkillCommandInfo
}

// NewSkillCommandRegistry creates a new empty registry.
func NewSkillCommandRegistry() *SkillCommandRegistry {
	return &SkillCommandRegistry{
		commands: make(map[string]SkillCommandInfo),
	}
}

// BuildPlanPath returns the default workspace-relative markdown path for
// a /plan invocation. Uses relative paths so file tools resolve them
// against the active working directory.
func BuildPlanPath(userInstruction string, now time.Time) string {
	slugSource := ""
	instruction := strings.TrimSpace(userInstruction)
	if instruction != "" {
		lines := strings.SplitN(instruction, "\n", 2)
		slugSource = lines[0]
	}

	slug := planSlugRe.ReplaceAllString(strings.ToLower(slugSource), "-")
	slug = strings.Trim(slug, "-")

	if slug != "" {
		parts := strings.Split(slug, "-")
		if len(parts) > 8 {
			parts = parts[:8]
		}
		slug = strings.Join(parts, "-")
		if len(slug) > 48 {
			slug = slug[:48]
		}
		slug = strings.Trim(slug, "-")
	}
	if slug == "" {
		slug = "conversation-plan"
	}

	timestamp := now.Format("2006-01-02_150405")
	return filepath.Join(".hera", "plans", fmt.Sprintf("%s-%s.md", timestamp, slug))
}

// ScanSkillCommands walks the skills directories and returns a mapping
// of "/command" -> SkillCommandInfo.
func (r *SkillCommandRegistry) ScanSkillCommands(skillsDirs []string, disabledNames map[string]bool) map[string]SkillCommandInfo {
	r.commands = make(map[string]SkillCommandInfo)
	seenNames := make(map[string]bool)

	for _, dir := range skillsDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if base == ".git" || base == ".github" || base == ".hub" {
					return filepath.SkipDir
				}
				return nil
			}
			if info.Name() != "SKILL.md" {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				slog.Debug("read skill file", "path", path, "error", err)
				return nil
			}

			frontmatter, body := ParseFrontmatter(string(data))
			name, _ := frontmatter["name"].(string)
			if name == "" {
				name = filepath.Base(filepath.Dir(path))
			}
			if seenNames[name] {
				return nil
			}
			if disabledNames[name] {
				return nil
			}

			description, _ := frontmatter["description"].(string)
			if description == "" {
				// Use first non-heading non-empty line from body
				for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
					line = strings.TrimSpace(line)
					if line != "" && !strings.HasPrefix(line, "#") {
						if len(line) > 80 {
							line = line[:80]
						}
						description = line
						break
					}
				}
			}

			seenNames[name] = true

			cmdName := strings.ToLower(name)
			cmdName = strings.ReplaceAll(cmdName, " ", "-")
			cmdName = strings.ReplaceAll(cmdName, "_", "-")
			cmdName = skillInvalidRe.ReplaceAllString(cmdName, "")
			cmdName = skillMultiHypRe.ReplaceAllString(cmdName, "-")
			cmdName = strings.Trim(cmdName, "-")
			if cmdName == "" {
				return nil
			}

			if description == "" {
				description = fmt.Sprintf("Invoke the %s skill", name)
			}

			r.commands["/"+cmdName] = SkillCommandInfo{
				Name:        name,
				Description: description,
				SkillPath:   path,
				SkillDir:    filepath.Dir(path),
			}

			return nil
		})
		if err != nil {
			slog.Debug("scan skills dir", "dir", dir, "error", err)
		}
	}

	return r.commands
}

// GetCommands returns the current skill commands map. If empty, returns nil.
func (r *SkillCommandRegistry) GetCommands() map[string]SkillCommandInfo {
	return r.commands
}

// ResolveCommandKey resolves a user-typed /command to its canonical key.
// Hyphens and underscores are treated interchangeably.
func (r *SkillCommandRegistry) ResolveCommandKey(command string) string {
	if command == "" {
		return ""
	}
	cmdKey := "/" + strings.ReplaceAll(command, "_", "-")
	if _, ok := r.commands[cmdKey]; ok {
		return cmdKey
	}
	return ""
}

// BuildSkillMessage constructs the user message content for a skill
// slash command invocation.
func BuildSkillMessage(skillName, content, userInstruction string) string {
	var parts []string

	activationNote := fmt.Sprintf(
		`[SYSTEM: The user has invoked the "%s" skill, indicating they want you to follow its instructions. The full skill content is loaded below.]`,
		skillName,
	)
	parts = append(parts, activationNote, "", strings.TrimSpace(content))

	if userInstruction != "" {
		parts = append(parts, "",
			fmt.Sprintf("The user has provided the following instruction alongside the skill invocation: %s", userInstruction))
	}

	return strings.Join(parts, "\n")
}

// BuildPreloadedSkillsPrompt loads one or more skills for session-wide preloading.
// Returns the combined prompt text, loaded skill names, and missing identifiers.
func BuildPreloadedSkillsPrompt(identifiers []string, loader func(string) (string, string, error)) (string, []string, []string) {
	var promptParts []string
	var loadedNames []string
	var missing []string
	seen := make(map[string]bool)

	for _, id := range identifiers {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		content, name, err := loader(id)
		if err != nil {
			missing = append(missing, id)
			continue
		}

		activationNote := fmt.Sprintf(
			`[SYSTEM: The user launched this CLI session with the "%s" skill preloaded. Treat its instructions as active guidance for the duration of this session unless the user overrides them.]`,
			name,
		)
		msg := strings.Join([]string{activationNote, "", strings.TrimSpace(content)}, "\n")
		promptParts = append(promptParts, msg)
		loadedNames = append(loadedNames, name)
	}

	return strings.Join(promptParts, "\n\n"), loadedNames, missing
}
