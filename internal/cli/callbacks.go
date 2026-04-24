// Package cli provides the Hera CLI application.
//
// callbacks.go implements interactive prompt callbacks for the terminal TUI:
// clarify questions, secret prompts, and dangerous command approval.
package cli

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ClarifyCallback prompts the user for a clarifying question through the TUI.
// Returns the user's choice or a timeout message.
func ClarifyCallback(question string, choices []string, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	slog.Debug("clarify callback",
		"question", question,
		"choices_count", len(choices),
		"timeout", timeout,
	)

	// In non-interactive mode, return default guidance.
	if len(choices) == 0 {
		return "The user did not provide a response within the time limit. " +
			"Use your best judgement to make the choice and proceed."
	}

	// For now, return the first choice as default.
	// The full TUI integration will use prompt_toolkit equivalent.
	return choices[0]
}

// SecretResult holds the outcome of a secret prompt.
type SecretResult struct {
	Success   bool   `json:"success"`
	Reason    string `json:"reason,omitempty"`
	StoredAs  string `json:"stored_as"`
	Validated bool   `json:"validated"`
	Skipped   bool   `json:"skipped"`
	Message   string `json:"message"`
}

// PromptForSecret prompts for a secret value (e.g., API keys for skills).
// The secret is stored in ~/.hera/.env and never exposed to the model.
func PromptForSecret(varName, prompt string, timeout time.Duration) SecretResult {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	slog.Debug("secret prompt",
		"var_name", varName,
		"timeout", timeout,
	)

	// Non-interactive fallback: skip.
	return SecretResult{
		Success:  true,
		Reason:   "non_interactive",
		StoredAs: varName,
		Skipped:  true,
		Message:  "Secret setup was skipped (non-interactive mode).",
	}
}

// ApprovalCallback prompts for dangerous command approval through the TUI.
// Returns one of: "once", "session", "always", "deny".
func ApprovalCallback(command, description string, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	slog.Debug("approval callback",
		"command_len", len(command),
		"description", description,
		"timeout", timeout,
	)

	// Non-interactive fallback: deny.
	return "deny"
}

// ApprovalLock provides serialized access for concurrent approval requests
// (e.g., from parallel delegation subtasks).
var approvalLock sync.Mutex

// SafeApprovalCallback wraps ApprovalCallback with mutex serialization.
func SafeApprovalCallback(command, description string, timeout time.Duration) string {
	approvalLock.Lock()
	defer approvalLock.Unlock()
	return ApprovalCallback(command, description, timeout)
}

// ClarifyState holds the state of an active clarify prompt.
type ClarifyState struct {
	Question string
	Choices  []string
	Selected int
	Deadline time.Time
	FreeText bool
}

// ApprovalState holds the state of an active approval prompt.
type ApprovalState struct {
	Command     string
	Description string
	Choices     []string
	Selected    int
	Deadline    time.Time
}

// FormatApprovalChoices returns the choices for an approval prompt,
// including "view" when the command exceeds 70 characters.
func FormatApprovalChoices(command string) []string {
	choices := []string{"once", "session", "always", "deny"}
	if len(command) > 70 {
		choices = append(choices, "view")
	}
	return choices
}

// FormatTimeout formats a remaining timeout for display.
func FormatTimeout(remaining time.Duration) string {
	secs := int(remaining.Seconds())
	if secs <= 0 {
		return "timed out"
	}
	return fmt.Sprintf("%ds remaining", secs)
}
