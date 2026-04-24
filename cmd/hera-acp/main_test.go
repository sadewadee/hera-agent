package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRunReturnsError verifies run() returns an error when no config or API key is available.
func TestRunReturnsError(t *testing.T) {
	// Clear env to ensure no API key is found.
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")

	err := run()
	// run() should return an error because no API key is configured.
	// It might fail at config load, API key check, or memory init, depending
	// on the environment. We just verify it does not succeed silently.
	assert.Error(t, err)
}
