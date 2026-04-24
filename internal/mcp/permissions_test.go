package mcp

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionStore_NewPermissionStore(t *testing.T) {
	ps := NewPermissionStore()
	require.NotNil(t, ps)
}

func TestPermissionStore_Request(t *testing.T) {
	ps := NewPermissionStore()
	p := ps.Request("command_exec", "rm -rf /", "agent1")
	require.NotNil(t, p)
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "command_exec", p.Type)
	assert.Equal(t, "rm -rf /", p.Resource)
	assert.Equal(t, "agent1", p.Requester)
	assert.Equal(t, "pending", p.Status)
	assert.False(t, p.CreatedAt.IsZero())
	assert.Nil(t, p.ResolvedAt)
}

func TestPermissionStore_ListPending(t *testing.T) {
	ps := NewPermissionStore()
	ps.Request("command_exec", "ls", "agent1")
	ps.Request("file_write", "/tmp/x", "agent2")

	pending := ps.ListPending()
	assert.Len(t, pending, 2)
}

func TestPermissionStore_ListPending_ExcludesResolved(t *testing.T) {
	ps := NewPermissionStore()
	p1 := ps.Request("command_exec", "ls", "agent1")
	ps.Request("file_write", "/tmp/x", "agent2")
	ps.Respond(p1.ID, "approve")

	pending := ps.ListPending()
	assert.Len(t, pending, 1)
}

func TestPermissionStore_Respond_Approve(t *testing.T) {
	ps := NewPermissionStore()
	p := ps.Request("command_exec", "ls", "agent1")

	result, err := ps.Respond(p.ID, "approve")
	require.NoError(t, err)
	assert.Equal(t, "approved", result.Status)
	assert.NotNil(t, result.ResolvedAt)
}

func TestPermissionStore_Respond_Deny(t *testing.T) {
	ps := NewPermissionStore()
	p := ps.Request("command_exec", "rm -rf /", "agent1")

	result, err := ps.Respond(p.ID, "deny")
	require.NoError(t, err)
	assert.Equal(t, "denied", result.Status)
	assert.NotNil(t, result.ResolvedAt)
}

func TestPermissionStore_Respond_InvalidAction(t *testing.T) {
	ps := NewPermissionStore()
	p := ps.Request("command_exec", "ls", "agent1")

	_, err := ps.Respond(p.ID, "maybe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")
}

func TestPermissionStore_Respond_NotFound(t *testing.T) {
	ps := NewPermissionStore()
	_, err := ps.Respond("nonexistent-id", "approve")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPermissionStore_Respond_AlreadyResolved(t *testing.T) {
	ps := NewPermissionStore()
	p := ps.Request("command_exec", "ls", "agent1")
	ps.Respond(p.ID, "approve")

	_, err := ps.Respond(p.ID, "deny")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestPermissionStore_Get(t *testing.T) {
	ps := NewPermissionStore()
	p := ps.Request("command_exec", "ls", "agent1")

	got, ok := ps.Get(p.ID)
	assert.True(t, ok)
	assert.Equal(t, p.ID, got.ID)
}

func TestPermissionStore_Get_NotFound(t *testing.T) {
	ps := NewPermissionStore()
	_, ok := ps.Get("nonexistent")
	assert.False(t, ok)
}

func TestPermissionStore_ConcurrentAccess(t *testing.T) {
	ps := NewPermissionStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := ps.Request("cmd", "ls", "agent")
			ps.Get(p.ID)
			ps.ListPending()
		}()
	}
	wg.Wait()
}
