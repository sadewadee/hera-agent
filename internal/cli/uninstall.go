package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
)

// UninstallOptions controls which components to remove.
type UninstallOptions struct {
	RemoveBinary  bool
	RemoveConfig  bool
	RemoveService bool
	Force         bool // skip confirmation prompts
}

// RunUninstall performs the uninstall process, removing the binary, config directory,
// and systemd service as requested.
func RunUninstall(opts UninstallOptions) error {
	reader := bufio.NewReader(os.Stdin)

	// 1. Remove systemd service (Linux only).
	if opts.RemoveService && runtime.GOOS == "linux" {
		servicePath := "/etc/systemd/system/hera.service"
		if fileExists(servicePath) {
			if opts.Force || confirmAction(reader, "Remove systemd service hera.service?") {
				// Stop and disable the service.
				_ = exec.Command("systemctl", "stop", "hera").Run()
				_ = exec.Command("systemctl", "disable", "hera").Run()
				if err := os.Remove(servicePath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", servicePath, err)
				} else {
					_ = exec.Command("systemctl", "daemon-reload").Run()
					fmt.Println("Removed systemd service.")
				}
			}
		}
	}

	// 2. Remove configuration directory.
	if opts.RemoveConfig {
		configDir := paths.HeraHome()
		if dirExists(configDir) {
			if opts.Force || confirmAction(reader, fmt.Sprintf("Remove configuration directory %s?", configDir)) {
				if err := os.RemoveAll(configDir); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", configDir, err)
				} else {
					fmt.Printf("Removed %s\n", configDir)
				}
			}
		}
	}

	// 3. Remove the binary.
	if opts.RemoveBinary {
		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine binary path: %w", err)
		}
		binaryPath, _ = filepath.EvalSymlinks(binaryPath)

		if opts.Force || confirmAction(reader, fmt.Sprintf("Remove binary %s?", binaryPath)) {
			if err := os.Remove(binaryPath); err != nil {
				return fmt.Errorf("could not remove binary: %w", err)
			}
			fmt.Printf("Removed %s\n", binaryPath)
		}
	}

	fmt.Println("Uninstall complete.")
	return nil
}

// confirmAction prompts the user and returns true if they answer yes.
func confirmAction(reader *bufio.Reader, prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
