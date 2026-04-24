package cli

import (
	"strings"
	"testing"
)

func TestHandleAuthCommand_EmptyArgs(t *testing.T) {
	result, err := HandleAuthCommand("")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Usage") {
		t.Errorf("expected usage message, got %q", result)
	}
}

func TestHandleAuthCommand_Status(t *testing.T) {
	result, err := HandleAuthCommand("status")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Auth status") {
		t.Errorf("expected auth status info, got %q", result)
	}
	if !strings.Contains(result, "JWT") || !strings.Contains(result, "HMAC") {
		t.Errorf("expected JWT signing info in status, got %q", result)
	}
}

func TestHandleAuthCommand_TokenShow(t *testing.T) {
	result, err := HandleAuthCommand("token show")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "security") {
		t.Errorf("expected security message for token show, got %q", result)
	}
}

func TestHandleAuthCommand_TokenGenerate(t *testing.T) {
	result, err := HandleAuthCommand("token generate")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Generated") {
		t.Errorf("expected generation message, got %q", result)
	}
}

func TestHandleAuthCommand_TokenMissingSubcommand(t *testing.T) {
	result, err := HandleAuthCommand("token")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Usage") {
		t.Errorf("expected usage message for 'token' without subcommand, got %q", result)
	}
}

func TestHandleAuthCommand_TokenUnknownSubcommand(t *testing.T) {
	result, err := HandleAuthCommand("token foobar")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Usage") {
		t.Errorf("expected usage message for unknown token subcommand, got %q", result)
	}
}

func TestHandleAuthCommand_Rotate(t *testing.T) {
	result, err := HandleAuthCommand("rotate")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "rotation") {
		t.Errorf("expected rotation message, got %q", result)
	}
	if !strings.Contains(result, "grace period") {
		t.Errorf("expected grace period mention, got %q", result)
	}
}

func TestHandleAuthCommand_Revoke(t *testing.T) {
	result, err := HandleAuthCommand("revoke abc123")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "abc123") {
		t.Errorf("expected prefix in revoke message, got %q", result)
	}
	if !strings.Contains(result, "Revoked") {
		t.Errorf("expected revoked message, got %q", result)
	}
}

func TestHandleAuthCommand_RevokeMissingPrefix(t *testing.T) {
	result, err := HandleAuthCommand("revoke")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Usage") {
		t.Errorf("expected usage message for revoke without prefix, got %q", result)
	}
}

func TestHandleAuthCommand_UnknownSubcommand(t *testing.T) {
	result, err := HandleAuthCommand("unknown")
	if err != nil {
		t.Fatalf("HandleAuthCommand error = %v", err)
	}
	if !strings.Contains(result, "Usage") {
		t.Errorf("expected usage message for unknown subcommand, got %q", result)
	}
}
