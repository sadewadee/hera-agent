package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRegisterCompletionsCommand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registerCompletionsCommand(rootCmd)

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "completion [bash|zsh|fish|powershell]" {
			found = true
			assert.Equal(t, []string{"bash", "zsh", "fish", "powershell"}, c.ValidArgs)
		}
	}
	assert.True(t, found, "completion command should be registered")
}
