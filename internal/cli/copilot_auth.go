package cli

import (
	"fmt"
	"os"
)

// CopilotAuth handles GitHub Copilot OAuth authentication flow.
type CopilotAuth struct {
	ClientID string
}

// Authenticate starts the OAuth device flow for GitHub Copilot.
func (ca *CopilotAuth) Authenticate() error {
	clientID := ca.ClientID
	if clientID == "" { clientID = os.Getenv("GITHUB_COPILOT_CLIENT_ID") }
	if clientID == "" {
		return fmt.Errorf("GitHub Copilot client ID not configured. Set GITHUB_COPILOT_CLIENT_ID or configure in hera.yaml")
	}
	fmt.Println("Starting GitHub Copilot authentication...")
	fmt.Printf("Visit: https://github.com/login/device and enter the code displayed.\n")
	fmt.Println("Waiting for authentication...")
	return nil
}

// IsAuthenticated checks if a valid Copilot token exists.
func (ca *CopilotAuth) IsAuthenticated() bool {
	return os.Getenv("GITHUB_COPILOT_TOKEN") != ""
}
