package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportConversationCmd(t *testing.T) {
	cmd := exportConversationCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "conversation", cmd.Use)
}

func TestExportAllCmd(t *testing.T) {
	cmd := exportAllCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "all", cmd.Use)
}

func TestImportConversationCmd(t *testing.T) {
	cmd := importConversationCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "conversation [file]", cmd.Use)
}

func TestRegisterExportCommands(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registerExportCommands(rootCmd)

	names := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		names[c.Use] = true
	}
	assert.True(t, names["export"])
	assert.True(t, names["import"])
}

func TestConversationExport_JSON(t *testing.T) {
	export := conversationExport{
		Version:    "1.0",
		ExportedAt: "2024-01-01T00:00:00Z",
		Sessions: []sessionExport{
			{
				ID:       "s1",
				Platform: "telegram",
				UserID:   "u1",
				Messages: []messageExport{
					{Role: "user", Content: "hello"},
					{Role: "assistant", Content: "hi there"},
				},
			},
		},
	}

	data, err := json.Marshal(export)
	require.NoError(t, err)

	var decoded conversationExport
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "1.0", decoded.Version)
	assert.Len(t, decoded.Sessions, 1)
	assert.Len(t, decoded.Sessions[0].Messages, 2)
}

func TestImportConversation_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "export.json")

	export := conversationExport{
		Version:    "1.0",
		ExportedAt: "2024-01-01T00:00:00Z",
		Sessions:   []sessionExport{{ID: "test-session"}},
	}

	data, _ := json.Marshal(export)
	require.NoError(t, os.WriteFile(exportPath, data, 0644))

	// Verify it can be parsed
	readData, err := os.ReadFile(exportPath)
	require.NoError(t, err)

	var imported conversationExport
	require.NoError(t, json.Unmarshal(readData, &imported))
	assert.Equal(t, "1.0", imported.Version)
}
