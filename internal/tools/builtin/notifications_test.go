package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationsTool_Name(t *testing.T) {
	tool := &NotificationsTool{}
	assert.Equal(t, "notifications", tool.Name())
}

func TestNotificationsTool_Description(t *testing.T) {
	tool := &NotificationsTool{}
	assert.Contains(t, tool.Description(), "notification")
}

func TestNotificationsTool_Parameters(t *testing.T) {
	tool := &NotificationsTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestNotificationsTool_InvalidArgs(t *testing.T) {
	tool := &NotificationsTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid arguments")
}

func TestRegisterNotifications(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterNotifications(registry)
	_, ok := registry.Get("notifications")
	assert.True(t, ok)
}
