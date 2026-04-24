package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- EnvKeyForServer ---

func TestEnvKeyForServer(t *testing.T) {
	assert.Equal(t, "MCP_GITHUB_API_KEY", EnvKeyForServer("github"))
	assert.Equal(t, "MCP_MY_SERVER_API_KEY", EnvKeyForServer("my-server"))
	assert.Equal(t, "MCP_CONTEXT7_API_KEY", EnvKeyForServer("context7"))
}

// --- ParseEnvAssignments ---

func TestParseEnvAssignments_Valid(t *testing.T) {
	result, err := ParseEnvAssignments([]string{"KEY=value", "FOO=bar"})
	require.NoError(t, err)
	assert.Equal(t, "value", result["KEY"])
	assert.Equal(t, "bar", result["FOO"])
}

func TestParseEnvAssignments_Empty(t *testing.T) {
	result, err := ParseEnvAssignments(nil)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseEnvAssignments_SkipsEmpty(t *testing.T) {
	result, err := ParseEnvAssignments([]string{"  ", "", "KEY=val"})
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestParseEnvAssignments_NoEqualsSign(t *testing.T) {
	_, err := ParseEnvAssignments([]string{"INVALID"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "KEY=VALUE")
}

func TestParseEnvAssignments_EmptyKey(t *testing.T) {
	_, err := ParseEnvAssignments([]string{"=value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing variable name")
}

func TestParseEnvAssignments_InvalidKeyChars(t *testing.T) {
	_, err := ParseEnvAssignments([]string{"bad-key=value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --env variable name")
}

func TestParseEnvAssignments_ValueWithEquals(t *testing.T) {
	result, err := ParseEnvAssignments([]string{"KEY=val=ue=more"})
	require.NoError(t, err)
	assert.Equal(t, "val=ue=more", result["KEY"])
}

// --- ApplyMCPPreset ---

func TestApplyMCPPreset_NoPreset(t *testing.T) {
	url, cmd, _, applied, err := ApplyMCPPreset("", "http://existing", "", nil)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, "http://existing", url)
	assert.Empty(t, cmd)
}

func TestApplyMCPPreset_UnknownPreset(t *testing.T) {
	_, _, _, _, err := ApplyMCPPreset("nonexistent", "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown MCP preset")
}

func TestApplyMCPPreset_ExistingOverride(t *testing.T) {
	MCPPresets["test-preset"] = &MCPPreset{URL: "http://preset.example.com"}
	defer delete(MCPPresets, "test-preset")

	url, cmd, args, applied, err := ApplyMCPPreset("test-preset", "http://override", "", nil)
	require.NoError(t, err)
	assert.False(t, applied) // existing URL takes precedence
	assert.Equal(t, "http://override", url)
	assert.Empty(t, cmd)
	assert.Nil(t, args)
}

func TestApplyMCPPreset_Applied(t *testing.T) {
	MCPPresets["test-preset"] = &MCPPreset{
		URL:     "http://preset.example.com",
		Command: "npx",
		Args:    []string{"-y", "mcp-server"},
	}
	defer delete(MCPPresets, "test-preset")

	url, cmd, args, applied, err := ApplyMCPPreset("test-preset", "", "", nil)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, "http://preset.example.com", url)
	assert.Equal(t, "npx", cmd)
	assert.Equal(t, []string{"-y", "mcp-server"}, args)
}

// --- InterpolateValue ---

func TestInterpolateValue(t *testing.T) {
	t.Setenv("TEST_INTERP_KEY", "resolved")
	assert.Equal(t, "resolved", InterpolateValue("${TEST_INTERP_KEY}"))
}

func TestInterpolateValue_NoVar(t *testing.T) {
	assert.Equal(t, "plain text", InterpolateValue("plain text"))
}

func TestInterpolateValue_Unset(t *testing.T) {
	os.Unsetenv("TOTALLY_UNSET_VAR_XYZ")
	assert.Equal(t, "", InterpolateValue("${TOTALLY_UNSET_VAR_XYZ}"))
}

// --- GetMCPServers ---

func TestGetMCPServers_Present(t *testing.T) {
	config := map[string]any{
		"mcp_servers": map[string]any{
			"github": map[string]any{"url": "http://github.example.com"},
		},
	}
	servers := GetMCPServers(config)
	assert.Len(t, servers, 1)
	assert.Equal(t, "http://github.example.com", servers["github"]["url"])
}

func TestGetMCPServers_Missing(t *testing.T) {
	servers := GetMCPServers(map[string]any{})
	assert.Empty(t, servers)
}

// --- FormatMCPList ---

func TestFormatMCPList_URLTransport(t *testing.T) {
	servers := map[string]map[string]any{
		"server1": {"url": "http://example.com/api"},
	}
	entries := FormatMCPList(servers)
	assert.Len(t, entries, 1)
	assert.Equal(t, "server1", entries[0].Name)
	assert.Equal(t, "http://example.com/api", entries[0].Transport)
	assert.True(t, entries[0].Enabled)
}

func TestFormatMCPList_CommandTransport(t *testing.T) {
	servers := map[string]map[string]any{
		"server1": {"command": "npx", "args": []any{"-y", "mcp-server"}},
	}
	entries := FormatMCPList(servers)
	assert.Len(t, entries, 1)
	assert.Contains(t, entries[0].Transport, "npx")
}

func TestFormatMCPList_DisabledBool(t *testing.T) {
	servers := map[string]map[string]any{
		"server1": {"url": "http://x", "enabled": false},
	}
	entries := FormatMCPList(servers)
	assert.False(t, entries[0].Enabled)
}

func TestFormatMCPList_DisabledString(t *testing.T) {
	servers := map[string]map[string]any{
		"server1": {"url": "http://x", "enabled": "false"},
	}
	entries := FormatMCPList(servers)
	assert.False(t, entries[0].Enabled)
}

func TestFormatMCPList_ToolsInclude(t *testing.T) {
	servers := map[string]map[string]any{
		"server1": {
			"url":   "http://x",
			"tools": map[string]any{"include": []any{"tool1", "tool2"}},
		},
	}
	entries := FormatMCPList(servers)
	assert.Equal(t, "2 selected", entries[0].ToolsStr)
}

func TestFormatMCPList_ToolsExclude(t *testing.T) {
	servers := map[string]map[string]any{
		"server1": {
			"url":   "http://x",
			"tools": map[string]any{"exclude": []any{"tool1"}},
		},
	}
	entries := FormatMCPList(servers)
	assert.Equal(t, "-1 excluded", entries[0].ToolsStr)
}

func TestFormatMCPList_LongURLTruncated(t *testing.T) {
	longURL := "http://example.com/very/long/path/that/exceeds/twenty/eight/chars"
	servers := map[string]map[string]any{
		"s": {"url": longURL},
	}
	entries := FormatMCPList(servers)
	assert.True(t, len(entries[0].Transport) <= 28)
	assert.True(t, strings.HasSuffix(entries[0].Transport, "..."))
}

// --- MCPTestTimeout ---

func TestMCPTestTimeout(t *testing.T) {
	assert.Equal(t, 30*time.Second, MCPTestTimeout)
}
