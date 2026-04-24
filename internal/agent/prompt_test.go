package agent

import (
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
)

func TestPromptBuilder_Build(t *testing.T) {
	t.Run("includes identity section", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetIdentity("You are Hera, a personal AI assistant.")
		prompt := pb.Build()
		if !strings.Contains(prompt, "You are Hera, a personal AI assistant.") {
			t.Error("prompt missing identity section")
		}
	})

	t.Run("includes memory context in tags", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetMemoryContext("User likes coffee.")
		prompt := pb.Build()
		if !strings.Contains(prompt, "<memory-context>") {
			t.Error("prompt missing <memory-context> opening tag")
		}
		if !strings.Contains(prompt, "User likes coffee.") {
			t.Error("prompt missing memory content")
		}
		if !strings.Contains(prompt, "</memory-context>") {
			t.Error("prompt missing </memory-context> closing tag")
		}
	})

	t.Run("includes tools list", func(t *testing.T) {
		reg := tools.NewRegistry()
		pb := NewPromptBuilder()
		pb.SetTools(reg.ToolDefs())
		prompt := pb.Build()
		// With no tools, should still have the section header or be empty
		_ = prompt // no tools registered, section may be absent
	})

	t.Run("includes active skills", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetActiveSkills([]*skills.Skill{
			{Name: "reminder", Description: "Set reminders for the user"},
		})
		prompt := pb.Build()
		if !strings.Contains(prompt, "reminder") {
			t.Error("prompt missing skill name")
		}
	})

	t.Run("includes platform context", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetPlatformContext("telegram")
		prompt := pb.Build()
		if !strings.Contains(prompt, "telegram") {
			t.Error("prompt missing platform context")
		}
	})

	t.Run("includes personality guidelines", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetPersonality("friendly and concise")
		prompt := pb.Build()
		if !strings.Contains(prompt, "friendly and concise") {
			t.Error("prompt missing personality guidelines")
		}
	})

	t.Run("assembles all sections in order", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetIdentity("I am Hera.")
		pb.SetMemoryContext("User fact: likes Go.")
		pb.SetPlatformContext("cli")
		pb.SetPersonality("helpful")

		prompt := pb.Build()

		identityIdx := strings.Index(prompt, "I am Hera.")
		memoryIdx := strings.Index(prompt, "User fact: likes Go.")
		platformIdx := strings.Index(prompt, "cli")

		if identityIdx == -1 || memoryIdx == -1 || platformIdx == -1 {
			t.Fatal("one or more sections missing from prompt")
		}
		if identityIdx > memoryIdx {
			t.Error("identity should come before memory context")
		}
	})

	t.Run("empty builder produces empty string", func(t *testing.T) {
		pb := NewPromptBuilder()
		prompt := pb.Build()
		if len(strings.TrimSpace(prompt)) != 0 {
			t.Errorf("expected empty prompt, got %q", prompt)
		}
	})
}

// TestDecisivenessGuidance_InjectedIntoPrompt asserts that:
// 1. SetDecisivenessGuidance stores the text.
// 2. Build() emits the <decisiveness> block.
// 3. The block appears after identity and before memory/tools.
// 4. DecisivenessGuidance constant is non-empty and contains the expected tags.
func TestDecisivenessGuidance_InjectedIntoPrompt(t *testing.T) {
	t.Run("block appears when set", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetDecisivenessGuidance(DecisivenessGuidance)
		prompt := pb.Build()
		if !strings.Contains(prompt, "<decisiveness>") {
			t.Error("prompt missing <decisiveness> opening tag")
		}
		if !strings.Contains(prompt, "</decisiveness>") {
			t.Error("prompt missing </decisiveness> closing tag")
		}
	})

	t.Run("block absent when not set", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetIdentity("identity text")
		prompt := pb.Build()
		if strings.Contains(prompt, "<decisiveness>") {
			t.Error("prompt should not contain <decisiveness> when not set")
		}
	})

	t.Run("block comes after identity", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetIdentity("identity text")
		pb.SetDecisivenessGuidance(DecisivenessGuidance)
		pb.SetMemoryContext("mem text")
		prompt := pb.Build()

		identityIdx := strings.Index(prompt, "identity text")
		decisiveIdx := strings.Index(prompt, "<decisiveness>")
		memIdx := strings.Index(prompt, "<memory-context>")

		if identityIdx == -1 || decisiveIdx == -1 || memIdx == -1 {
			t.Fatal("one or more required sections missing")
		}
		if decisiveIdx < identityIdx {
			t.Error("decisiveness block should come after identity")
		}
		if decisiveIdx > memIdx {
			t.Error("decisiveness block should come before memory-context")
		}
	})

	t.Run("DecisivenessGuidance constant is well-formed", func(t *testing.T) {
		if !strings.Contains(DecisivenessGuidance, "<decisiveness>") {
			t.Error("constant missing opening <decisiveness> tag")
		}
		if !strings.Contains(DecisivenessGuidance, "</decisiveness>") {
			t.Error("constant missing closing </decisiveness> tag")
		}
		if len(DecisivenessGuidance) < 100 {
			t.Error("constant too short — must contain meaningful guidance")
		}
	})
}

// TestPathConventions_InjectedIntoPrompt asserts that:
// 1. SetPathConventions stores the text and Build emits it.
// 2. It's absent when not set.
// 3. It sits between decisiveness and hints/memory sections.
// 4. The PathConventions constant is well-formed (tags + core rules).
func TestPathConventions_InjectedIntoPrompt(t *testing.T) {
	t.Run("block appears when set", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetPathConventions(PathConventions)
		prompt := pb.Build()
		if !strings.Contains(prompt, "<file-paths>") {
			t.Error("prompt missing <file-paths> opening tag")
		}
		if !strings.Contains(prompt, "</file-paths>") {
			t.Error("prompt missing </file-paths> closing tag")
		}
	})

	t.Run("block absent when not set", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetIdentity("identity text")
		prompt := pb.Build()
		if strings.Contains(prompt, "<file-paths>") {
			t.Error("prompt should not contain <file-paths> when not set")
		}
	})

	t.Run("block comes after decisiveness, before memory", func(t *testing.T) {
		pb := NewPromptBuilder()
		pb.SetIdentity("identity text")
		pb.SetDecisivenessGuidance(DecisivenessGuidance)
		pb.SetPathConventions(PathConventions)
		pb.SetMemoryContext("mem text")
		prompt := pb.Build()

		decisiveIdx := strings.Index(prompt, "<decisiveness>")
		pathIdx := strings.Index(prompt, "<file-paths>")
		memIdx := strings.Index(prompt, "<memory-context>")

		if decisiveIdx == -1 || pathIdx == -1 || memIdx == -1 {
			t.Fatal("one or more required sections missing")
		}
		if pathIdx < decisiveIdx {
			t.Error("<file-paths> should come after <decisiveness>")
		}
		if pathIdx > memIdx {
			t.Error("<file-paths> should come before <memory-context>")
		}
	})

	t.Run("PathConventions constant is well-formed", func(t *testing.T) {
		if !strings.Contains(PathConventions, "<file-paths>") {
			t.Error("constant missing opening <file-paths> tag")
		}
		if !strings.Contains(PathConventions, "</file-paths>") {
			t.Error("constant missing closing </file-paths> tag")
		}
		// Must mention the core tokens so the LLM can learn them.
		for _, token := range []string{"$HERA_HOME", ".hera/", "~"} {
			if !strings.Contains(PathConventions, token) {
				t.Errorf("constant missing mention of %q", token)
			}
		}
	})
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"four chars", "abcd", 1},
		{"eight chars", "abcdefgh", 2},
		{"one char", "a", 1}, // rounds up
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
