// Package cli provides the Hera CLI application.
//
// env_loader.go provides helpers for loading Hera .env files consistently
// across all entrypoints (CLI, gateway, cron).
package cli

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
)

// LoadHeraDotenv loads Hera environment files with user config taking precedence.
//
// Behavior:
//   - ~/.hera/.env overrides stale shell-exported values when present.
//   - Project .env acts as a dev fallback and only fills missing values when
//     the user env exists.
//   - If no user env exists, the project .env also overrides stale shell vars.
func LoadHeraDotenv(heraHome, projectEnv string) []string {
	var loaded []string

	if heraHome == "" {
		heraHome = paths.HeraHome()
	}

	userEnv := filepath.Join(heraHome, ".env")
	if envFileExists(userEnv) {
		loadDotenvFile(userEnv, true)
		loaded = append(loaded, userEnv)
	}

	if projectEnv != "" && envFileExists(projectEnv) {
		// If user env was loaded, project env only fills missing values.
		loadDotenvFile(projectEnv, len(loaded) == 0)
		loaded = append(loaded, projectEnv)
	}

	return loaded
}

// loadDotenvFile reads a .env file and sets environment variables.
// If override is true, existing values are overwritten.
func loadDotenvFile(path string, override bool) {
	f, err := os.Open(path)
	if err != nil {
		slog.Debug("failed to open env file", "path", path, "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if key == "" {
			continue
		}

		if override || os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func envFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
