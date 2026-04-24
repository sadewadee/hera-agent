package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserIDContext_RoundTrip(t *testing.T) {
	ctx := WithUserID(context.Background(), "telegram-810485832")
	assert.Equal(t, "telegram-810485832", UserIDFromContext(ctx))
}

func TestUserIDContext_EmptyIsNoOp(t *testing.T) {
	base := context.Background()
	ctx := WithUserID(base, "")
	// No key stored => lookup returns empty string.
	assert.Equal(t, "", UserIDFromContext(ctx))
}

func TestUserIDContext_NilContext(t *testing.T) {
	//nolint:staticcheck // intentionally passing nil to verify safety
	assert.Equal(t, "", UserIDFromContext(nil))
}

func TestUserIDContext_NotSet(t *testing.T) {
	assert.Equal(t, "", UserIDFromContext(context.Background()))
}

func TestUserIDContext_Override(t *testing.T) {
	ctx := WithUserID(context.Background(), "first")
	ctx = WithUserID(ctx, "second")
	assert.Equal(t, "second", UserIDFromContext(ctx))
}
