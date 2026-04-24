package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/sadewadee/hera/internal/paths"
)

// Loader discovers and loads skills from directories.
type Loader struct {
	mu       sync.RWMutex
	skills   map[string]*Skill
	dirs     []string
	platform string // current platform for filtering (e.g., "cli", "telegram")
}

// NewLoader creates a skill loader with the given search directories.
func NewLoader(dirs ...string) *Loader {
	return &Loader{
		skills: make(map[string]*Skill),
		dirs:   dirs,
	}
}

// LoadAll scans all directories and loads skills.
func (l *Loader) LoadAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, dir := range l.dirs {
		if err := l.loadDir(dir); err != nil {
			return fmt.Errorf("load skills from %s: %w", dir, err)
		}
	}
	return nil
}

func (l *Loader) loadDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		skill, err := ParseSkillFile(path)
		if err != nil {
			return nil // skip invalid skills
		}

		// Determine tier based on directory
		skill.Tier = l.tierForPath(path)
		l.skills[skill.Name] = skill
		return nil
	})
}

func (l *Loader) tierForPath(path string) string {
	for _, dir := range l.dirs {
		if strings.Contains(dir, "optional") {
			if strings.HasPrefix(path, dir) {
				return "optional"
			}
		}
	}
	if strings.HasPrefix(path, paths.UserSkills()) {
		return "user"
	}
	return "bundled"
}

// SetPlatform sets the current platform for skill filtering.
// Skills with a non-empty Platforms field will only match if the current platform is listed.
func (l *Loader) SetPlatform(platform string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.platform = platform
}

// matchesPlatform returns true if the skill is available on the current platform.
// Skills with an empty Platforms field match all platforms.
func (l *Loader) matchesPlatform(s *Skill) bool {
	if len(s.Platforms) == 0 {
		return true
	}
	if l.platform == "" {
		return true
	}
	for _, p := range s.Platforms {
		if strings.EqualFold(p, l.platform) {
			return true
		}
	}
	return false
}

// Get returns a skill by name, filtered by current platform.
func (l *Loader) Get(name string) (*Skill, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	s, ok := l.skills[name]
	if !ok {
		return nil, false
	}
	if !l.matchesPlatform(s) {
		return nil, false
	}
	return s, ok
}

// FindByTrigger returns skills matching a trigger word, filtered by current platform.
func (l *Loader) FindByTrigger(trigger string) []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	trigger = strings.ToLower(trigger)
	var matched []*Skill
	for _, s := range l.skills {
		if !l.matchesPlatform(s) {
			continue
		}
		for _, t := range s.Triggers {
			if strings.ToLower(t) == trigger {
				matched = append(matched, s)
				break
			}
		}
	}
	return matched
}

// All returns all loaded skills.
func (l *Loader) All() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]*Skill, 0, len(l.skills))
	for _, s := range l.skills {
		result = append(result, s)
	}
	return result
}

// ParseSkillFile reads and parses a skill markdown file with YAML frontmatter.
func ParseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	skill, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter in %s: %w", path, err)
	}

	skill.Content = strings.TrimSpace(body)
	skill.FilePath = path
	return skill, nil
}

func parseFrontmatter(content string) (*Skill, string, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	end := strings.Index(content[3:], "---")
	if end == -1 {
		return nil, "", fmt.Errorf("unclosed frontmatter")
	}
	end += 3

	frontmatter := content[3:end]
	body := content[end+3:]

	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, "", err
	}

	return &skill, body, nil
}
