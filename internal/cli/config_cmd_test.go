package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigShowCmd(t *testing.T) {
	cmd := configShowCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "show", cmd.Use)
}

func TestConfigSetCmd(t *testing.T) {
	cmd := configSetCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "set [key] [value]", cmd.Use)
}

func TestConfigGetCmd(t *testing.T) {
	cmd := configGetCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "get [key]", cmd.Use)
}

func TestConfigPathCmd(t *testing.T) {
	cmd := configPathCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "path", cmd.Use)
}

func TestConfigInitCmd(t *testing.T) {
	cmd := configInitCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "init", cmd.Use)
}

func TestRegisterConfigCommands(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registerConfigCommands(rootCmd)

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "config" {
			found = true
			// Verify subcommands exist
			subCmds := make(map[string]bool)
			for _, sc := range c.Commands() {
				subCmds[sc.Use] = true
			}
			assert.True(t, subCmds["show"])
			assert.True(t, subCmds["set [key] [value]"])
			assert.True(t, subCmds["get [key]"])
			assert.True(t, subCmds["path"])
			assert.True(t, subCmds["init"])
		}
	}
	assert.True(t, found, "config command should be registered")
}
