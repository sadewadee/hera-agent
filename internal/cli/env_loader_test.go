package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHeraDotenv_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	loaded := LoadHeraDotenv(tmpDir, "")
	assert.Empty(t, loaded)
}

func TestLoadHeraDotenv_UserEnvOnly(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte("TEST_HERA_USER_ONLY=uservalue\n"), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_USER_ONLY", "")
	loaded := LoadHeraDotenv(tmpDir, "")
	assert.Len(t, loaded, 1)
	assert.Equal(t, envPath, loaded[0])
	assert.Equal(t, "uservalue", os.Getenv("TEST_HERA_USER_ONLY"))
}

func TestLoadHeraDotenv_ProjectEnvOnly(t *testing.T) {
	tmpDir := t.TempDir()
	projectEnv := filepath.Join(tmpDir, "project.env")
	err := os.WriteFile(projectEnv, []byte("TEST_HERA_PROJECT=projvalue\n"), 0644)
	require.NoError(t, err)

	heraDir := filepath.Join(tmpDir, "hera-home")
	require.NoError(t, os.MkdirAll(heraDir, 0755))
	// No .env in heraDir.

	t.Setenv("TEST_HERA_PROJECT", "")
	loaded := LoadHeraDotenv(heraDir, projectEnv)
	assert.Len(t, loaded, 1)
	assert.Equal(t, projectEnv, loaded[0])
	assert.Equal(t, "projvalue", os.Getenv("TEST_HERA_PROJECT"))
}

func TestLoadHeraDotenv_BothFiles_UserOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	heraDir := filepath.Join(tmpDir, "hera-home")
	require.NoError(t, os.MkdirAll(heraDir, 0755))
	userEnv := filepath.Join(heraDir, ".env")
	err := os.WriteFile(userEnv, []byte("TEST_HERA_BOTH=user\n"), 0644)
	require.NoError(t, err)

	projectEnv := filepath.Join(tmpDir, "project.env")
	err = os.WriteFile(projectEnv, []byte("TEST_HERA_BOTH=project\n"), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_BOTH", "")
	loaded := LoadHeraDotenv(heraDir, projectEnv)
	assert.Len(t, loaded, 2)
	// User env is loaded with override=true, so it wins.
	// Project env is loaded with override=false (user env exists), so it fills missing only.
	assert.Equal(t, "user", os.Getenv("TEST_HERA_BOTH"))
}

func TestLoadHeraDotenv_ProjectOnlyFillsMissing(t *testing.T) {
	tmpDir := t.TempDir()

	heraDir := filepath.Join(tmpDir, "hera-home")
	require.NoError(t, os.MkdirAll(heraDir, 0755))
	userEnv := filepath.Join(heraDir, ".env")
	err := os.WriteFile(userEnv, []byte("TEST_HERA_A=fromuser\n"), 0644)
	require.NoError(t, err)

	projectEnv := filepath.Join(tmpDir, "project.env")
	err = os.WriteFile(projectEnv, []byte("TEST_HERA_A=fromproject\nTEST_HERA_B=projectonly\n"), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_A", "")
	t.Setenv("TEST_HERA_B", "")
	loaded := LoadHeraDotenv(heraDir, projectEnv)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "fromuser", os.Getenv("TEST_HERA_A"))
	assert.Equal(t, "projectonly", os.Getenv("TEST_HERA_B"))
}

func TestLoadDotenvFile_Comments(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "# this is a comment\nTEST_HERA_COMMENT=value\n\n# another comment\n"
	err := os.WriteFile(envPath, []byte(content), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_COMMENT", "")
	loadDotenvFile(envPath, true)
	assert.Equal(t, "value", os.Getenv("TEST_HERA_COMMENT"))
}

func TestLoadDotenvFile_QuotedValues(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "TEST_HERA_DQ=\"double quoted\"\nTEST_HERA_SQ='single quoted'\n"
	err := os.WriteFile(envPath, []byte(content), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_DQ", "")
	t.Setenv("TEST_HERA_SQ", "")
	loadDotenvFile(envPath, true)
	assert.Equal(t, "double quoted", os.Getenv("TEST_HERA_DQ"))
	assert.Equal(t, "single quoted", os.Getenv("TEST_HERA_SQ"))
}

func TestLoadDotenvFile_NoOverride(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte("TEST_HERA_NOOV=newvalue\n"), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_NOOV", "existing")
	loadDotenvFile(envPath, false)
	assert.Equal(t, "existing", os.Getenv("TEST_HERA_NOOV"))
}

func TestLoadDotenvFile_Override(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte("TEST_HERA_OV=newvalue\n"), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HERA_OV", "existing")
	loadDotenvFile(envPath, true)
	assert.Equal(t, "newvalue", os.Getenv("TEST_HERA_OV"))
}

func TestLoadDotenvFile_SkipsInvalidLines(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "no-equals-sign\n=no-key\nVALID_KEY=valid\n"
	err := os.WriteFile(envPath, []byte(content), 0644)
	require.NoError(t, err)

	t.Setenv("VALID_KEY", "")
	loadDotenvFile(envPath, true)
	assert.Equal(t, "valid", os.Getenv("VALID_KEY"))
}

func TestLoadDotenvFile_NonExistentFile(t *testing.T) {
	// Should not panic, just log and return.
	loadDotenvFile("/nonexistent/path/.env", true)
}

func TestEnvFileExists_True(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("X=1\n"), 0644))
	assert.True(t, envFileExists(envPath))
}

func TestEnvFileExists_False(t *testing.T) {
	assert.False(t, envFileExists("/nonexistent/.env"))
}

func TestEnvFileExists_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	assert.False(t, envFileExists(tmpDir))
}
