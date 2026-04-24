package agent

import (
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/skills"
)

// DecisivenessGuidance is injected into every prompt regardless of personality.
// It anchors the action-bias principles defined in configs/SOUL.md so they survive
// personality overrides that might add "ask for clarification first" phrasing.
const DecisivenessGuidance = `<decisiveness>
When you receive a request:
1. If there is an obvious default interpretation, act on it immediately.
2. If required info is retrievable via a tool (file read, shell command, memory lookup), call the tool — do not ask.
3. Ask ONE specific closed-ended question only when: (a) the info cannot be retrieved by any tool, AND (b) the ambiguity changes which tool you would call next.
4. Keep going until the task resolves. Don't stop with a plan — execute it. Label any assumption explicitly so the user can redirect.
5. Act when ≥70% confident. Ask only when <70% AND the action is high-cost or irreversible (file deletion, git force-push, external API write, paid API call). Reads and reversible edits: just do them.
</decisiveness>`

// PathConventions describes how file-I/O tools interpret path arguments.
// Injected alongside DecisivenessGuidance so the LLM can predict where its
// writes will land. Backed by paths.Normalize in internal/paths.
const PathConventions = `<file-paths>
When you pass a path to file_read/file_write/csv/pdf_reader/archive/database/neutts/patch/binary_ext tools:
1. Absolute path (e.g. /tmp/foo, C:\Users\me\foo): used as-is.
2. "~" or "~/foo": expands to the user's $HOME directory.
3. "$HERA_HOME/foo" or "${HERA_HOME}/foo": resolves to the Hera home directory (default ~/.hera, overridable via HERA_HOME env).
4. ".hera/foo" (as a leading segment): automatically redirected to HERA_HOME/foo. This is a safety net — the log shows the redirect. Prefer "$HERA_HOME/..." to make intent explicit.
5. Bare relative path (e.g. "foo.md", "subdir/bar"): resolved against the current working directory.

For agent-generated artifacts that should survive across runs, prefer "$HERA_HOME/..." paths so they land in the user's Hera home, not in whatever CWD the user happens to launch from.
</file-paths>`

// PromptBuilder assembles the system prompt from multiple sections.
type PromptBuilder struct {
	identity          string
	decisivenessBlock string
	pathConventions   string
	memoryContext     string
	hints             string
	tools             []llm.ToolDef
	activeSkills      []*skills.Skill
	platform          string
	personality       string
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// SetIdentity sets the identity section (from SOUL.md).
func (pb *PromptBuilder) SetIdentity(identity string) {
	pb.identity = identity
}

// SetDecisivenessGuidance sets the decisiveness block. Call with the
// DecisivenessGuidance constant from all entry-point binaries.
// Independent of personality — always injected.
func (pb *PromptBuilder) SetDecisivenessGuidance(text string) {
	pb.decisivenessBlock = text
}

// SetPathConventions sets the file-path conventions block. Call with the
// PathConventions constant from every entrypoint — independent of personality.
func (pb *PromptBuilder) SetPathConventions(text string) {
	pb.pathConventions = text
}

// SetMemoryContext sets the memory context section.
func (pb *PromptBuilder) SetMemoryContext(ctx string) {
	pb.memoryContext = ctx
}

// SetTools sets the available tools for the prompt.
func (pb *PromptBuilder) SetTools(defs []llm.ToolDef) {
	pb.tools = defs
}

// SetActiveSkills sets the currently active skills.
func (pb *PromptBuilder) SetActiveSkills(s []*skills.Skill) {
	pb.activeSkills = s
}

// SetPlatformContext sets the platform context (e.g., "telegram", "discord").
func (pb *PromptBuilder) SetPlatformContext(platform string) {
	pb.platform = platform
}

// SetPersonality sets the personality/conversation guidelines.
func (pb *PromptBuilder) SetPersonality(personality string) {
	pb.personality = personality
}

// AddHints appends subdirectory hints (from .hera.md, AGENTS.md, etc.) to the prompt.
func (pb *PromptBuilder) AddHints(hints string) {
	pb.hints = hints
}

// Build assembles the system prompt from all configured sections.
// Priority order: identity → decisiveness → path conventions → hints → memory → tools → skills → platform → personality.
func (pb *PromptBuilder) Build() string {
	var sections []string

	// 1. Identity
	if pb.identity != "" {
		sections = append(sections, pb.identity)
	}

	// 2. Decisiveness (action-bias anchor — independent of personality)
	if pb.decisivenessBlock != "" {
		sections = append(sections, pb.decisivenessBlock)
	}

	// 3. Path conventions (file-I/O tools: ~, $HERA_HOME, .hera/… semantics)
	if pb.pathConventions != "" {
		sections = append(sections, pb.pathConventions)
	}

	// 4. Subdirectory hints (wired from hints.go)
	if pb.hints != "" {
		sections = append(sections, fmt.Sprintf("<hints>\n%s\n</hints>", pb.hints))
	}

	// 4. Memory context
	if pb.memoryContext != "" {
		sections = append(sections, fmt.Sprintf("<memory-context>\n%s\n</memory-context>", pb.memoryContext))
	}

	// 5. Available tools
	if len(pb.tools) > 0 {
		var toolLines []string
		toolLines = append(toolLines, "## Available Tools")
		for _, t := range pb.tools {
			toolLines = append(toolLines, fmt.Sprintf("- **%s**: %s", t.Name, t.Description))
		}
		sections = append(sections, strings.Join(toolLines, "\n"))
	}

	// 6. Active skills
	if len(pb.activeSkills) > 0 {
		var skillLines []string
		skillLines = append(skillLines, "## Active Skills")
		for _, s := range pb.activeSkills {
			skillLines = append(skillLines, fmt.Sprintf("- **%s**: %s", s.Name, s.Description))
		}
		sections = append(sections, strings.Join(skillLines, "\n"))
	}

	// 7. Platform context
	if pb.platform != "" {
		sections = append(sections, fmt.Sprintf("## Platform\nYou are currently operating on: %s", pb.platform))
	}

	// 8. Personality/conversation guidelines
	if pb.personality != "" {
		sections = append(sections, fmt.Sprintf("## Conversation Style\n%s", pb.personality))
	}

	return strings.Join(sections, "\n\n")
}

// EstimateTokens approximates the token count for a string.
// Uses the heuristic: 4 characters ~ 1 token, with a minimum of 1 for non-empty strings.
func EstimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	tokens := len(s) / 4
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}

// EstimateTokensForMessages estimates the total token count for a slice of messages.
func EstimateTokensForMessages(messages []llm.Message) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content)
	}
	return total
}
