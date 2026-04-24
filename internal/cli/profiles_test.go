package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ValidateProfileID ---

func TestValidateProfileID_Valid(t *testing.T) {
	assert.NoError(t, ValidateProfileID("myprofile"))
	assert.NoError(t, ValidateProfileID("profile-1"))
	assert.NoError(t, ValidateProfileID("profile_2"))
	assert.NoError(t, ValidateProfileID("a"))
}

func TestValidateProfileID_Invalid(t *testing.T) {
	assert.Error(t, ValidateProfileID(""))
	assert.Error(t, ValidateProfileID("-invalid"))
	assert.Error(t, ValidateProfileID("_invalid"))
	assert.Error(t, ValidateProfileID("UPPER"))
	assert.Error(t, ValidateProfileID("has space"))
	assert.Error(t, ValidateProfileID("has.dot"))
}

func TestValidateProfileID_ReservedDefault(t *testing.T) {
	err := ValidateProfileID("default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}

// --- ProfilePath ---

func TestProfilePath_Default(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)
	path := ProfilePath("default")
	assert.Equal(t, tmpDir, path)
}

func TestProfilePath_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)
	path := ProfilePath("")
	assert.Equal(t, tmpDir, path)
}

func TestProfilePath_Named(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)
	path := ProfilePath("test")
	expected := filepath.Join(tmpDir, "profiles", "test")
	assert.Equal(t, expected, path)
}

// --- ProfilesRoot ---

func TestProfilesRoot(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)
	root := ProfilesRoot()
	assert.Equal(t, filepath.Join(tmpDir, "profiles"), root)
}

// --- CreateProfile ---

func TestCreateProfile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	err := CreateProfile("testprofile", false, "")
	require.NoError(t, err)

	profileDir := ProfilePath("testprofile")
	info, err := os.Stat(profileDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check bootstrapped subdirs.
	for _, dir := range profileDirs {
		dirPath := filepath.Join(profileDir, dir)
		_, err := os.Stat(dirPath)
		assert.NoError(t, err, "missing subdir: %s", dir)
	}

	// Check SOUL.md was created.
	_, err = os.Stat(filepath.Join(profileDir, "SOUL.md"))
	assert.NoError(t, err)
}

func TestCreateProfile_InvalidName(t *testing.T) {
	err := CreateProfile("INVALID", false, "")
	assert.Error(t, err)
}

func TestCreateProfile_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	require.NoError(t, CreateProfile("existing", false, ""))
	err := CreateProfile("existing", false, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateProfile_Clone(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	// Create a "source" profile with a config.yaml.
	require.NoError(t, CreateProfile("source", false, ""))
	sourcePath := ProfilePath("source")
	configContent := "model: gpt-4\n"
	require.NoError(t, os.WriteFile(filepath.Join(sourcePath, "config.yaml"), []byte(configContent), 0644))

	// Clone from source.
	err := CreateProfile("cloned", true, "source")
	require.NoError(t, err)

	clonedConfig := filepath.Join(ProfilePath("cloned"), "config.yaml")
	data, err := os.ReadFile(clonedConfig)
	require.NoError(t, err)
	assert.Equal(t, configContent, string(data))
}

// --- DeleteProfile ---

func TestDeleteProfile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	require.NoError(t, CreateProfile("todelete", false, ""))
	err := DeleteProfile("todelete")
	require.NoError(t, err)

	_, err = os.Stat(ProfilePath("todelete"))
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteProfile_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	err := DeleteProfile("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDeleteProfile_InvalidName(t *testing.T) {
	err := DeleteProfile("default")
	assert.Error(t, err)
}

// --- ListProfiles ---

func TestListProfiles_DefaultOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	profiles := ListProfiles()
	assert.GreaterOrEqual(t, len(profiles), 1)
	assert.Equal(t, "default", profiles[0].Name)
	assert.True(t, profiles[0].Default)
}

func TestListProfiles_WithNamed(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HERA_HOME", tmpDir)

	require.NoError(t, CreateProfile("alpha", false, ""))
	require.NoError(t, CreateProfile("beta", false, ""))

	profiles := ListProfiles()
	assert.GreaterOrEqual(t, len(profiles), 3) // default + alpha + beta

	names := make(map[string]bool)
	for _, p := range profiles {
		names[p.Name] = true
	}
	assert.True(t, names["default"])
	assert.True(t, names["alpha"])
	assert.True(t, names["beta"])
}
