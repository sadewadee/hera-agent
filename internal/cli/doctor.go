package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sadewadee/hera/internal/config"
)

// checkResult represents the result of a single diagnostic check.
type checkResult struct {
	Name    string
	Status  string // "ok", "warn", "fail"
	Message string
}

// RunDoctor runs all diagnostic checks and prints results.
func RunDoctor() error {
	fmt.Println()
	fmt.Println("=== Hera Doctor ===")
	fmt.Println()
	fmt.Println("Running diagnostic checks...")
	fmt.Println()

	checks := []checkResult{
		checkGoVersion(),
		checkConfigFile(),
		checkAPIKeys(),
		checkHeraDir(),
		checkDatabase(),
		checkPlatform(),
	}

	passCount := 0
	warnCount := 0
	failCount := 0

	for _, c := range checks {
		icon := statusIcon(c.Status)
		fmt.Printf("  %s %-30s %s\n", icon, c.Name, c.Message)

		switch c.Status {
		case "ok":
			passCount++
		case "warn":
			warnCount++
		case "fail":
			failCount++
		}
	}

	fmt.Println()
	fmt.Printf("Results: %d passed, %d warnings, %d failed\n", passCount, warnCount, failCount)

	if failCount > 0 {
		fmt.Println()
		fmt.Println("Some checks failed. Run 'hera setup' to fix configuration issues.")
	} else if warnCount > 0 {
		fmt.Println()
		fmt.Println("Some warnings detected. Hera should work but with limited functionality.")
	} else {
		fmt.Println()
		fmt.Println("All checks passed. Hera is ready to use!")
	}

	return nil
}

func statusIcon(status string) string {
	switch status {
	case "ok":
		return "[PASS]"
	case "warn":
		return "[WARN]"
	case "fail":
		return "[FAIL]"
	default:
		return "[????]"
	}
}

func checkGoVersion() checkResult {
	ver := runtime.Version()
	return checkResult{
		Name:    "Go Runtime",
		Status:  "ok",
		Message: ver,
	}
}

func checkConfigFile() checkResult {
	configPath := filepath.Join(config.HeraDir(), "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return checkResult{
			Name:    "Configuration File",
			Status:  "fail",
			Message: fmt.Sprintf("not found at %s (run 'hera setup')", configPath),
		}
	}

	_, err := config.Load()
	if err != nil {
		return checkResult{
			Name:    "Configuration File",
			Status:  "fail",
			Message: fmt.Sprintf("parse error: %v", err),
		}
	}

	return checkResult{
		Name:    "Configuration File",
		Status:  "ok",
		Message: configPath,
	}
}

func checkAPIKeys() checkResult {
	// Check for common API keys in environment
	keys := []struct {
		Name string
		Env  string
	}{
		{"OpenAI", "OPENAI_API_KEY"},
		{"Anthropic", "ANTHROPIC_API_KEY"},
		{"Gemini", "GEMINI_API_KEY"},
		{"Mistral", "MISTRAL_API_KEY"},
		{"OpenRouter", "OPENROUTER_API_KEY"},
	}

	var found []string
	for _, k := range keys {
		if v := os.Getenv(k.Env); v != "" {
			found = append(found, k.Name)
		}
	}

	if len(found) == 0 {
		// Also check config file
		cfg, err := config.Load()
		if err == nil {
			for name, p := range cfg.Provider {
				if p.APIKey != "" {
					found = append(found, name)
				}
			}
		}
	}

	if len(found) == 0 {
		return checkResult{
			Name:    "API Keys",
			Status:  "warn",
			Message: "no API keys found (set via env vars or config)",
		}
	}

	return checkResult{
		Name:    "API Keys",
		Status:  "ok",
		Message: fmt.Sprintf("found: %s", strings.Join(found, ", ")),
	}
}

func checkHeraDir() checkResult {
	heraDir := config.HeraDir()
	info, err := os.Stat(heraDir)
	if os.IsNotExist(err) {
		return checkResult{
			Name:    "Hera Directory",
			Status:  "warn",
			Message: fmt.Sprintf("%s does not exist (will be created on first use)", heraDir),
		}
	}
	if !info.IsDir() {
		return checkResult{
			Name:    "Hera Directory",
			Status:  "fail",
			Message: fmt.Sprintf("%s exists but is not a directory", heraDir),
		}
	}

	return checkResult{
		Name:    "Hera Directory",
		Status:  "ok",
		Message: heraDir,
	}
}

func checkDatabase() checkResult {
	cfg, err := config.Load()
	if err != nil {
		return checkResult{
			Name:    "Database",
			Status:  "warn",
			Message: "cannot load config to check database",
		}
	}

	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = filepath.Join(config.HeraDir(), "hera.db")
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return checkResult{
			Name:    "Database",
			Status:  "warn",
			Message: fmt.Sprintf("database not found at %s (will be created on first use)", dbPath),
		}
	}

	return checkResult{
		Name:    "Database",
		Status:  "ok",
		Message: dbPath,
	}
}

func checkPlatform() checkResult {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Check for optional platform-specific tools
	var available []string
	optionalTools := []string{"git", "docker", "sqlite3"}
	for _, tool := range optionalTools {
		if _, err := exec.LookPath(tool); err == nil {
			available = append(available, tool)
		}
	}

	msg := fmt.Sprintf("%s/%s", osName, arch)
	if len(available) > 0 {
		msg += fmt.Sprintf(" (tools: %s)", strings.Join(available, ", "))
	}

	return checkResult{
		Name:    "Platform",
		Status:  "ok",
		Message: msg,
	}
}
