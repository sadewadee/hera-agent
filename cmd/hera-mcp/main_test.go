package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRunReturnsError verifies run() returns an error when no config or API key
// is available.
func TestRunReturnsError(t *testing.T) {
	// Clear env vars to ensure no API key is found.
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	err := run()
	// run() should return an error because no API key is configured and
	// the default provider requires one.
	assert.Error(t, err)
}
