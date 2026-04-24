package cli

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const systemdServiceTemplate = `[Unit]
Description=Hera AI Agent Gateway
After=network.target

[Service]
Type=simple
User={{USER}}
ExecStart={{BINARY}} gateway start --config {{CONFIG}}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
Environment=HOME={{HOME}}

[Install]
WantedBy=multi-user.target
`

// GenerateSystemdService creates a systemd service file content.
func GenerateSystemdService(binaryPath, configPath, userName string) string {
	u, _ := user.Current()
	if userName == "" && u != nil {
		userName = u.Username
	}
	homeDir := ""
	if u != nil {
		homeDir = u.HomeDir
	}

	svc := systemdServiceTemplate
	svc = strings.ReplaceAll(svc, "{{USER}}", userName)
	svc = strings.ReplaceAll(svc, "{{BINARY}}", binaryPath)
	svc = strings.ReplaceAll(svc, "{{CONFIG}}", configPath)
	svc = strings.ReplaceAll(svc, "{{HOME}}", homeDir)

	return svc
}

// InstallSystemdService generates and writes a systemd service file.
// It tries user-level systemd first (~/.config/systemd/user/), falling
// back to instructions for system-level (/etc/systemd/system/).
func InstallSystemdService(binaryPath, configPath string) error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	// Make paths absolute.
	if !filepath.IsAbs(binaryPath) {
		abs, err := filepath.Abs(binaryPath)
		if err == nil {
			binaryPath = abs
		}
	}
	if !filepath.IsAbs(configPath) {
		abs, err := filepath.Abs(configPath)
		if err == nil {
			configPath = abs
		}
	}

	serviceContent := GenerateSystemdService(binaryPath, configPath, u.Username)

	// Try user-level systemd directory.
	userServiceDir := filepath.Join(u.HomeDir, ".config", "systemd", "user")
	if err := os.MkdirAll(userServiceDir, 0o755); err == nil {
		servicePath := filepath.Join(userServiceDir, "hera.service")
		if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err == nil {
			fmt.Printf("Service file written to: %s\n\n", servicePath)
			fmt.Println("To enable and start:")
			fmt.Println("  systemctl --user daemon-reload")
			fmt.Println("  systemctl --user enable hera")
			fmt.Println("  systemctl --user start hera")
			fmt.Println()
			fmt.Println("To check status:")
			fmt.Println("  systemctl --user status hera")
			fmt.Println()
			fmt.Println("To view logs:")
			fmt.Println("  journalctl --user -u hera -f")
			return nil
		}
	}

	// Fall back: print instructions for system-level install.
	fmt.Println("Could not write user-level service. For system-level install:")
	fmt.Printf("\nWrite the following to /etc/systemd/system/hera.service:\n\n")
	fmt.Println(serviceContent)
	fmt.Println("Then run:")
	fmt.Println("  sudo systemctl daemon-reload")
	fmt.Println("  sudo systemctl enable hera")
	fmt.Println("  sudo systemctl start hera")

	return nil
}
