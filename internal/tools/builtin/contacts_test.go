package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactsTool_Name(t *testing.T) {
	tool := &ContactsTool{}
	assert.Equal(t, "contacts", tool.Name())
}

func TestContactsTool_Description(t *testing.T) {
	tool := &ContactsTool{}
	assert.Contains(t, tool.Description(), "contacts")
}

func TestContactsTool_InvalidArgs(t *testing.T) {
	tool := &ContactsTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestContactsTool_CreateAndGet(t *testing.T) {
	tool := &ContactsTool{}

	// Create a contact
	args, _ := json.Marshal(contactsArgs{Action: "create", Name: "Alice", Email: "alice@test.com", Phone: "555-1234"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Alice")

	var contact Contact
	require.NoError(t, json.Unmarshal([]byte(result.Content), &contact))
	assert.Equal(t, "Alice", contact.Name)
	assert.Equal(t, "alice@test.com", contact.Email)

	// Get the contact
	args2, _ := json.Marshal(contactsArgs{Action: "get", ID: contact.ID})
	result2, err := tool.Execute(context.Background(), args2)
	require.NoError(t, err)
	assert.False(t, result2.IsError)
	assert.Contains(t, result2.Content, "Alice")
}

func TestContactsTool_CreateNoName(t *testing.T) {
	tool := &ContactsTool{}
	args, _ := json.Marshal(contactsArgs{Action: "create"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "name is required")
}

func TestContactsTool_List(t *testing.T) {
	tool := &ContactsTool{}

	// List empty
	args, _ := json.Marshal(contactsArgs{Action: "list"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	// Create two contacts
	args1, _ := json.Marshal(contactsArgs{Action: "create", Name: "Bob"})
	tool.Execute(context.Background(), args1)
	args2, _ := json.Marshal(contactsArgs{Action: "create", Name: "Carol"})
	tool.Execute(context.Background(), args2)

	result2, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result2.Content, "Bob")
	assert.Contains(t, result2.Content, "Carol")
}

func TestContactsTool_Search(t *testing.T) {
	tool := &ContactsTool{}
	args1, _ := json.Marshal(contactsArgs{Action: "create", Name: "Dave", Email: "dave@example.com"})
	tool.Execute(context.Background(), args1)
	args2, _ := json.Marshal(contactsArgs{Action: "create", Name: "Eve"})
	tool.Execute(context.Background(), args2)

	searchArgs, _ := json.Marshal(contactsArgs{Action: "search", Query: "dave"})
	result, err := tool.Execute(context.Background(), searchArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Dave")
}

func TestContactsTool_Update(t *testing.T) {
	tool := &ContactsTool{}
	createArgs, _ := json.Marshal(contactsArgs{Action: "create", Name: "Frank"})
	createResult, _ := tool.Execute(context.Background(), createArgs)
	var c Contact
	json.Unmarshal([]byte(createResult.Content), &c)

	updateArgs, _ := json.Marshal(contactsArgs{Action: "update", ID: c.ID, Email: "frank@new.com"})
	result, err := tool.Execute(context.Background(), updateArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "frank@new.com")
}

func TestContactsTool_UpdateNotFound(t *testing.T) {
	tool := &ContactsTool{}
	args, _ := json.Marshal(contactsArgs{Action: "update", ID: "nonexistent"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "not found")
}

func TestContactsTool_Delete(t *testing.T) {
	tool := &ContactsTool{}
	createArgs, _ := json.Marshal(contactsArgs{Action: "create", Name: "Grace"})
	createResult, _ := tool.Execute(context.Background(), createArgs)
	var c Contact
	json.Unmarshal([]byte(createResult.Content), &c)

	deleteArgs, _ := json.Marshal(contactsArgs{Action: "delete", ID: c.ID})
	result, err := tool.Execute(context.Background(), deleteArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "deleted")
}

func TestContactsTool_DeleteNotFound(t *testing.T) {
	tool := &ContactsTool{}
	args, _ := json.Marshal(contactsArgs{Action: "delete", ID: "nonexistent"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestContactsTool_UnknownAction(t *testing.T) {
	tool := &ContactsTool{}
	args, _ := json.Marshal(contactsArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestRegisterContacts(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterContacts(registry)
	_, ok := registry.Get("contacts")
	assert.True(t, ok)
}
