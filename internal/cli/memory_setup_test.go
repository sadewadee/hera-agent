package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- WriteEnvVars ---

func TestWriteEnvVars_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	err := WriteEnvVars(envPath, map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "KEY1=value1")
	assert.Contains(t, content, "KEY2=value2")
}

func TestWriteEnvVars_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Write initial content.
	require.NoError(t, os.WriteFile(envPath, []byte("KEY1=old\nKEY3=keep\n"), 0644))

	err := WriteEnvVars(envPath, map[string]string{
		"KEY1": "new",
		"KEY2": "added",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "KEY1=new")
	assert.Contains(t, content, "KEY2=added")
	assert.Contains(t, content, "KEY3=keep")
}

func TestWriteEnvVars_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "sub", "dir", ".env")

	err := WriteEnvVars(envPath, map[string]string{"KEY": "val"})
	require.NoError(t, err)

	_, err = os.Stat(envPath)
	assert.NoError(t, err)
}

// --- MemoryProvider struct ---

func TestMemoryProvider_Fields(t *testing.T) {
	p := MemoryProvider{
		Name:        "test-provider",
		Description: "A test",
		SetupHint:   "just testing",
		Available:   true,
	}
	assert.Equal(t, "test-provider", p.Name)
	assert.True(t, p.Available)
}

// --- MemorySetupField struct ---

func TestMemorySetupField_Fields(t *testing.T) {
	f := MemorySetupField{
		Key:         "api_key",
		Description: "API Key",
		Default:     "",
		Secret:      true,
		EnvVar:      "MY_API_KEY",
		Choices:     []string{"a", "b"},
	}
	assert.Equal(t, "api_key", f.Key)
	assert.True(t, f.Secret)
	assert.Len(t, f.Choices, 2)
}
