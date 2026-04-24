package cli

import (
	"testing"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"ok", "[PASS]"},
		{"warn", "[WARN]"},
		{"fail", "[FAIL]"},
		{"unknown", "[????]"},
		{"", "[????]"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusIcon(tt.status)
			if got != tt.want {
				t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestCheckGoVersion(t *testing.T) {
	result := checkGoVersion()
	if result.Status != "ok" {
		t.Errorf("checkGoVersion status = %q, want %q", result.Status, "ok")
	}
	if result.Name != "Go Runtime" {
		t.Errorf("checkGoVersion name = %q, want %q", result.Name, "Go Runtime")
	}
	if result.Message == "" {
		t.Error("checkGoVersion message is empty")
	}
}

func TestCheckPlatform(t *testing.T) {
	result := checkPlatform()
	if result.Status != "ok" {
		t.Errorf("checkPlatform status = %q, want %q", result.Status, "ok")
	}
	if result.Name != "Platform" {
		t.Errorf("checkPlatform name = %q, want %q", result.Name, "Platform")
	}
	if result.Message == "" {
		t.Error("checkPlatform message is empty")
	}
}

func TestCheckResult_Fields(t *testing.T) {
	cr := checkResult{
		Name:    "Test Check",
		Status:  "ok",
		Message: "everything is fine",
	}
	if cr.Name != "Test Check" {
		t.Errorf("Name = %q, want %q", cr.Name, "Test Check")
	}
	if cr.Status != "ok" {
		t.Errorf("Status = %q, want %q", cr.Status, "ok")
	}
	if cr.Message != "everything is fine" {
		t.Errorf("Message = %q, want %q", cr.Message, "everything is fine")
	}
}
