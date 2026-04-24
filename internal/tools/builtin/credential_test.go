package builtin

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialTool_Name(t *testing.T) {
	tool := &CredentialTool{}
	assert.Equal(t, "credential", tool.Name())
}

func TestCredentialTool_InvalidArgs(t *testing.T) {
	tool := &CredentialTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestCredentialTool_Get_Found(t *testing.T) {
	os.Setenv("HERA_TEST_CRED", "secret-value")
	defer os.Unsetenv("HERA_TEST_CRED")

	tool := &CredentialTool{}
	args, _ := json.Marshal(credentialArgs{Action: "get", Name: "HERA_TEST_CRED"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "REDACTED")
	assert.Contains(t, result.Content, "12 chars")
}

func TestCredentialTool_Get_NotFound(t *testing.T) {
	os.Unsetenv("HERA_NONEXISTENT_VAR")
	tool := &CredentialTool{}
	args, _ := json.Marshal(credentialArgs{Action: "get", Name: "HERA_NONEXISTENT_VAR"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "not found")
}

func TestCredentialTool_Set(t *testing.T) {
	tool := &CredentialTool{}
	args, _ := json.Marshal(credentialArgs{Action: "set", Name: "MY_KEY", Value: "my-secret"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "stored securely")
}

func TestCredentialTool_List(t *testing.T) {
	tool := &CredentialTool{}
	args, _ := json.Marshal(credentialArgs{Action: "list", Name: "all"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "keychain")
}

func TestCredentialTool_Delete(t *testing.T) {
	tool := &CredentialTool{}
	args, _ := json.Marshal(credentialArgs{Action: "delete", Name: "OLD_KEY"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "deleted")
}

func TestCredentialTool_UnknownAction(t *testing.T) {
	tool := &CredentialTool{}
	args, _ := json.Marshal(credentialArgs{Action: "invalid", Name: "x"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
