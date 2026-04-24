package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- parseSpec ---

func TestParseSpec_Shorthand(t *testing.T) {
	url, dir, err := parseSpec("alice/my-plugin")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/alice/my-plugin.git", url)
	assert.Equal(t, "alice-my-plugin", dir)
}

func TestParseSpec_FullHTTPS(t *testing.T) {
	url, dir, err := parseSpec("https://github.com/bob/cool-plugin.git")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/bob/cool-plugin.git", url)
	assert.Equal(t, "bob-cool-plugin", dir)
}

func TestParseSpec_EmptyReturnsError(t *testing.T) {
	_, _, err := parseSpec("")
	require.Error(t, err)
}

func TestParseSpec_InvalidShorthandReturnsError(t *testing.T) {
	_, _, err := parseSpec("no-slash-here")
	require.Error(t, err)
}

func TestParseSpec_SanitisesDirName(t *testing.T) {
	_, dir, err := parseSpec("alice/my plugin") // space in repo name (unusual but possible in URL)
	require.NoError(t, err)
	// The space after slash becomes part of repo name → sanitised
	assert.NotContains(t, dir, " ")
}

// --- Manager (no git, filesystem-only operations) ---

func TestManager_List_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	infos, err := m.List()
	require.NoError(t, err)
	assert.Empty(t, infos)
}

func TestManager_List_NonexistentDir(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nonexistent"))
	infos, err := m.List()
	require.NoError(t, err)
	assert.Nil(t, infos)
}

func TestManager_List_WithPluginDirs(t *testing.T) {
	dir := t.TempDir()
	// Create two plugin directories, one with plugin.yaml.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "alice-plugin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "bob-tool"), 0o755))
	writeManifest(t, filepath.Join(dir, "alice-plugin"), "alice-plugin", "1.0.0")

	m := NewManager(dir)
	infos, err := m.List()
	require.NoError(t, err)
	assert.Len(t, infos, 2)
}

func TestManager_List_IgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "real-plugin"), 0o755))

	m := NewManager(dir)
	infos, err := m.List()
	require.NoError(t, err)
	assert.Len(t, infos, 1)
}

func TestManager_EnableDisable(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	m := NewManager(dir)

	// Should start enabled.
	infos, _ := m.List()
	require.Len(t, infos, 1)
	assert.True(t, infos[0].Enabled)

	// Disable.
	require.NoError(t, m.Disable("my-plugin"))
	infos, _ = m.List()
	assert.False(t, infos[0].Enabled)

	// Enable again.
	require.NoError(t, m.Enable("my-plugin"))
	infos, _ = m.List()
	assert.True(t, infos[0].Enabled)
}

func TestManager_DisableNonexistent(t *testing.T) {
	m := NewManager(t.TempDir())
	err := m.Disable("ghost")
	require.Error(t, err)
}

func TestManager_EnableNonexistent(t *testing.T) {
	m := NewManager(t.TempDir())
	err := m.Enable("ghost")
	require.Error(t, err)
}

func TestManager_Remove(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	m := NewManager(dir)
	require.NoError(t, m.Remove("my-plugin"))

	infos, _ := m.List()
	assert.Empty(t, infos)
}

func TestManager_RemoveNonexistent(t *testing.T) {
	m := NewManager(t.TempDir())
	err := m.Remove("ghost")
	require.Error(t, err)
}

// --- LoadManifest ---

func TestLoadManifest_Valid(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "test-plugin", "2.0.0")

	m, err := LoadManifest(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "test-plugin", m.Name)
	assert.Equal(t, "2.0.0", m.Version)
}

func TestLoadManifest_Missing(t *testing.T) {
	m, err := LoadManifest(filepath.Join(t.TempDir(), "plugin.yaml"))
	require.NoError(t, err) // absent = nil, no error
	assert.Nil(t, m)
}

func TestLoadManifest_WithProvides(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: full-plugin
version: 0.1.0
provides:
  skills: skills/
  hooks: hooks.d/
  tools: tools.d/
  mcp_server:
    command: python3
    args: ["-m", "myplugin"]
    mode: on_demand
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(yaml), 0o644))

	m, err := LoadManifest(filepath.Join(dir, "plugin.yaml"))
	require.NoError(t, err)
	require.NotNil(t, m.Provides)
	assert.Equal(t, "skills/", m.Provides.Skills)
	assert.Equal(t, "hooks.d/", m.Provides.Hooks)
	assert.Equal(t, "tools.d/", m.Provides.Tools)
	require.NotNil(t, m.Provides.MCP)
	assert.Equal(t, "python3", m.Provides.MCP.Command)
}

// --- LoadEnabled ---

func TestLoadEnabled_NoPlugins(t *testing.T) {
	result := LoadEnabled(filepath.Join(t.TempDir(), "plugins"))
	assert.Empty(t, result.SkillDirs)
	assert.Empty(t, result.MCPEntries)
}

func TestLoadEnabled_MergesSkillsAndMCP(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	yaml := `
name: my-plugin
version: 0.1.0
provides:
  skills: skills/
  mcp_server:
    command: python3
    args: ["-m", "myplugin"]
`
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(yaml), 0o644))

	result := LoadEnabled(dir)
	require.Len(t, result.SkillDirs, 1)
	assert.Equal(t, filepath.Join(pluginDir, "skills"), result.SkillDirs[0])
	require.Len(t, result.MCPEntries, 1)
	assert.Equal(t, "python3", result.MCPEntries[0].Command)
}

func TestLoadEnabled_SkipsDisabledPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "disabled-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	writeManifest(t, pluginDir, "disabled-plugin", "1.0.0")
	// Disable it.
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, ".disabled"), nil, 0o644))

	result := LoadEnabled(dir)
	assert.Empty(t, result.SkillDirs)
}

// --- helpers ---

func writeManifest(t *testing.T, dir, name, version string) {
	t.Helper()
	yaml := "name: " + name + "\nversion: " + version + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(yaml), 0o644))
}
