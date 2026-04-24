// Package builtin provides built-in tool implementations.
//
// path_security.go provides shared path validation helpers used by
// tools that operate on files: skill manager, skill hub, cronjob
// tools, and credential file tools. Centralises the resolve +
// relative_to traversal check to avoid duplication.
package builtin

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateWithinDir checks that path resolves to a location inside root.
// Returns an error message if validation fails, or empty string if safe.
// Uses filepath.Abs and filepath.EvalSymlinks to follow symlinks and
// normalise ".." components.
func ValidateWithinDir(path, root string) string {
	resolved, err := resolvePathFully(path)
	if err != nil {
		return fmt.Sprintf("path escapes allowed directory: %v", err)
	}
	rootResolved, err := resolvePathFully(root)
	if err != nil {
		return fmt.Sprintf("path escapes allowed directory: %v", err)
	}

	// Ensure the resolved path starts with the resolved root.
	if !strings.HasPrefix(resolved, rootResolved+string(filepath.Separator)) && resolved != rootResolved {
		return fmt.Sprintf("path escapes allowed directory: %s is not within %s", resolved, rootResolved)
	}
	return ""
}

// HasTraversalComponent returns true if pathStr contains ".." traversal
// components. This is a quick check for obvious traversal attempts
// before performing full resolution.
func HasTraversalComponent(pathStr string) bool {
	parts := strings.Split(filepath.ToSlash(pathStr), "/")
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}

// resolvePathFully resolves a path to its absolute, symlink-free form.
func resolvePathFully(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	// EvalSymlinks also cleans the path and resolves symlinks.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If the file doesn't exist yet, fall back to cleaned abs path.
		return filepath.Clean(abs), nil
	}
	return resolved, nil
}
