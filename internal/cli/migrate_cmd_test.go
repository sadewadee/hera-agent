package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateUpCmd(t *testing.T) {
	cmd := migrateUpCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "up", cmd.Use)
}

func TestMigrateStatusCmd(t *testing.T) {
	cmd := migrateStatusCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "status", cmd.Use)
}

func TestMigrateCreateCmd(t *testing.T) {
	cmd := migrateCreateCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "create [name]", cmd.Use)
}

func TestRegisterMigrateCommands(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registerMigrateCommands(rootCmd)

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "migrate" {
			found = true
			subCmds := make(map[string]bool)
			for _, sc := range c.Commands() {
				subCmds[sc.Use] = true
			}
			assert.True(t, subCmds["up"])
			assert.True(t, subCmds["status"])
			assert.True(t, subCmds["create [name]"])
		}
	}
	assert.True(t, found)
}

func TestGetDBPath(t *testing.T) {
	path, err := getDBPath()
	require.NoError(t, err)
	assert.Contains(t, path, ".hera")
	assert.Contains(t, path, "hera.db")
}
