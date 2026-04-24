// Package plugins implements the third-party plugin system for Hera.
//
// Plugins are git repositories cloned into $HERA_HOME/plugins/<owner-repo>/.
// Each plugin has a plugin.yaml manifest declaring what it provides:
// skills, hooks, tools, and/or an MCP server command.
//
// Plugin lifecycle:
//
//	hera plugins install owner/repo  → git clone into $HERA_HOME/plugins/
//	hera plugins update  owner/repo  → git pull
//	hera plugins remove  owner/repo  → rm -rf
//	hera plugins enable  owner/repo  → remove .disabled marker
//	hera plugins disable owner/repo  → touch .disabled marker
//	hera plugins list                → print table
//
// The Manager never auto-loads installed plugins — call the Loader
// at startup to merge plugin assets into the running agent.
package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// disabledMarker is a sentinel file placed in a plugin directory to
	// disable it without removing the clone.
	disabledMarker = ".disabled"
	// manifestFilename is the plugin manifest inside each plugin directory.
	manifestFilename = "plugin.yaml"
)

// Manifest describes the contents of a plugin.yaml file.
type Manifest struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Author      string    `yaml:"author"`
	Description string    `yaml:"description"`
	Provides    *Provides `yaml:"provides,omitempty"`
}

// Provides declares which subdirectories and capabilities the plugin offers.
type Provides struct {
	Skills string     `yaml:"skills,omitempty"` // relative path to skills dir
	Hooks  string     `yaml:"hooks,omitempty"`  // relative path to hooks.d dir
	Tools  string     `yaml:"tools,omitempty"`  // relative path to tools.d dir
	MCP    *MCPServer `yaml:"mcp_server,omitempty"`
}

// MCPServer describes an MCP server bundled with the plugin.
type MCPServer struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
	Mode    string   `yaml:"mode,omitempty"` // on_demand or persistent
}

// PluginInfo describes an installed plugin.
type PluginInfo struct {
	// DirName is the directory name under $HERA_HOME/plugins/ (e.g. "owner-repo").
	DirName string
	// Path is the absolute path to the plugin directory.
	Path string
	// Enabled reports whether the plugin is active (no .disabled marker).
	Enabled bool
	// Manifest is the parsed plugin.yaml, or nil if absent/unreadable.
	Manifest *Manifest
}

// Manager manages the plugin lifecycle for Hera.
// It operates on $HERA_HOME/plugins/ and uses the system git binary.
type Manager struct {
	// PluginsDir is the absolute path to the plugins directory.
	PluginsDir string
}

// NewManager creates a Manager rooted at pluginsDir.
// The directory need not exist; Install creates it on first use.
func NewManager(pluginsDir string) *Manager {
	return &Manager{PluginsDir: pluginsDir}
}

// Install clones a plugin from spec into $HERA_HOME/plugins/.
//
// spec may be:
//   - A full git URL: "https://github.com/owner/repo.git"
//   - A GitHub shorthand: "owner/repo" → expanded to https://github.com/owner/repo.git
//
// Returns an error when the directory already exists (use Update instead).
func (m *Manager) Install(spec string) (*PluginInfo, error) {
	repoURL, dirName, err := parseSpec(spec)
	if err != nil {
		return nil, err
	}

	dest := filepath.Join(m.PluginsDir, dirName)
	if _, statErr := os.Stat(dest); statErr == nil {
		return nil, fmt.Errorf("plugin %q already installed at %s; use 'hera plugins update' to upgrade", dirName, dest)
	}

	if err := os.MkdirAll(m.PluginsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create plugins dir: %w", err)
	}

	if err := runGit(m.PluginsDir, "clone", "--depth=1", repoURL, dest); err != nil {
		return nil, fmt.Errorf("git clone %s: %w", repoURL, err)
	}

	return m.info(dirName)
}

// Update runs git pull in an already-installed plugin directory.
func (m *Manager) Update(name string) (*PluginInfo, error) {
	dest := filepath.Join(m.PluginsDir, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin %q not installed", name)
	}
	if err := runGit(dest, "pull", "--ff-only"); err != nil {
		return nil, fmt.Errorf("git pull in %s: %w", dest, err)
	}
	return m.info(name)
}

// Remove deletes a plugin directory entirely.
func (m *Manager) Remove(name string) error {
	dest := filepath.Join(m.PluginsDir, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q not installed", name)
	}
	return os.RemoveAll(dest)
}

// Enable removes the .disabled marker so the plugin is loaded on next startup.
func (m *Manager) Enable(name string) error {
	dest := filepath.Join(m.PluginsDir, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q not installed", name)
	}
	marker := filepath.Join(dest, disabledMarker)
	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove .disabled marker: %w", err)
	}
	return nil
}

// Disable creates a .disabled marker so the plugin is skipped by the Loader.
func (m *Manager) Disable(name string) error {
	dest := filepath.Join(m.PluginsDir, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q not installed", name)
	}
	marker := filepath.Join(dest, disabledMarker)
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create .disabled marker: %w", err)
	}
	f.Close()
	return nil
}

// List returns info for every directory under PluginsDir.
// Non-directory entries are ignored. Missing plugin.yaml is not an error.
func (m *Manager) List() ([]*PluginInfo, error) {
	entries, err := os.ReadDir(m.PluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}
	var result []*PluginInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := m.info(e.Name())
		if err != nil {
			continue
		}
		result = append(result, info)
	}
	return result, nil
}

// info builds a PluginInfo for a named subdirectory.
func (m *Manager) info(name string) (*PluginInfo, error) {
	path := filepath.Join(m.PluginsDir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin dir %q not found", path)
	}

	_, err := os.Stat(filepath.Join(path, disabledMarker))
	enabled := os.IsNotExist(err)

	manifest, _ := LoadManifest(filepath.Join(path, manifestFilename))

	return &PluginInfo{
		DirName:  name,
		Path:     path,
		Enabled:  enabled,
		Manifest: manifest,
	}, nil
}

// LoadManifest parses a plugin.yaml file. Returns nil manifest without error
// when the file does not exist (allows plugins that omit plugin.yaml).
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugin manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse plugin manifest: %w", err)
	}
	return &m, nil
}

// parseSpec converts an install spec to a git URL and directory name.
//
//   - "owner/repo"                    → "https://github.com/owner/repo.git", "owner-repo"
//   - "https://example.com/org/r.git" → same URL, "org-r" (last two path segments)
func parseSpec(spec string) (repoURL, dirName string, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", fmt.Errorf("empty plugin spec")
	}

	if strings.HasPrefix(spec, "https://") || strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "git@") {
		repoURL = spec
		// Derive dirName from the last path segments.
		clean := strings.TrimSuffix(spec, ".git")
		parts := strings.Split(strings.TrimSuffix(clean, "/"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("cannot derive plugin dir name from URL %q", spec)
		}
		dirName = parts[len(parts)-2] + "-" + parts[len(parts)-1]
	} else {
		// Shorthand: "owner/repo"
		parts := strings.SplitN(spec, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid plugin spec %q: expected owner/repo or full git URL", spec)
		}
		repoURL = "https://github.com/" + parts[0] + "/" + parts[1] + ".git"
		dirName = parts[0] + "-" + parts[1]
	}

	// Sanitise dirName: allow only alphanumeric, hyphen, underscore, dot.
	dirName = sanitiseDirName(dirName)
	return repoURL, dirName, nil
}

// sanitiseDirName replaces characters not in [a-zA-Z0-9._-] with "-".
func sanitiseDirName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// runGit executes git with the given arguments in dir.
// stdout and stderr from git are captured and included in the error on failure.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
