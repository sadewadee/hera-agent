package acp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPermissionOptions(t *testing.T) {
	opts := DefaultPermissionOptions()
	assert.Len(t, opts, 3)

	kinds := make(map[string]bool)
	for _, o := range opts {
		kinds[o.Kind] = true
		assert.NotEmpty(t, o.OptionID)
		assert.NotEmpty(t, o.Name)
	}
	assert.True(t, kinds["allow_once"])
	assert.True(t, kinds["allow_always"])
	assert.True(t, kinds["reject_once"])
}

func TestMakeApprovalCallback_AllowOnce(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		return &PermissionResponse{OptionID: "allow_once", Outcome: "allowed"}, nil
	}

	cb := MakeApprovalCallback(requestFn, "session-1", 5*time.Second)
	result := cb("echo hello", "run a shell command")
	assert.Equal(t, "once", result)
}

func TestMakeApprovalCallback_AllowAlways(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		return &PermissionResponse{OptionID: "allow_always", Outcome: "allowed"}, nil
	}

	cb := MakeApprovalCallback(requestFn, "session-2", 5*time.Second)
	result := cb("rm -rf /tmp/test", "delete temp files")
	assert.Equal(t, "always", result)
}

func TestMakeApprovalCallback_Deny(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		return &PermissionResponse{OptionID: "deny", Outcome: "rejected"}, nil
	}

	cb := MakeApprovalCallback(requestFn, "session-3", 5*time.Second)
	result := cb("rm -rf /", "dangerous command")
	assert.Equal(t, "deny", result)
}

func TestMakeApprovalCallback_Error(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		return nil, errors.New("network error")
	}

	cb := MakeApprovalCallback(requestFn, "session-4", 5*time.Second)
	result := cb("some-command", "some description")
	assert.Equal(t, "deny", result)
}

func TestMakeApprovalCallback_NilResponse(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		return nil, nil
	}

	cb := MakeApprovalCallback(requestFn, "session-5", 5*time.Second)
	result := cb("cmd", "desc")
	assert.Equal(t, "deny", result)
}

func TestMakeApprovalCallback_ZeroTimeout(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		// Verify context has a deadline
		_, hasDeadline := ctx.Deadline()
		assert.True(t, hasDeadline)
		return &PermissionResponse{OptionID: "allow_once", Outcome: "allowed"}, nil
	}

	// Zero timeout should default to 60s
	cb := MakeApprovalCallback(requestFn, "session-6", 0)
	result := cb("cmd", "desc")
	assert.Equal(t, "once", result)
}

func TestMakeApprovalCallback_UnknownOptionID(t *testing.T) {
	requestFn := func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error) {
		// Return an unknown option_id
		return &PermissionResponse{OptionID: "unknown-option", Outcome: "allowed"}, nil
	}

	cb := MakeApprovalCallback(requestFn, "session-7", 5*time.Second)
	result := cb("cmd", "desc")
	// Falls back to "once"
	assert.Equal(t, "once", result)
}

func TestToolCallInfo_Fields(t *testing.T) {
	tc := ToolCallInfo{
		ID:      "tc-1",
		Command: "ls -la",
		Kind:    "execute",
	}
	assert.Equal(t, "tc-1", tc.ID)
	assert.Equal(t, "ls -la", tc.Command)
	assert.Equal(t, "execute", tc.Kind)
}

func TestPermissionOption_Fields(t *testing.T) {
	opt := PermissionOption{
		OptionID: "allow_once",
		Kind:     "allow_once",
		Name:     "Allow once",
	}
	assert.Equal(t, "allow_once", opt.OptionID)
	assert.Equal(t, "allow_once", opt.Kind)
	assert.Equal(t, "Allow once", opt.Name)
}
