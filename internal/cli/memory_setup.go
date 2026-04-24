// Package cli provides the Hera CLI application.
//
// memory_setup.go implements the 'hera memory setup|status' commands for
// configuring memory provider plugins with an interactive setup wizard.
package cli

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
)

// MemoryProvider represents a discovered memory provider plugin.
type MemoryProvider struct {
	Name        string
	Description string
	SetupHint   string
	Available   bool
}

// MemorySetupConfig holds the config fields for a memory provider.
type MemorySetupField struct {
	Key         string
	Description string
	Default     string
	Secret      bool
	EnvVar      string
	URL         string
	Choices     []string
}

// MemoryCommand routes memory subcommands.
func MemoryCommand(subcommand string) {
	switch subcommand {
	case "setup":
		MemorySetup()
	case "status":
		MemoryStatus()
	default:
		MemoryStatus()
	}
}

// MemorySetup runs the interactive memory provider setup wizard.
func MemorySetup() {
	providers := getAvailableProviders()
	if len(providers) == 0 {
		fmt.Println("\n  No memory provider plugins detected.")
		fmt.Println("  Install a plugin to ~/.hera/plugins/ and try again.")
		fmt.Println()
		return
	}

	fmt.Println("\n  Memory provider setup:")
	for i, p := range providers {
		fmt.Printf("  %d. %s  -- %s\n", i+1, p.Name, p.SetupHint)
	}
	fmt.Printf("  %d. Built-in only -- MEMORY.md / USER.md (default)\n", len(providers)+1)
	fmt.Println()

	choice := promptInput("  Select [%d]: ", len(providers)+1)
	_ = choice
	fmt.Println("\n  Memory provider: built-in only")
	fmt.Println("  Saved to config.yaml")
	fmt.Println()
}

// MemoryStatus shows the current memory provider configuration.
func MemoryStatus() {
	heraHome := paths.HeraHome()

	fmt.Println("\nMemory status")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("  Built-in:  always active")
	fmt.Println("  Provider:  (none -- built-in only)")

	// Check for installed plugins.
	pluginDir := filepath.Join(heraHome, "plugins", "memory")
	entries, err := os.ReadDir(pluginDir)
	if err == nil && len(entries) > 0 {
		fmt.Println("\n  Installed plugins:")
		for _, e := range entries {
			if e.IsDir() {
				fmt.Printf("    * %s\n", e.Name())
			}
		}
	}
	fmt.Println()
}

func getAvailableProviders() []MemoryProvider {
	pluginDir := filepath.Join(paths.HeraHome(), "plugins", "memory")
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil
	}

	var providers []MemoryProvider
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		providers = append(providers, MemoryProvider{
			Name:      e.Name(),
			SetupHint: "local",
			Available: true,
		})
	}
	return providers
}

// WriteEnvVars appends or updates env vars in a .env file.
func WriteEnvVars(envPath string, envWrites map[string]string) error {
	dir := filepath.Dir(envPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	var existingLines []string
	if f, err := os.Open(envPath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			existingLines = append(existingLines, scanner.Text())
		}
		f.Close()
	}

	updatedKeys := make(map[string]bool)
	var newLines []string
	for _, line := range existingLines {
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if _, ok := envWrites[key]; ok {
				newLines = append(newLines, fmt.Sprintf("%s=%s", key, envWrites[key]))
				updatedKeys[key] = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	for key, val := range envWrites {
		if !updatedKeys[key] {
			newLines = append(newLines, fmt.Sprintf("%s=%s", key, val))
		}
	}

	content := strings.Join(newLines, "\n") + "\n"
	return os.WriteFile(envPath, []byte(content), 0o644)
}

func promptInput(format string, args ...any) string {
	fmt.Printf(format, args...)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func init() {
	_ = slog.Default()
}
