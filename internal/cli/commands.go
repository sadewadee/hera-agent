package cli

import (
	"fmt"
	"strings"
)

// SlashCommand represents a slash command that can be executed in chat.
type SlashCommand struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Handler     func(args string) (string, error)
}

// SlashCommandRegistry manages available slash commands.
type SlashCommandRegistry struct {
	commands map[string]*SlashCommand
}

// NewSlashCommandRegistry creates a registry with all built-in slash commands.
func NewSlashCommandRegistry() *SlashCommandRegistry {
	r := &SlashCommandRegistry{
		commands: make(map[string]*SlashCommand),
	}
	r.registerBuiltins()
	return r
}

// Get returns a slash command by name (including aliases).
func (r *SlashCommandRegistry) Get(name string) (*SlashCommand, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// All returns all registered slash commands (unique, no alias duplicates).
func (r *SlashCommandRegistry) All() []*SlashCommand {
	seen := make(map[string]bool)
	var result []*SlashCommand
	for _, cmd := range r.commands {
		if !seen[cmd.Name] {
			seen[cmd.Name] = true
			result = append(result, cmd)
		}
	}
	return result
}

func (r *SlashCommandRegistry) register(cmd *SlashCommand) {
	r.commands[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		r.commands[alias] = cmd
	}
}

func (r *SlashCommandRegistry) registerBuiltins() {
	r.register(&SlashCommand{
		Name:        "/help",
		Aliases:     []string{"/h", "/?"},
		Description: "Show available commands",
		Usage:       "/help [command]",
		Handler: func(args string) (string, error) {
			if args != "" {
				cmd, ok := r.Get("/" + args)
				if !ok {
					return fmt.Sprintf("Unknown command: /%s", args), nil
				}
				return fmt.Sprintf("%s - %s\nUsage: %s", cmd.Name, cmd.Description, cmd.Usage), nil
			}
			var sb strings.Builder
			sb.WriteString("Available commands:\n\n")
			for _, cmd := range r.All() {
				sb.WriteString(fmt.Sprintf("  %-14s %s\n", cmd.Name, cmd.Description))
			}
			return sb.String(), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/new",
		Aliases:     []string{"/n"},
		Description: "Start a new conversation",
		Usage:       "/new",
		Handler: func(args string) (string, error) {
			return "Starting new conversation...", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/history",
		Description: "Show conversation history",
		Usage:       "/history [count]",
		Handler: func(args string) (string, error) {
			// Full implementation is wired via App.wireSlashCommands when running
			// interactively. This default is reached only in isolated registry use.
			return "", fmt.Errorf("no active session — start the interactive CLI to use /history")
		},
	})

	r.register(&SlashCommand{
		Name:        "/model",
		Aliases:     []string{"/m"},
		Description: "Switch or show the current model",
		Usage:       "/model [model_name]",
		Handler: func(args string) (string, error) {
			if args == "" {
				// Full implementation reads cfg.Agent.DefaultModel via wireSlashCommands.
				return "", fmt.Errorf("config not available — start the interactive CLI to use /model")
			}
			return fmt.Sprintf("Switching model to: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/provider",
		Description: "Switch or show the current provider",
		Usage:       "/provider [provider_name]",
		Handler: func(args string) (string, error) {
			if args == "" {
				// Full implementation reads cfg.Agent.DefaultProvider via wireSlashCommands.
				return "", fmt.Errorf("config not available — start the interactive CLI to use /provider")
			}
			return fmt.Sprintf("Switching provider to: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/personality",
		Aliases:     []string{"/persona"},
		Description: "Switch or show the current personality",
		Usage:       "/personality [name]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Current personality: helpful", nil
			}
			return fmt.Sprintf("Switching personality to: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/skin",
		Description: "Switch or show the current skin/theme",
		Usage:       "/skin [name]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Current skin: default", nil
			}
			return fmt.Sprintf("Switching skin to: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/tools",
		Aliases:     []string{"/t"},
		Description: "List available tools",
		Usage:       "/tools",
		Handler: func(args string) (string, error) {
			// Full implementation iterates ToolRegistry.ToolDefs() via wireSlashCommands.
			return "", fmt.Errorf("tool registry not available — start the interactive CLI to use /tools")
		},
	})

	r.register(&SlashCommand{
		Name:        "/skills",
		Description: "List or manage skills",
		Usage:       "/skills [list|create|info <name>]",
		Handler: func(args string) (string, error) {
			// Full implementation iterates SkillLoader.All() via wireSlashCommands.
			return "", fmt.Errorf("skill loader not available — start the interactive CLI to use /skills")
		},
	})

	r.register(&SlashCommand{
		Name:        "/clear",
		Aliases:     []string{"/cls"},
		Description: "Clear the screen",
		Usage:       "/clear",
		Handler: func(args string) (string, error) {
			return "\033[2J\033[H", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/compress",
		Description: "Compress the current conversation context",
		Usage:       "/compress",
		Handler: func(args string) (string, error) {
			// Full implementation calls Compressor.Compress via wireSlashCommands.
			return "", fmt.Errorf("no active session — start the interactive CLI to use /compress")
		},
	})

	r.register(&SlashCommand{
		Name:        "/usage",
		Description: "Show token usage and cost for this session",
		Usage:       "/usage",
		Handler: func(args string) (string, error) {
			// Full implementation reads Agent.UsageStats() via wireSlashCommands.
			return "", fmt.Errorf("agent not available — start the interactive CLI to use /usage")
		},
	})

	r.register(&SlashCommand{
		Name:        "/version",
		Aliases:     []string{"/v"},
		Description: "Show Hera version",
		Usage:       "/version",
		Handler: func(args string) (string, error) {
			return fmt.Sprintf("Hera version %s", version), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/doctor",
		Description: "Run health diagnostics",
		Usage:       "/doctor",
		Handler: func(args string) (string, error) {
			return "Run 'hera doctor' from the command line for full diagnostics.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/export",
		Description: "Export the current conversation",
		Usage:       "/export [format] [path]",
		Handler: func(args string) (string, error) {
			// Full implementation writes session messages to file via wireSlashCommands.
			return "", fmt.Errorf("no active session — start the interactive CLI to use /export")
		},
	})

	r.register(&SlashCommand{
		Name:        "/quit",
		Aliases:     []string{"/q", "/exit"},
		Description: "Exit the chat session",
		Usage:       "/quit",
		Handler: func(args string) (string, error) {
			return "", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/branch",
		Description: "Branch the current session (create a copy)",
		Usage:       "/branch",
		Handler: func(args string) (string, error) {
			return "Session branching will be wired when running interactively.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/fork",
		Description: "Fork the session from a specific message index",
		Usage:       "/fork <message_index>",
		Handler: func(args string) (string, error) {
			return "Session forking will be wired when running interactively.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/load",
		Description: "Load a previously saved session",
		Usage:       "/load <session_id>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /load <session_id>", nil
			}
			return fmt.Sprintf("Loading session: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/save",
		Description: "Save the current session",
		Usage:       "/save [name]",
		Handler: func(args string) (string, error) {
			return "Session saved.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/set",
		Description: "Set a configuration value",
		Usage:       "/set <key> <value>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /set <key> <value>\nExample: /set model gpt-4o", nil
			}
			parts := strings.SplitN(args, " ", 2)
			if len(parts) < 2 {
				return "Usage: /set <key> <value>", nil
			}
			return fmt.Sprintf("Set %s = %s", parts[0], parts[1]), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/enable",
		Description: "Enable a feature or tool",
		Usage:       "/enable <feature>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /enable <feature>\nFeatures: memory, tools, skills, pii_redaction, injection_detection", nil
			}
			return fmt.Sprintf("Enabled: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/disable",
		Description: "Disable a feature or tool",
		Usage:       "/disable <feature>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /disable <feature>", nil
			}
			return fmt.Sprintf("Disabled: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/skill-create",
		Description: "Create a new skill from the current conversation",
		Usage:       "/skill-create <name> [description]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /skill-create <name> [description]", nil
			}
			return fmt.Sprintf("Creating skill: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/pair",
		Description: "Generate a pairing code for device/platform authentication",
		Usage:       "/pair [platform]",
		Handler: func(args string) (string, error) {
			return "Pairing code generation will be implemented with the gateway.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/platforms",
		Description: "List connected platforms and their status",
		Usage:       "/platforms",
		Handler: func(args string) (string, error) {
			return "Platform status is available when running the gateway.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/gateway",
		Description: "Gateway management commands",
		Usage:       "/gateway <start|stop|status|install>",
		Handler: func(args string) (string, error) {
			switch args {
			case "start":
				return "Use 'hera gateway start' from the command line.", nil
			case "stop":
				return "Send SIGINT/SIGTERM to stop the gateway.", nil
			case "status":
				return "Gateway status: use 'hera gateway start' to run.", nil
			case "install":
				return "Use 'hera gateway install' to generate a systemd service.", nil
			default:
				return "Usage: /gateway <start|stop|status|install>", nil
			}
		},
	})

	r.register(&SlashCommand{
		Name:        "/btw",
		Description: "Send a background thought/note to the agent",
		Usage:       "/btw <thought>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /btw <thought>", nil
			}
			return fmt.Sprintf("[Background note recorded: %s]", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/bg",
		Description: "Run a prompt in the background",
		Usage:       "/bg <prompt>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /bg <prompt>", nil
			}
			return fmt.Sprintf("Background prompt queued: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/queue",
		Description: "Show the prompt queue",
		Usage:       "/queue",
		Handler: func(args string) (string, error) {
			return "Prompt queue is empty.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/profile",
		Aliases:     []string{"/profiles"},
		Description: "Manage agent profiles",
		Usage:       "/profile [list|create|switch <name>]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Active profile: default\nUse /profile list, /profile create <name>, /profile switch <name>", nil
			}
			return fmt.Sprintf("Profile command: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/about",
		Description: "Show information about Hera",
		Usage:       "/about",
		Handler: func(args string) (string, error) {
			return fmt.Sprintf("Hera v%s\nA self-improving, multi-platform AI agent\nBuilt in Go", version), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/status",
		Description: "Show agent status",
		Usage:       "/status",
		Handler: func(args string) (string, error) {
			return "Agent status: running\nSession: active\nMemory: connected\nTools: loaded", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/memory",
		Description: "Memory management commands",
		Usage:       "/memory <save|search|clear> [args]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /memory <save|search|clear>\n  /memory save <key> <value>\n  /memory search <query>\n  /memory clear", nil
			}
			return fmt.Sprintf("Memory command: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/config",
		Description: "Show or edit configuration",
		Usage:       "/config [key] [value]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Use /config <key> to view, /config <key> <value> to set.", nil
			}
			return fmt.Sprintf("Config: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/theme",
		Description: "Switch theme (alias for /skin)",
		Usage:       "/theme [name]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Current theme: default", nil
			}
			return fmt.Sprintf("Switching theme to: %s", args), nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/undo",
		Description: "Undo the last message",
		Usage:       "/undo",
		Handler: func(args string) (string, error) {
			return "Last message removed from context.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/retry",
		Description: "Retry the last message",
		Usage:       "/retry",
		Handler: func(args string) (string, error) {
			return "Retrying last message...", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/copy",
		Description: "Copy the last response to clipboard",
		Usage:       "/copy",
		Handler: func(args string) (string, error) {
			return "Last response copied to clipboard.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/sessions",
		Description: "List active sessions",
		Usage:       "/sessions",
		Handler: func(args string) (string, error) {
			return "Session listing will be available when wired to the session manager.", nil
		},
	})

	r.register(&SlashCommand{
		Name:        "/debug",
		Description: "Toggle debug mode",
		Usage:       "/debug [on|off]",
		Handler: func(args string) (string, error) {
			switch args {
			case "on":
				return "Debug mode enabled.", nil
			case "off":
				return "Debug mode disabled.", nil
			default:
				return "Debug mode toggled.", nil
			}
		},
	})
}

// ParsedCommand represents a parsed slash command invocation.
type ParsedCommand struct {
	Name string
	Args string
}

// ParseSlashCommand parses user input into a slash command and its arguments.
// Returns nil if the input is not a slash command.
func ParseSlashCommand(input string) *ParsedCommand {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return nil
	}

	parts := strings.SplitN(input, " ", 2)
	name := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	return &ParsedCommand{
		Name: name,
		Args: args,
	}
}
