package builtin

import (
	"fmt"

	"github.com/sadewadee/hera/internal/tools"
)

// checkCommandSafety returns a non-nil error Result if the command matches a
// dangerous pattern. Pass non-nil protectedPaths to also block path access.
// Returns nil when the command is safe to proceed.
func checkCommandSafety(command string, protectedPaths []string) *tools.Result {
	if pattern, dangerous := isDangerous(command); dangerous {
		return &tools.Result{
			Content: fmt.Sprintf("blocked: command matches dangerous pattern %q — requires explicit approval", pattern),
			IsError: true,
		}
	}
	if len(protectedPaths) > 0 {
		if path, protected := accessesProtectedPath(command, protectedPaths); protected {
			return &tools.Result{
				Content: fmt.Sprintf("blocked: command accesses protected path %q", path),
				IsError: true,
			}
		}
	}
	return nil
}
