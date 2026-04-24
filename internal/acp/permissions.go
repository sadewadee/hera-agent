// Package acp provides the Agent Client Protocol implementation.
//
// permissions.go implements ACP permission bridging that maps ACP
// approval requests to Hera approval callbacks. When a tool requires
// user approval, this bridges the agent's synchronous approval callback
// to the ACP client's asynchronous permission request flow.
package acp

import (
	"context"
	"log/slog"
	"time"
)

// PermissionOption represents a permission choice offered to the user.
type PermissionOption struct {
	OptionID string `json:"option_id"`
	Kind     string `json:"kind"` // allow_once, allow_always, reject_once, reject_always
	Name     string `json:"name"`
}

// PermissionResponse holds the user's response to a permission request.
type PermissionResponse struct {
	OptionID string `json:"option_id"`
	Outcome  string `json:"outcome"` // "allowed" or "rejected"
}

// PermissionRequestFunc is the function signature for requesting permission
// from the ACP client.
type PermissionRequestFunc func(ctx context.Context, sessionID string, toolCall ToolCallInfo, options []PermissionOption) (*PermissionResponse, error)

// ToolCallInfo describes a tool call pending approval.
type ToolCallInfo struct {
	ID      string `json:"id"`
	Command string `json:"command"`
	Kind    string `json:"kind"` // "execute", "read", "edit", etc.
}

// kindToHera maps ACP PermissionOption kinds to Hera approval result strings.
var kindToHera = map[string]string{
	"allow_once":    "once",
	"allow_always":  "always",
	"reject_once":   "deny",
	"reject_always": "deny",
}

// DefaultPermissionOptions returns the standard set of permission options.
func DefaultPermissionOptions() []PermissionOption {
	return []PermissionOption{
		{OptionID: "allow_once", Kind: "allow_once", Name: "Allow once"},
		{OptionID: "allow_always", Kind: "allow_always", Name: "Allow always"},
		{OptionID: "deny", Kind: "reject_once", Name: "Deny"},
	}
}

// MakeApprovalCallback returns a Hera-compatible approval callback that
// bridges to the ACP client's permission request call.
//
// The returned function has the signature:
//
//	func(command string, description string) string
//
// It returns "once", "always", or "deny".
func MakeApprovalCallback(
	requestPermission PermissionRequestFunc,
	sessionID string,
	timeout time.Duration,
) func(command, description string) string {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	return func(command, description string) string {
		options := DefaultPermissionOptions()
		toolCall := ToolCallInfo{
			ID:      "perm-check",
			Command: command,
			Kind:    "execute",
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		response, err := requestPermission(ctx, sessionID, toolCall, options)
		if err != nil {
			slog.Warn("permission request failed", "error", err)
			return "deny"
		}

		if response == nil || response.Outcome != "allowed" {
			return "deny"
		}

		// Look up the kind from the options list.
		for _, opt := range options {
			if opt.OptionID == response.OptionID {
				if result, ok := kindToHera[opt.Kind]; ok {
					return result
				}
			}
		}

		return "once" // fallback for unknown option_id
	}
}
