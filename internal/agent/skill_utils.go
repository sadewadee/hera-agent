package agent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// PlatformMap maps user-facing platform names to Go's runtime.GOOS values.
var PlatformMap = map[string]string{
	"macos":   "darwin",
	"linux":   "linux",
	"windows": "windows",
}

// ExcludedSkillDirs contains directory names to skip during skill scanning.
var ExcludedSkillDirs = map[string]bool{
	".git":    true,
	".github": true,
	".hub":    true,
}

// SkillConfigPrefix is the storage prefix for skill config vars in config.yaml.
const SkillConfigPrefix = "skills.config"

// ParseFrontmatter parses YAML frontmatter from a markdown string.
// Returns the frontmatter as a map and the remaining body.
func ParseFrontmatter(content string) (map[string]interface{}, string) {
	frontmatter := make(map[string]interface{})
	body := content

	if !strings.HasPrefix(content, "---") {
		return frontmatter, body
	}

	// Find closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		// Try with \n--- at end
		if strings.HasSuffix(rest, "\n---") {
			idx = len(rest) - 4
		}
		if idx < 0 {
			return frontmatter, body
		}
	}

	yamlContent := rest[:idx]
	body = rest[idx+4:] // skip "\n---\n"
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &parsed); err != nil {
		// Fallback: simple key:value parsing
		for _, line := range strings.Split(strings.TrimSpace(yamlContent), "\n") {
			if !strings.Contains(line, ":") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			frontmatter[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		return frontmatter, body
	}

	if parsed != nil {
		frontmatter = parsed
	}
	return frontmatter, body
}

// SkillMatchesPlatform returns true when the skill is compatible with the
// current OS. Skills declare platform requirements via a "platforms" list
// in YAML frontmatter. If the field is absent or empty, the skill is
// compatible with all platforms.
func SkillMatchesPlatform(frontmatter map[string]interface{}) bool {
	platforms, ok := frontmatter["platforms"]
	if !ok || platforms == nil {
		return true
	}

	var platList []string
	switch v := platforms.(type) {
	case []interface{}:
		for _, p := range v {
			platList = append(platList, strings.ToLower(strings.TrimSpace(fmt.Sprint(p))))
		}
	case string:
		platList = []string{strings.ToLower(strings.TrimSpace(v))}
	default:
		return true
	}

	current := runtime.GOOS
	for _, p := range platList {
		mapped, ok := PlatformMap[p]
		if !ok {
			mapped = p
		}
		if strings.HasPrefix(current, mapped) {
			return true
		}
	}
	return false
}

// GetDisabledSkillNames reads disabled skill names from config.yaml.
func GetDisabledSkillNames(configPath, platform string) map[string]bool {
	disabled := make(map[string]bool)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return disabled
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return disabled
	}

	skillsCfg, _ := cfg["skills"].(map[string]interface{})
	if skillsCfg == nil {
		return disabled
	}

	if platform != "" {
		platDisabled, _ := skillsCfg["platform_disabled"].(map[string]interface{})
		if platDisabled != nil {
			if names, ok := platDisabled[platform]; ok {
				return normalizeStringSet(names)
			}
		}
	}

	return normalizeStringSet(skillsCfg["disabled"])
}

func normalizeStringSet(v interface{}) map[string]bool {
	result := make(map[string]bool)
	if v == nil {
		return result
	}
	switch val := v.(type) {
	case string:
		s := strings.TrimSpace(val)
		if s != "" {
			result[s] = true
		}
	case []interface{}:
		for _, item := range val {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				result[s] = true
			}
		}
	}
	return result
}

// GetExternalSkillsDirs reads skills.external_dirs from config.yaml
// and returns validated, existing directory paths.
func GetExternalSkillsDirs(configPath, localSkillsDir string) []string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	skillsCfg, _ := cfg["skills"].(map[string]interface{})
	if skillsCfg == nil {
		return nil
	}

	rawDirs := skillsCfg["external_dirs"]
	if rawDirs == nil {
		return nil
	}

	var dirList []string
	switch v := rawDirs.(type) {
	case string:
		dirList = []string{v}
	case []interface{}:
		for _, item := range v {
			dirList = append(dirList, fmt.Sprint(item))
		}
	default:
		return nil
	}

	localResolved, _ := filepath.Abs(localSkillsDir)
	seen := make(map[string]bool)
	var result []string

	for _, entry := range dirList {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// Expand ~ and environment variables
		if strings.HasPrefix(entry, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				entry = filepath.Join(home, entry[1:])
			}
		}
		entry = os.ExpandEnv(entry)
		abs, err := filepath.Abs(entry)
		if err != nil {
			continue
		}
		if abs == localResolved {
			continue
		}
		if seen[abs] {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			slog.Debug("external skills dir does not exist", "path", abs)
			continue
		}
		seen[abs] = true
		result = append(result, abs)
	}

	return result
}

// SkillConditions holds conditional activation fields from skill frontmatter.
type SkillConditions struct {
	FallbackForToolsets []string
	RequiresToolsets    []string
	FallbackForTools    []string
	RequiresTools       []string
}

// ExtractSkillConditions extracts conditional activation fields from
// parsed frontmatter.
func ExtractSkillConditions(frontmatter map[string]interface{}) SkillConditions {
	metadata, _ := frontmatter["metadata"].(map[string]interface{})
	if metadata == nil {
		return SkillConditions{}
	}
	hera, _ := metadata["hera"].(map[string]interface{})
	if hera == nil {
		return SkillConditions{}
	}
	return SkillConditions{
		FallbackForToolsets: extractStringList(hera["fallback_for_toolsets"]),
		RequiresToolsets:    extractStringList(hera["requires_toolsets"]),
		FallbackForTools:    extractStringList(hera["fallback_for_tools"]),
		RequiresTools:       extractStringList(hera["requires_tools"]),
	}
}

func extractStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		var result []string
		for _, item := range val {
			result = append(result, fmt.Sprint(item))
		}
		return result
	case string:
		return []string{val}
	}
	return nil
}

// SkillConfigVar describes a config variable declared by a skill.
type SkillConfigVar struct {
	Key         string      `json:"key"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Prompt      string      `json:"prompt"`
	Skill       string      `json:"skill,omitempty"`
}

// ExtractSkillConfigVars extracts config variable declarations from
// parsed frontmatter.
func ExtractSkillConfigVars(frontmatter map[string]interface{}) []SkillConfigVar {
	metadata, _ := frontmatter["metadata"].(map[string]interface{})
	if metadata == nil {
		return nil
	}
	hera, _ := metadata["hera"].(map[string]interface{})
	if hera == nil {
		return nil
	}

	raw := hera["config"]
	if raw == nil {
		return nil
	}

	var items []interface{}
	switch v := raw.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		items = []interface{}{v}
	default:
		return nil
	}

	seen := make(map[string]bool)
	var result []SkillConfigVar

	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		key := strings.TrimSpace(fmt.Sprint(m["key"]))
		if key == "" || seen[key] {
			continue
		}
		desc := strings.TrimSpace(fmt.Sprint(m["description"]))
		if desc == "" {
			continue
		}

		entry := SkillConfigVar{
			Key:         key,
			Description: desc,
		}
		if def, ok := m["default"]; ok {
			entry.Default = def
		}
		if prompt, ok := m["prompt"].(string); ok && strings.TrimSpace(prompt) != "" {
			entry.Prompt = strings.TrimSpace(prompt)
		} else {
			entry.Prompt = desc
		}
		seen[key] = true
		result = append(result, entry)
	}
	return result
}

// ExtractSkillDescription extracts a truncated description from frontmatter.
func ExtractSkillDescription(frontmatter map[string]interface{}) string {
	rawDesc, _ := frontmatter["description"].(string)
	if rawDesc == "" {
		return ""
	}
	desc := strings.Trim(strings.TrimSpace(rawDesc), "'\"")
	if len(desc) > 60 {
		return desc[:57] + "..."
	}
	return desc
}

// IterSkillIndexFiles walks skillsDir yielding sorted paths matching filename.
// Excludes .git, .github, .hub directories.
func IterSkillIndexFiles(skillsDir, filename string) ([]string, error) {
	var matches []string

	err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if ExcludedSkillDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == filename {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by relative path
	sort := func(paths []string) {
		// Simple lexicographic sort
		for i := 0; i < len(paths); i++ {
			for j := i + 1; j < len(paths); j++ {
				if paths[j] < paths[i] {
					paths[i], paths[j] = paths[j], paths[i]
				}
			}
		}
	}
	sort(matches)
	return matches, nil
}

// ResolveDotPath walks a nested map following a dotted key path.
// Returns nil if any part is missing.
func ResolveDotPath(config map[string]interface{}, dottedKey string) interface{} {
	parts := strings.Split(dottedKey, ".")
	var current interface{} = config

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}
