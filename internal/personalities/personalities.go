// Package personalities bundles the built-in personality profiles and
// resolves a personality name (e.g. "kawaii") to the guidelines text that
// should be injected into the agent's system prompt.
//
// Resolution order:
//  1. User override at ~/.hera/personalities/<name>.yaml (if HOME is set).
//  2. Bundled profile embedded in the binary.
//  3. Caller's original string (returned unchanged).
//
// Each YAML file has the shape:
//
//	name: <name>
//	description: <one-liner>
//	traits: [list of short traits]
//	guidelines: |
//	  Multi-line system-prompt body injected verbatim.
//
// Only the guidelines field is passed to the LLM.
package personalities

import (
	"embed"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var bundled embed.FS

// nameRE validates that a string looks like a personality file name rather
// than inline personality content: short, lowercase, alphanumeric with
// dashes/underscores. Anything with whitespace or longer than 64 chars is
// treated as literal prompt content and left alone.
var nameRE = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

type profile struct {
	Name       string `yaml:"name"`
	Guidelines string `yaml:"guidelines"`
}

// Resolve returns the guidelines text for the given value. If the value is
// not a personality name (contains newlines, is too long, has whitespace, etc.)
// it is returned as-is so existing configs that embed literal personality
// prompts keep working.
//
// When the value is a name, user override takes precedence over the bundled
// profile. If neither is found, the original name is returned unchanged.
func Resolve(value string) string {
	trimmed := strings.TrimSpace(value)
	if !nameRE.MatchString(trimmed) {
		return value
	}

	if guidelines, ok := loadUserOverride(trimmed); ok {
		return guidelines
	}
	if guidelines, ok := loadBundled(trimmed); ok {
		return guidelines
	}
	return value
}

// List returns the names of all bundled personalities, sorted.
func List() []string {
	entries, err := bundled.ReadDir(".")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := strings.TrimSuffix(e.Name(), ".yaml")
		if n != "" {
			names = append(names, n)
		}
	}
	return names
}

func loadUserOverride(name string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	path := filepath.Join(home, ".hera", "personalities", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return parseGuidelines(data)
}

func loadBundled(name string) (string, bool) {
	data, err := bundled.ReadFile(name + ".yaml")
	if err != nil {
		return "", false
	}
	return parseGuidelines(data)
}

func parseGuidelines(data []byte) (string, bool) {
	var p profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return "", false
	}
	g := strings.TrimSpace(p.Guidelines)
	if g == "" {
		return "", false
	}
	return g, true
}
