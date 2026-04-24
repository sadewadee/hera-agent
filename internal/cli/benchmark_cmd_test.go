package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchTokenizeCmd(t *testing.T) {
	cmd := benchTokenizeCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "tokenize", cmd.Use)
}

func TestBenchLatencyCmd(t *testing.T) {
	cmd := benchLatencyCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "latency", cmd.Use)
}

func TestBenchThroughputCmd(t *testing.T) {
	cmd := benchThroughputCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "throughput", cmd.Use)
}

func TestRegisterBenchmarkCommands(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registerBenchmarkCommands(rootCmd)

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "benchmark" {
			found = true
			subCmds := make(map[string]bool)
			for _, sc := range c.Commands() {
				subCmds[sc.Use] = true
			}
			assert.True(t, subCmds["tokenize"])
			assert.True(t, subCmds["latency"])
			assert.True(t, subCmds["throughput"])
		}
	}
	assert.True(t, found)
}

func TestBenchTokenizeCmd_Run(t *testing.T) {
	cmd := benchTokenizeCmd()
	cmd.Flags().Set("iterations", "10")
	err := cmd.RunE(cmd, nil)
	assert.NoError(t, err)
}
