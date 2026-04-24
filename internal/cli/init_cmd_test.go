package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunInit_CreatesDirectoryStructure verifies that runInit creates all
// required subdirectories under HERA_HOME.
func TestRunInit_CreatesDirectoryStructure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERA_HOME", home)
	t.Setenv("HERA_BUNDLED", "") // no bundled dir

	err := runInit(false)
	require.NoError(t, err)

	expectedDirs := []string{
		home,
		filepath.Join(home, "skills"),
		filepath.Join(home, "hooks.d"),
		filepath.Join(home, "tools.d"),
		filepath.Join(home, "plugins"),
		filepath.Join(home, "logs"),
	}
	for _, d := range expectedDirs {
		info, err := os.Stat(d)
		require.NoError(t, err, "expected dir %s to exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}
}

// TestRunInit_Idempotent verifies that calling runInit twice does not fail
// and does not overwrite any user files.
func TestRunInit_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERA_HOME", home)
	t.Setenv("HERA_BUNDLED", "") // no bundled dir

	require.NoError(t, runInit(false), "first call")
	require.NoError(t, runInit(false), "second call (idempotent)")
}

// TestRunInit_SeedsSkillsFromBundled verifies that when HERA_BUNDLED is set
// to a valid directory containing skills, those skills are copied to HERA_HOME.
func TestRunInit_SeedsSkillsFromBundled(t *testing.T) {
	home := t.TempDir()
	bundled := t.TempDir()
	t.Setenv("HERA_HOME", home)
	t.Setenv("HERA_BUNDLED", bundled)

	// Create bundled skills.
	skillContent := "# Hello\nA bundled skill."
	skillPath := filepath.Join(bundled, "skills", "general", "hello.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0o755))
	require.NoError(t, os.WriteFile(skillPath, []byte(skillContent), 0o644))

	err := runInit(false)
	require.NoError(t, err)

	// Skill must be in user skills dir.
	userSkill := filepath.Join(home, "skills", "general", "hello.md")
	data, err := os.ReadFile(userSkill)
	require.NoError(t, err)
	assert.Equal(t, skillContent, string(data))
}

// TestRunInit_SeedsConfigIfAbsent verifies that the example config is copied
// from bundled configs when config.yaml does not exist.
func TestRunInit_SeedsConfigIfAbsent(t *testing.T) {
	home := t.TempDir()
	bundled := t.TempDir()
	t.Setenv("HERA_HOME", home)
	t.Setenv("HERA_BUNDLED", bundled)

	// Create bundled example config.
	exampleContent := "# Hera example config\nagent:\n  default_provider: openai\n"
	examplePath := filepath.Join(bundled, "configs", "hera.example.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(examplePath), 0o755))
	require.NoError(t, os.WriteFile(examplePath, []byte(exampleContent), 0o644))

	err := runInit(false)
	require.NoError(t, err)

	configPath := filepath.Join(home, "config.yaml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, exampleContent, string(data))
}

// TestRunInit_DoesNotOverwriteExistingConfig verifies that an existing
// config.yaml is never replaced by hera init.
func TestRunInit_DoesNotOverwriteExistingConfig(t *testing.T) {
	home := t.TempDir()
	bundled := t.TempDir()
	t.Setenv("HERA_HOME", home)
	t.Setenv("HERA_BUNDLED", bundled)

	// Create bundled example config.
	exampleContent := "# example"
	examplePath := filepath.Join(bundled, "configs", "hera.example.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(examplePath), 0o755))
	require.NoError(t, os.WriteFile(examplePath, []byte(exampleContent), 0o644))

	// User already has config.
	userConfigContent := "# user config — do not overwrite"
	require.NoError(t, os.MkdirAll(home, 0o755))
	configPath := filepath.Join(home, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(userConfigContent), 0o644))

	err := runInit(false)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, userConfigContent, string(data), "user config must not be overwritten")
}
