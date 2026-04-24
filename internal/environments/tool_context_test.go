package environments

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolContext(t *testing.T) {
	ctx := context.Background()
	tc := NewToolContext(ctx, "test-env", "session-1", "/tmp/work")
	require.NotNil(t, tc)
	assert.Equal(t, "test-env", tc.Environment())
	assert.Equal(t, "session-1", tc.SessionID())
	assert.Equal(t, "/tmp/work", tc.WorkDir())
	assert.Equal(t, ctx, tc.Context())
}

func TestToolContext_SetAndGet(t *testing.T) {
	tc := NewToolContext(context.Background(), "env", "s1", "/tmp")

	tc.Set("key1", "value1")
	tc.Set("key2", 42)

	v1, ok := tc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v1)

	v2, ok := tc.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, v2)
}

func TestToolContext_GetNotFound(t *testing.T) {
	tc := NewToolContext(context.Background(), "env", "s1", "/tmp")
	_, ok := tc.Get("missing")
	assert.False(t, ok)
}

func TestToolContext_OverwriteMetadata(t *testing.T) {
	tc := NewToolContext(context.Background(), "env", "s1", "/tmp")
	tc.Set("key", "old")
	tc.Set("key", "new")
	v, ok := tc.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "new", v)
}
