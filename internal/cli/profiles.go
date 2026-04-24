// Package cli provides the Hera CLI application.
//
// profiles.go implements profile management for multiple isolated Hera
// instances. Each profile is a fully independent HERA_HOME directory
// with its own config, env, memory, sessions, skills, gateway, cron, logs.
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
)

var profileIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// Directories bootstrapped inside every new profile.
var profileDirs = []string{
	"memories", "sessions", "skills", "skins", "logs",
	"plans", "workspace", "cron", "home",
}

// Files copied during --clone.
var cloneConfigFiles = []string{
	"config.yaml", ".env", "SOUL.md",
}

// Profile represents a Hera profile.
type Profile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Active  bool   `json:"active"`
	Default bool   `json:"default"`
}

// ProfilesRoot returns the directory containing all profiles.
func ProfilesRoot() string {
	return filepath.Join(paths.HeraHome(), "profiles")
}

// ProfilePath returns the path for a named profile.
func ProfilePath(name string) string {
	if name == "default" || name == "" {
		return paths.HeraHome()
	}
	return filepath.Join(ProfilesRoot(), name)
}

// ValidateProfileID checks if a profile name is valid.
func ValidateProfileID(name string) error {
	if !profileIDRE.MatchString(name) {
		return fmt.Errorf("invalid profile name '%s': must match %s", name, profileIDRE.String())
	}
	if name == "default" {
		return fmt.Errorf("'default' is reserved for the main ~/.hera directory")
	}
	return nil
}

// ListProfiles returns all available profiles.
func ListProfiles() []Profile {
	var profiles []Profile

	// Default profile.
	defaultPath := ProfilePath("default")
	profiles = append(profiles, Profile{
		Name:    "default",
		Path:    defaultPath,
		Default: true,
	})

	// Named profiles.
	root := ProfilesRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return profiles
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		profiles = append(profiles, Profile{
			Name: name,
			Path: filepath.Join(root, name),
		})
	}
	return profiles
}

// CreateProfile creates a new profile directory with bootstrapped subdirs.
func CreateProfile(name string, clone bool, sourceProfile string) error {
	if err := ValidateProfileID(name); err != nil {
		return err
	}

	profileDir := ProfilePath(name)
	if _, err := os.Stat(profileDir); err == nil {
		return fmt.Errorf("profile '%s' already exists at %s", name, profileDir)
	}

	// Create directory tree.
	for _, dir := range profileDirs {
		dirPath := filepath.Join(profileDir, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dirPath, err)
		}
	}

	// Seed default SOUL.md.
	soulPath := filepath.Join(profileDir, "SOUL.md")
	if err := os.WriteFile(soulPath, []byte(DefaultSoulMD), 0o644); err != nil {
		slog.Warn("failed to seed SOUL.md", "error", err)
	}

	// Clone config files if requested.
	if clone && sourceProfile != "" {
		sourcePath := ProfilePath(sourceProfile)
		for _, file := range cloneConfigFiles {
			src := filepath.Join(sourcePath, file)
			dst := filepath.Join(profileDir, file)
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				slog.Warn("failed to clone file",
					"file", file,
					"error", err,
				)
			}
		}
	}

	slog.Info("profile created", "name", name, "path", profileDir)
	return nil
}

// DeleteProfile removes a profile directory.
func DeleteProfile(name string) error {
	if err := ValidateProfileID(name); err != nil {
		return err
	}

	profileDir := ProfilePath(name)
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	return os.RemoveAll(profileDir)
}

// ProfileCommand routes profile subcommands.
func ProfileCommand(action, name string, clone bool) {
	switch action {
	case "list", "ls", "":
		profiles := ListProfiles()
		fmt.Println("\nProfiles:")
		for _, p := range profiles {
			marker := ""
			if p.Default {
				marker = " (default)"
			}
			fmt.Printf("  %s%s  %s\n", p.Name, marker, p.Path)
		}
		fmt.Println()

	case "create":
		if name == "" {
			fmt.Println("Usage: hera profile create <name> [--clone]")
			return
		}
		if err := CreateProfile(name, clone, "default"); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Profile '%s' created at %s\n", name, ProfilePath(name))

	case "delete", "rm":
		if name == "" {
			fmt.Println("Usage: hera profile delete <name>")
			return
		}
		if err := DeleteProfile(name); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Profile '%s' deleted\n", name)

	case "use":
		if name == "" {
			fmt.Println("Usage: hera profile use <name>")
			return
		}
		fmt.Printf("Active profile: %s\n", name)

	default:
		fmt.Printf("Unknown profile command: %s\n", action)
	}
}

// ResolveProfile resolves the HERA_HOME for a given profile flag.
func ResolveProfile(profileFlag string) string {
	if profileFlag == "" || profileFlag == "default" {
		return ProfilePath("default")
	}
	normalized := strings.TrimSpace(strings.ToLower(profileFlag))
	return ProfilePath(normalized)
}
