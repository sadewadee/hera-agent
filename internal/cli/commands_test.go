package cli

import (
	"strings"
	"testing"
)

func TestParseSlashCommand_ValidCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantArgs string
	}{
		{"/help", "/help", ""},
		{"/help tools", "/help", "tools"},
		{"/model gpt-4o", "/model", "gpt-4o"},
		{"/quit", "/quit", ""},
		{"/export json ~/output.json", "/export", "json ~/output.json"},
		{"  /help  ", "/help", ""},
		{"/HELP", "/help", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := ParseSlashCommand(tt.input)
			if cmd == nil {
				t.Fatalf("ParseSlashCommand(%q) = nil, want command", tt.input)
			}
			if cmd.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cmd.Name, tt.wantName)
			}
			if cmd.Args != tt.wantArgs {
				t.Errorf("Args = %q, want %q", cmd.Args, tt.wantArgs)
			}
		})
	}
}

func TestParseSlashCommand_NotACommand(t *testing.T) {
	tests := []string{
		"hello",
		"not a command",
		"",
		"  ",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			cmd := ParseSlashCommand(input)
			if cmd != nil {
				t.Errorf("ParseSlashCommand(%q) = %v, want nil", input, cmd)
			}
		})
	}
}

func TestSlashCommandRegistry_RegisterAndGet(t *testing.T) {
	r := NewSlashCommandRegistry()

	// Test that built-in commands are registered
	tests := []struct {
		name    string
		aliases []string
	}{
		{"/help", []string{"/h", "/?"}},
		{"/new", []string{"/n"}},
		{"/model", []string{"/m"}},
		{"/quit", []string{"/q", "/exit"}},
		{"/version", []string{"/v"}},
		{"/clear", []string{"/cls"}},
		{"/personality", []string{"/persona"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := r.Get(tt.name)
			if !ok {
				t.Fatalf("Get(%q) not found", tt.name)
			}
			if cmd.Name != tt.name {
				t.Errorf("Name = %q, want %q", cmd.Name, tt.name)
			}

			// Test aliases
			for _, alias := range tt.aliases {
				aliasCmd, ok := r.Get(alias)
				if !ok {
					t.Errorf("Get(%q) alias not found", alias)
					continue
				}
				if aliasCmd.Name != tt.name {
					t.Errorf("alias %q points to %q, want %q", alias, aliasCmd.Name, tt.name)
				}
			}
		})
	}
}

func TestSlashCommandRegistry_All(t *testing.T) {
	r := NewSlashCommandRegistry()
	all := r.All()
	if len(all) == 0 {
		t.Error("All() returned empty list")
	}

	// Check that we have at least the core commands
	expectedCount := 16 // /help, /new, /history, /model, /provider, /personality, /skin, /tools, /skills, /clear, /compress, /usage, /version, /doctor, /export, /quit
	if len(all) < expectedCount {
		t.Errorf("All() returned %d commands, want at least %d", len(all), expectedCount)
	}
}

// TestVersion asserts the version constant has the expected format and value.
func TestVersion(t *testing.T) {
	if version != "0.0.142" {
		t.Errorf("version = %q, want %q", version, "0.0.142")
	}
	// Must be three numeric dot-separated parts.
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		t.Errorf("version %q does not have three dot-separated parts", version)
	}
}

// TestStubCommandsReturnErrors asserts that commands which require wired agent
// context return errors (not misleading "not yet implemented" strings) when
// called on the bare registry without wireSlashCommands.
func TestStubCommandsReturnErrors(t *testing.T) {
	r := NewSlashCommandRegistry()

	stubs := []struct {
		name string
		args string
	}{
		{"/history", ""},
		{"/model", ""},
		{"/provider", ""},
		{"/tools", ""},
		{"/skills", ""},
		{"/compress", ""},
		{"/usage", ""},
		{"/export", ""},
	}

	for _, tt := range stubs {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := r.Get(tt.name)
			if !ok {
				t.Fatalf("command %q not registered", tt.name)
			}
			_, err := cmd.Handler(tt.args)
			if err == nil {
				t.Errorf("%s handler: want error when called without wired context, got nil", tt.name)
			}
			if strings.Contains(err.Error(), "not yet implemented") {
				t.Errorf("%s handler error contains misleading 'not yet implemented': %q", tt.name, err.Error())
			}
		})
	}
}

func TestSlashCommandRegistry_HelpHandler(t *testing.T) {
	r := NewSlashCommandRegistry()
	cmd, ok := r.Get("/help")
	if !ok {
		t.Fatal("Get('/help') not found")
	}

	// Test help with no args (list all)
	output, err := cmd.Handler("")
	if err != nil {
		t.Fatalf("Handler('') error = %v", err)
	}
	if output == "" {
		t.Error("Handler('') returned empty output")
	}

	// Test help with a specific command
	output, err = cmd.Handler("model")
	if err != nil {
		t.Fatalf("Handler('model') error = %v", err)
	}
	if output == "" {
		t.Error("Handler('model') returned empty output")
	}

	// Test help with unknown command
	output, err = cmd.Handler("nonexistent")
	if err != nil {
		t.Fatalf("Handler('nonexistent') error = %v", err)
	}
	if output == "" {
		t.Error("Handler('nonexistent') returned empty output")
	}
}
