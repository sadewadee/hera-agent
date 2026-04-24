package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
)

// buildTestApp creates a minimal App wired with wireSlashCommands.
func buildTestApp(t *testing.T) (*App, *SlashCommandRegistry) {
	t.Helper()
	cfg := &config.Config{}
	cfg.Agent.DefaultProvider = "openai"
	cfg.Agent.DefaultModel = "gpt-4o"
	cfg.Memory.Provider = "sqlite"
	cfg.Memory.DBPath = filepath.Join(t.TempDir(), "test.db")

	deps := AppDeps{Config: cfg}
	app := &App{deps: deps}
	app.skin = NewSkinEngine()

	// Create a real session manager so /import can create sessions.
	sm := agent.NewSessionManager(0)
	deps.Sessions = sm
	app.deps = deps

	reg := NewSlashCommandRegistry()
	app.wireSlashCommands(reg, context.Background())
	return app, reg
}

func TestWired_Context_NoSession(t *testing.T) {
	app, reg := buildTestApp(t)
	_ = app

	cmd, ok := reg.Get("/context")
	if !ok {
		t.Fatal("/context not registered")
	}
	out, err := cmd.Handler("")
	if err != nil {
		t.Fatalf("/context error: %v", err)
	}
	if !strings.Contains(out, "provider:") {
		t.Errorf("/context output missing 'provider:', got: %s", out)
	}
	if !strings.Contains(out, "openai") {
		t.Errorf("/context output missing provider value, got: %s", out)
	}
	if !strings.Contains(out, "session_id:") {
		t.Errorf("/context output missing 'session_id:', got: %s", out)
	}
}

func TestWired_Context_WithSession(t *testing.T) {
	app, reg := buildTestApp(t)
	sm := app.deps.Sessions
	sess := sm.Create("cli", "local")
	sess.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "hello"})
	app.currentSession = sess

	cmd, _ := reg.Get("/context")
	out, err := cmd.Handler("")
	if err != nil {
		t.Fatalf("/context error: %v", err)
	}
	if !strings.Contains(out, sess.ID) {
		t.Errorf("/context output missing session ID %q, got: %s", sess.ID, out)
	}
	if !strings.Contains(out, "messages:") {
		t.Errorf("/context output missing 'messages:', got: %s", out)
	}
}

func TestWired_Settings(t *testing.T) {
	_, reg := buildTestApp(t)
	cmd, ok := reg.Get("/settings")
	if !ok {
		t.Fatal("/settings not registered")
	}
	out, err := cmd.Handler("")
	if err != nil {
		t.Fatalf("/settings error: %v", err)
	}
	if !strings.Contains(out, "agent.default_provider:") {
		t.Errorf("/settings output missing provider field, got: %s", out)
	}
	if !strings.Contains(out, "openai") {
		t.Errorf("/settings output missing 'openai', got: %s", out)
	}
	if !strings.Contains(out, "memory.provider:") {
		t.Errorf("/settings output missing memory field, got: %s", out)
	}
	if !strings.Contains(out, "Config directory:") {
		t.Errorf("/settings output missing config directory, got: %s", out)
	}
}

func TestWired_Import_ValidJSON(t *testing.T) {
	app, reg := buildTestApp(t)

	// Create a valid transcript JSON file.
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
		{Role: llm.RoleAssistant, Content: "hi there"},
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "transcript.json")
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, ok := reg.Get("/import")
	if !ok {
		t.Fatal("/import not registered")
	}
	out, err := cmd.Handler(f)
	if err != nil {
		t.Fatalf("/import error: %v", err)
	}
	if !strings.Contains(out, "2 messages") {
		t.Errorf("/import output = %q, want '2 messages'", out)
	}
	// Verify session was set on the app.
	if app.currentSession == nil {
		t.Error("/import should set app.currentSession")
	}
	if len(app.currentSession.GetMessages()) != 2 {
		t.Errorf("session has %d messages, want 2", len(app.currentSession.GetMessages()))
	}
}

func TestWired_Import_NoArgs(t *testing.T) {
	_, reg := buildTestApp(t)
	cmd, _ := reg.Get("/import")
	out, err := cmd.Handler("")
	if err != nil {
		t.Fatalf("/import error: %v", err)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("/import no-args output = %q, want 'Usage:'", out)
	}
}

func TestWired_Import_InvalidJSON(t *testing.T) {
	_, reg := buildTestApp(t)
	f := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(f, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, _ := reg.Get("/import")
	_, err := cmd.Handler(f)
	if err == nil {
		t.Fatal("/import should error on invalid JSON")
	}
}

func TestWired_Import_MissingFile(t *testing.T) {
	_, reg := buildTestApp(t)
	cmd, _ := reg.Get("/import")
	_, err := cmd.Handler("/no/such/file.json")
	if err == nil {
		t.Fatal("/import should error for missing file")
	}
}
