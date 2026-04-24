package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCopilotClient(t *testing.T) {
	c := NewCopilotClient("test-token")
	require.NotNil(t, c)
	assert.Equal(t, "test-token", c.Token)
	assert.Equal(t, "https://api.github.com/copilot", c.BaseURL)
	assert.NotNil(t, c.HTTPClient)
}

func TestCopilotClient_IsAvailable_WithToken(t *testing.T) {
	c := NewCopilotClient("test-token")
	assert.True(t, c.IsAvailable())
}

func TestCopilotClient_IsAvailable_WithoutToken(t *testing.T) {
	c := NewCopilotClient("")
	assert.False(t, c.IsAvailable())
}

func TestCopilotClient_Complete_NoToken(t *testing.T) {
	c := NewCopilotClient("")
	_, err := c.Complete(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestCopilotClient_Complete_WithToken(t *testing.T) {
	c := NewCopilotClient("test-token")
	result, err := c.Complete(context.Background(), "hello world")
	require.NoError(t, err)
	assert.Contains(t, result, "hello world")
}
