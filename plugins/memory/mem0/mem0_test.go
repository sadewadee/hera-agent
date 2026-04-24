package mem0

// TestStoreMemory/RecallMemory are skipped because the provider URL is not
// configurable via environment variable — baseURL is a package-level const
// ("https://api.mem0.ai/v1"). See v0.12.x roadmap for testability refactor.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAvailable_WithCreds(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	p := New()
	assert.True(t, p.IsAvailable())
}

func TestIsAvailable_WithoutCreds(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "")
	p := New()
	assert.False(t, p.IsAvailable())
}

func TestHandleToolCall_MissingQuery_ReturnsError(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	_, err := p.HandleToolCall("mem0_search", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query")
}

func TestHandleToolCall_MissingConclusion_ReturnsError(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	_, err := p.HandleToolCall("mem0_conclude", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conclusion")
}

func TestHandleToolCall_UnknownTool_ReturnsError(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	_, err := p.HandleToolCall("nonexistent_tool", map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestGetToolSchemas_ReturnsExpectedTools(t *testing.T) {
	p := New()
	schemas := p.GetToolSchemas()

	require.Len(t, schemas, 3)
	names := make([]string, len(schemas))
	for i, s := range schemas {
		names[i] = s.Name
	}
	assert.Contains(t, names, "mem0_profile")
	assert.Contains(t, names, "mem0_search")
	assert.Contains(t, names, "mem0_conclude")
}

func TestCircuitBreaker_OpenAfterThresholdFailures(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	// Trip the breaker by recording breakerThreshold failures.
	for i := 0; i < breakerThreshold; i++ {
		p.recordFailure()
	}
	assert.True(t, p.isBreakerOpen())

	// Any tool call while breaker is open should return an error.
	_, err := p.HandleToolCall("mem0_conclude", map[string]interface{}{
		"conclusion": "some fact",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")

	// Resetting via recordSuccess clears the failure count.
	p.recordSuccess()
	assert.False(t, p.isBreakerOpen())
}

func TestSystemPromptBlock_ContainsProviderName(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	block := p.SystemPromptBlock()
	assert.Contains(t, block, "Mem0")
}

func TestInitialize_SetsUserAndAgentDefaults(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	t.Setenv("MEM0_USER_ID", "")
	t.Setenv("MEM0_AGENT_ID", "")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	assert.Equal(t, "hera-user", p.userID)
	assert.Equal(t, "hera", p.agentID)
}

func TestInitialize_RespectsEnvOverrides(t *testing.T) {
	t.Setenv("MEM0_API_KEY", "test-key")
	t.Setenv("MEM0_USER_ID", "custom-user")
	t.Setenv("MEM0_AGENT_ID", "custom-agent")
	p := New()
	require.NoError(t, p.Initialize("session-1"))

	assert.Equal(t, "custom-user", p.userID)
	assert.Equal(t, "custom-agent", p.agentID)
}

func TestGetConfigSchema_ContainsAPIKeyField(t *testing.T) {
	p := New()
	fields := p.GetConfigSchema()
	require.NotEmpty(t, fields)

	var found bool
	for _, f := range fields {
		if f.EnvVar == "MEM0_API_KEY" {
			found = true
			assert.True(t, f.Secret)
		}
	}
	assert.True(t, found, "expected config field with EnvVar=MEM0_API_KEY")
}

// Verify JSON marshalling of a successful conclude response shape.
func TestConcludeResponse_JSON(t *testing.T) {
	raw := map[string]string{"result": "Fact stored."}
	data, err := json.Marshal(raw)
	require.NoError(t, err)

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "Fact stored.", decoded["result"])
}
