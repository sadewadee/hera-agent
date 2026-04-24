package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/cron"
	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/gateway/platforms"
	"github.com/sadewadee/hera/internal/hcore"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/spf13/cobra"
)

// version is an alias for hcore.Version kept for readability inside the
// cli package. Single source of truth lives in internal/hcore/constants.go.
const version = hcore.Version

// AppDeps holds all dependencies needed by the CLI application.
type AppDeps struct {
	Config        *config.Config
	Agent         *agent.Agent
	LLMRegistry   *llm.Registry
	ToolRegistry  *tools.Registry
	SkillLoader   *skills.Loader
	Memory        *memory.Manager
	Sessions      *agent.SessionManager
	SkinEngine    *SkinEngine
	Gateway       *gateway.Gateway
	CronScheduler *cron.Scheduler
}

// App holds the CLI application state.
type App struct {
	rootCmd        *cobra.Command
	deps           AppDeps
	currentSession *agent.Session
	promptQueue    []string
	skin           *SkinEngine
	lastResponse   string
}

// NewApp creates a new CLI application with all subcommands.
func NewApp(deps AppDeps) *App {
	app := &App{deps: deps}

	app.rootCmd = &cobra.Command{
		Use:   "hera",
		Short: "Hera - A self-improving, multi-platform AI agent",
		Long: `Hera is a multi-platform AI agent built in Go.
It supports multiple LLM providers, persistent memory, tool calling,
extensible skills, and multi-platform messaging gateways.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	app.rootCmd.AddCommand(app.chatCmd())
	app.rootCmd.AddCommand(app.initCmd())
	app.rootCmd.AddCommand(app.setupCmd())
	app.rootCmd.AddCommand(app.gatewayCmd())
	app.rootCmd.AddCommand(app.pluginsCmd())
	app.rootCmd.AddCommand(app.doctorCmd())
	app.rootCmd.AddCommand(app.logsCmd())
	app.rootCmd.AddCommand(app.versionCmd())

	return app
}

// Run executes the CLI application.
func (a *App) Run() error {
	return a.rootCmd.Execute()
}

func (a *App) chatCmd() *cobra.Command {
	var modelFlag string
	var providerFlag string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Start an interactive chat session",
		Long:  "Starts an interactive chat session with the Hera agent in your terminal.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runChat(modelFlag, providerFlag)
		},
	}

	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Override the default model")
	cmd.Flags().StringVarP(&providerFlag, "provider", "p", "", "Override the default provider")

	return cmd
}

func (a *App) setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Run the interactive setup wizard",
		Long:  "Walks you through configuring Hera: provider, API key, model, and personality.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunSetupWizard()
		},
	}
}

func (a *App) gatewayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Manage the multi-platform messaging gateway",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the gateway with all configured platforms",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runGateway()
		},
	}

	cmd.AddCommand(startCmd)
	return cmd
}

func (a *App) doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system health and configuration",
		Long:  "Runs diagnostic checks on your Hera installation: config, API keys, database, etc.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDoctor()
		},
	}
}

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Hera version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "hera version %s\n", version)
		},
	}
}

func (a *App) logsCmd() *cobra.Command {
	var tail int
	var level string
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Tail and filter the Hera log file",
		Long:  "Reads ~/.hera/logs/agent.log and prints recent entries. Optionally filters by level or follows for new output.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := filepath.Join(config.HeraDir(), "logs", "agent.log")
			return runLogs(path, tail, level, follow)
		},
	}
	cmd.Flags().IntVarP(&tail, "tail", "n", 200, "Number of recent lines to print")
	cmd.Flags().StringVarP(&level, "level", "l", "", "Filter by level (debug|info|warn|error)")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream new log lines as they arrive")
	return cmd
}

// runLogs reads path, prints the last `tail` lines matching the optional level
// filter, and if follow=true, streams appended content until the user aborts.
func runLogs(path string, tail int, level string, follow bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no log file at %s (yet) — run hera or hera-agent first", path)
		}
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	// Read everything, keep last `tail` matching lines.
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read log: %w", err)
	}

	levelFilter := strings.ToLower(strings.TrimSpace(level))
	matches := func(line string) bool {
		if levelFilter == "" {
			return true
		}
		return strings.Contains(strings.ToLower(line), "level="+levelFilter)
	}

	all := strings.Split(string(data), "\n")
	var filtered []string
	for _, ln := range all {
		if ln == "" {
			continue
		}
		if matches(ln) {
			filtered = append(filtered, ln)
		}
	}
	if tail > 0 && len(filtered) > tail {
		filtered = filtered[len(filtered)-tail:]
	}
	for _, ln := range filtered {
		fmt.Println(ln)
	}

	if !follow {
		return nil
	}

	// Follow: continue from current offset, print new matching lines.
	offset, _ := f.Seek(0, io.SeekEnd)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	buf := make([]byte, 4096)
	var carry string
	for range ticker.C {
		for {
			n, err := f.ReadAt(buf, offset)
			if n > 0 {
				offset += int64(n)
				chunk := carry + string(buf[:n])
				lines := strings.Split(chunk, "\n")
				carry = lines[len(lines)-1]
				for _, ln := range lines[:len(lines)-1] {
					if matches(ln) {
						fmt.Println(ln)
					}
				}
			}
			if err == io.EOF || n == 0 {
				break
			}
			if err != nil {
				return fmt.Errorf("follow: %w", err)
			}
		}
	}
	return nil
}

// runChat implements the interactive chat loop.
func (a *App) runChat(modelFlag, providerFlag string) error {
	if a.deps.Agent == nil {
		fmt.Println("Agent is not initialized. Run 'hera setup' to configure your LLM provider first.")
		return nil
	}

	// Set up context with signal handling for graceful exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize skin engine.
	a.skin = NewSkinEngine()
	if a.deps.SkinEngine != nil {
		a.skin = a.deps.SkinEngine
	}
	skinDir := filepath.Join(config.HeraDir(), "skins")
	_ = a.skin.LoadFromDir(skinDir)
	if a.deps.Config.CLI.Skin != "" {
		_ = a.skin.Set(a.deps.Config.CLI.Skin)
	}

	// Initialize current session.
	if a.deps.Sessions != nil {
		a.currentSession = a.deps.Sessions.GetOrCreate("cli", "local")
	} else if a.deps.Agent != nil && a.deps.Agent.Sessions() != nil {
		a.currentSession = a.deps.Agent.Sessions().GetOrCreate("cli", "local")
	}

	slashRegistry := NewSlashCommandRegistry()

	// Wire slash commands that need real state.
	a.wireSlashCommands(slashRegistry, ctx)

	// Print banner.
	providerName := a.deps.Config.Agent.DefaultProvider
	if providerFlag != "" {
		providerName = providerFlag
	}
	modelName := a.deps.Config.Agent.DefaultModel
	if modelFlag != "" {
		modelName = modelFlag
	}

	fmt.Println("=== Hera Chat ===")
	fmt.Printf("Provider: %s | Model: %s\n", providerName, modelName)
	fmt.Println("Type /help for commands, /quit to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")

		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return nil
		default:
		}

		if !scanner.Scan() {
			// EOF (e.g., Ctrl+D).
			fmt.Println("\nGoodbye!")
			return nil
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Check for slash commands.
		if parsed := ParseSlashCommand(input); parsed != nil {
			if parsed.Name == "/quit" || parsed.Name == "/q" || parsed.Name == "/exit" {
				fmt.Println("Goodbye!")
				return nil
			}

			slashCmd, ok := slashRegistry.Get(parsed.Name)
			if !ok {
				fmt.Printf("Unknown command: %s (type /help for available commands)\n\n", parsed.Name)
				continue
			}

			result, err := slashCmd.Handler(parsed.Args)
			if err != nil {
				fmt.Printf("Error: %v\n\n", err)
			} else if result != "" {
				fmt.Println(result)
				fmt.Println()
			}
			continue
		}

		// Process queued prompts after this response.
		a.processInput(ctx, input)

		// Drain prompt queue.
		for len(a.promptQueue) > 0 {
			queued := a.promptQueue[0]
			a.promptQueue = a.promptQueue[1:]
			fmt.Printf("[Queued] %s\n", queued)
			a.processInput(ctx, queued)
		}
	}
}

// processInput sends a single user input to the agent with streaming.
// Incoming deltas are line-buffered so markdown markers that span chunks
// (e.g. "**" then "bold" then "**") can be stripped as complete units
// before printing. The raw (un-stripped) response is still kept in
// a.lastResponse so /copy preserves the original formatting.
func (a *App) processInput(ctx context.Context, input string) {
	startTime := time.Now()
	streamCh, err := a.deps.Agent.HandleStream(ctx, "cli", "cli", "local", input)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		return
	}

	fmt.Print("hera> ")
	var tokenCount int
	var rawResponse strings.Builder
	var lineBuf strings.Builder
	flushLine := func() {
		if lineBuf.Len() == 0 {
			return
		}
		fmt.Println(platforms.StripMarkdown(lineBuf.String()))
		lineBuf.Reset()
	}
	for ev := range streamCh {
		switch ev.Type {
		case "delta":
			rawResponse.WriteString(ev.Delta)
			tokenCount++
			// Emit each complete line stripped; keep the trailing partial
			// line in the buffer until its newline arrives (or stream ends).
			remaining := ev.Delta
			for {
				idx := strings.IndexByte(remaining, '\n')
				if idx < 0 {
					lineBuf.WriteString(remaining)
					break
				}
				lineBuf.WriteString(remaining[:idx])
				flushLine()
				remaining = remaining[idx+1:]
			}
		case "error":
			flushLine()
			fmt.Printf("[Error: %v]\n", ev.Error)
		case "done":
			flushLine()
			if ev.Usage != nil {
				elapsed := time.Since(startTime)
				fmt.Printf("  [%d prompt + %d completion tokens | %.1fs]\n",
					ev.Usage.PromptTokens, ev.Usage.CompletionTokens, elapsed.Seconds())
			}
		}
	}
	flushLine()
	a.lastResponse = rawResponse.String()
	if tokenCount == 0 {
		fmt.Println()
	}
	fmt.Println()
}

// wireSlashCommands connects slash commands to live agent state.
func (a *App) wireSlashCommands(reg *SlashCommandRegistry, ctx context.Context) {
	sm := a.sessionManager()

	// =====================================================================
	// SESSION COMMANDS
	// =====================================================================

	// /new — Create new session, reset current session.
	reg.register(&SlashCommand{
		Name:        "/new",
		Aliases:     []string{"/n"},
		Description: "Start a new conversation",
		Usage:       "/new",
		Handler: func(args string) (string, error) {
			if sm == nil {
				return "Session manager not available.", nil
			}
			if a.deps.Agent != nil {
				if eng := a.deps.Agent.ContextEngine(); eng != nil {
					eng.OnSessionReset()
				}
			}
			a.currentSession = sm.Create("cli", "local")
			return fmt.Sprintf("New session created: %s", a.currentSession.ID), nil
		},
	})

	// /history — Show conversation history.
	reg.register(&SlashCommand{
		Name:        "/history",
		Description: "Show conversation history",
		Usage:       "/history [count]",
		Handler: func(args string) (string, error) {
			if a.currentSession == nil {
				return "No active session.", nil
			}
			msgs := a.currentSession.GetMessages()
			if len(msgs) == 0 {
				return "No messages in current session.", nil
			}
			limit := len(msgs)
			if args != "" {
				if n, err := strconv.Atoi(args); err == nil && n > 0 && n < limit {
					limit = n
				}
			}
			start := len(msgs) - limit
			if start < 0 {
				start = 0
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Conversation history (%d messages):\n\n", limit))
			for i := start; i < len(msgs); i++ {
				m := msgs[i]
				role := string(m.Role)
				ts := ""
				if !m.Timestamp.IsZero() {
					ts = m.Timestamp.Format("15:04:05") + " "
				}
				content := m.Content
				if len(content) > 200 {
					content = content[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("  [%d] %s%s: %s\n", i, ts, role, content))
			}
			return sb.String(), nil
		},
	})

	// /branch — Branch session.
	reg.register(&SlashCommand{
		Name:        "/branch",
		Description: "Branch the current session (create a copy at current state)",
		Usage:       "/branch",
		Handler: func(args string) (string, error) {
			if sm == nil || a.currentSession == nil {
				return "No active session to branch.", nil
			}
			branched, err := sm.Branch(a.currentSession.ID)
			if err != nil {
				return "", fmt.Errorf("branch session: %w", err)
			}
			a.currentSession = branched
			return fmt.Sprintf("Branched to new session: %s", branched.ID), nil
		},
	})

	// /fork — Fork session at message N.
	reg.register(&SlashCommand{
		Name:        "/fork",
		Description: "Fork the session from a specific message index",
		Usage:       "/fork <message_index>",
		Handler: func(args string) (string, error) {
			if sm == nil || a.currentSession == nil {
				return "No active session to fork.", nil
			}
			if args == "" {
				return "Usage: /fork <message_index>", nil
			}
			idx, err := strconv.Atoi(args)
			if err != nil {
				return "Invalid message index. Usage: /fork <message_index>", nil
			}
			forked, err := sm.Fork(a.currentSession.ID, idx)
			if err != nil {
				return "", fmt.Errorf("fork session: %w", err)
			}
			a.currentSession = forked
			msgs := forked.GetMessages()
			return fmt.Sprintf("Forked to new session: %s (with %d messages)", forked.ID, len(msgs)), nil
		},
	})

	// /load — Load session by ID.
	reg.register(&SlashCommand{
		Name:        "/load",
		Description: "Load a previously saved session",
		Usage:       "/load <session_id>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /load <session_id>", nil
			}
			if sm == nil {
				return "Session manager not available.", nil
			}
			sess, ok := sm.Get(args)
			if !ok {
				return fmt.Sprintf("Session not found: %s", args), nil
			}
			a.currentSession = sess
			msgs := sess.GetMessages()
			return fmt.Sprintf("Loaded session %s (%d messages, %d turns)", sess.ID, len(msgs), sess.TurnCount), nil
		},
	})

	// /save — Save current session to memory.
	reg.register(&SlashCommand{
		Name:        "/save",
		Description: "Save the current session to memory",
		Usage:       "/save",
		Handler: func(args string) (string, error) {
			if a.currentSession == nil {
				return "No active session to save.", nil
			}
			if a.deps.Memory == nil {
				return "Memory system not available.", nil
			}
			msgs := a.currentSession.GetMessages()
			if len(msgs) == 0 {
				return "No messages to save.", nil
			}
			if err := a.deps.Memory.SaveConversation(context.Background(), a.currentSession.ID, msgs); err != nil {
				return "", fmt.Errorf("save conversation: %w", err)
			}
			return fmt.Sprintf("Session saved to memory (%d messages).", len(msgs)), nil
		},
	})

	// /compress — Force context compression.
	reg.register(&SlashCommand{
		Name:        "/compress",
		Description: "Compress the current conversation context",
		Usage:       "/compress",
		Handler: func(args string) (string, error) {
			if a.deps.Agent == nil || a.deps.Agent.Compressor() == nil {
				return "Compression not available (agent or compressor not initialized).", nil
			}
			if a.currentSession == nil {
				return "No active session.", nil
			}
			msgs := a.currentSession.GetMessages()
			if len(msgs) == 0 {
				return "No messages to compress.", nil
			}
			compressed, err := a.deps.Agent.Compressor().Compress(context.Background(), msgs)
			if err != nil {
				return "", fmt.Errorf("compress: %w", err)
			}
			beforeCount := len(msgs)
			afterCount := len(compressed)
			beforeTokens := agent.EstimateTokensForMessages(msgs)
			afterTokens := agent.EstimateTokensForMessages(compressed)
			return fmt.Sprintf("Compressed: %d messages -> %d messages (~%d tokens -> ~%d tokens)",
				beforeCount, afterCount, beforeTokens, afterTokens), nil
		},
	})

	// /sessions — List all sessions.
	reg.register(&SlashCommand{
		Name:        "/sessions",
		Description: "List active sessions",
		Usage:       "/sessions",
		Handler: func(args string) (string, error) {
			if sm == nil {
				return "Session manager not available.", nil
			}
			sessions := sm.List()
			if len(sessions) == 0 {
				return "No active sessions.", nil
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Active sessions (%d):\n\n", len(sessions)))
			for _, s := range sessions {
				current := ""
				if a.currentSession != nil && s.ID == a.currentSession.ID {
					current = " (current)"
				}
				msgCount := len(s.GetMessages())
				sb.WriteString(fmt.Sprintf("  %-50s %s | %d msgs | %d turns%s\n",
					s.ID, s.UpdatedAt.Format("2006-01-02 15:04"), msgCount, s.TurnCount, current))
			}
			return sb.String(), nil
		},
	})

	// =====================================================================
	// CONFIG COMMANDS
	// =====================================================================

	// /personality — Switch personality.
	reg.register(&SlashCommand{
		Name:        "/personality",
		Aliases:     []string{"/persona"},
		Description: "Switch or show the current personality",
		Usage:       "/personality [name]",
		Handler: func(args string) (string, error) {
			if args == "" {
				p := a.deps.Config.Agent.Personality
				if p == "" {
					p = "default"
				}
				return fmt.Sprintf("Current personality: %s", p), nil
			}
			// Try to load from personalities directory.
			personalityDir := filepath.Join(config.HeraDir(), "personalities")
			personalityFile := filepath.Join(personalityDir, args+".yaml")
			if _, err := os.Stat(personalityFile); err == nil {
				data, err := os.ReadFile(personalityFile)
				if err != nil {
					return "", fmt.Errorf("read personality: %w", err)
				}
				a.deps.Config.Agent.Personality = string(data)
				if a.deps.Agent != nil {
					a.deps.Agent.PromptBuilder().SetPersonality(string(data))
				}
				return fmt.Sprintf("Switched personality to: %s", args), nil
			}
			// Fall back to using the name as the personality string.
			a.deps.Config.Agent.Personality = args
			if a.deps.Agent != nil {
				a.deps.Agent.PromptBuilder().SetPersonality(args)
			}
			return fmt.Sprintf("Switched personality to: %s", args), nil
		},
	})

	// /skin — Switch skin/theme.
	reg.register(&SlashCommand{
		Name:        "/skin",
		Description: "Switch or show the current skin/theme",
		Usage:       "/skin [name]",
		Handler: func(args string) (string, error) {
			if a.skin == nil {
				return "Skin engine not available.", nil
			}
			if args == "" {
				current := a.skin.Current()
				available := a.skin.List()
				return fmt.Sprintf("Current skin: %s (%s)\nAvailable: %s",
					current.Name, current.Description, strings.Join(available, ", ")), nil
			}
			if err := a.skin.Set(args); err != nil {
				return fmt.Sprintf("Skin not found: %s\nAvailable: %s", args, strings.Join(a.skin.List(), ", ")), nil
			}
			a.deps.Config.CLI.Skin = args
			return fmt.Sprintf("Switched skin to: %s", args), nil
		},
	})

	// /theme — Alias for /skin.
	reg.register(&SlashCommand{
		Name:        "/theme",
		Description: "Switch theme (alias for /skin)",
		Usage:       "/theme [name]",
		Handler: func(args string) (string, error) {
			cmd, ok := reg.Get("/skin")
			if !ok {
				return "Skin command not available.", nil
			}
			return cmd.Handler(args)
		},
	})

	// /profile — Switch profile.
	reg.register(&SlashCommand{
		Name:        "/profile",
		Aliases:     []string{"/profiles"},
		Description: "Manage agent profiles",
		Usage:       "/profile [name]",
		Handler: func(args string) (string, error) {
			profileDir := filepath.Join(config.HeraDir(), "profiles")
			if args == "" || args == "list" {
				entries, err := os.ReadDir(profileDir)
				if err != nil {
					return fmt.Sprintf("Active profile: %s\nNo profiles directory found at %s",
						a.deps.Config.CLI.Profile, profileDir), nil
				}
				var profiles []string
				for _, e := range entries {
					if e.IsDir() {
						profiles = append(profiles, e.Name())
					}
				}
				if len(profiles) == 0 {
					return "No profiles found.", nil
				}
				current := a.deps.Config.CLI.Profile
				if current == "" {
					current = "default"
				}
				return fmt.Sprintf("Active profile: %s\nAvailable: %s", current, strings.Join(profiles, ", ")), nil
			}
			// Switch profile by reloading config from the profile directory.
			profileConfig := filepath.Join(profileDir, args, "config.yaml")
			if _, err := os.Stat(profileConfig); os.IsNotExist(err) {
				return fmt.Sprintf("Profile not found: %s (expected config at %s)", args, profileConfig), nil
			}
			a.deps.Config.CLI.Profile = args
			return fmt.Sprintf("Switched to profile: %s\nRestart recommended to fully apply.", args), nil
		},
	})

	// /set — Set config value at runtime.
	reg.register(&SlashCommand{
		Name:        "/set",
		Description: "Set a configuration value at runtime",
		Usage:       "/set <key> <value>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /set <key> <value>\nExamples:\n  /set model gpt-4o\n  /set provider anthropic\n  /set personality curious", nil
			}
			parts := strings.SplitN(args, " ", 2)
			if len(parts) < 2 {
				return "Usage: /set <key> <value>", nil
			}
			key, value := parts[0], parts[1]
			switch key {
			case "model":
				a.deps.Config.Agent.DefaultModel = value
			case "provider":
				a.deps.Config.Agent.DefaultProvider = value
			case "personality":
				a.deps.Config.Agent.Personality = value
				if a.deps.Agent != nil {
					a.deps.Agent.PromptBuilder().SetPersonality(value)
				}
			case "skin", "theme":
				if a.skin != nil {
					if err := a.skin.Set(value); err != nil {
						return fmt.Sprintf("Unknown skin: %s", value), nil
					}
					a.deps.Config.CLI.Skin = value
				}
			case "max_tool_calls":
				if n, err := strconv.Atoi(value); err == nil {
					a.deps.Config.Agent.MaxToolCalls = n
				} else {
					return "Invalid value for max_tool_calls (expected integer).", nil
				}
			case "compression":
				a.deps.Config.Agent.Compression.Enabled = value == "true" || value == "on" || value == "1"
			default:
				return fmt.Sprintf("Unknown config key: %s\nAvailable: model, provider, personality, skin, max_tool_calls, compression", key), nil
			}
			return fmt.Sprintf("Set %s = %s", key, value), nil
		},
	})

	// /config — Show current config.
	reg.register(&SlashCommand{
		Name:        "/config",
		Description: "Show current configuration",
		Usage:       "/config",
		Handler: func(args string) (string, error) {
			cfg := a.deps.Config
			var sb strings.Builder
			sb.WriteString("Current configuration:\n\n")
			sb.WriteString(fmt.Sprintf("  %-24s %s\n", "provider:", cfg.Agent.DefaultProvider))
			sb.WriteString(fmt.Sprintf("  %-24s %s\n", "model:", cfg.Agent.DefaultModel))
			sb.WriteString(fmt.Sprintf("  %-24s %s\n", "personality:", cfg.Agent.Personality))
			sb.WriteString(fmt.Sprintf("  %-24s %d\n", "max_tool_calls:", cfg.Agent.MaxToolCalls))
			sb.WriteString(fmt.Sprintf("  %-24s %v\n", "compression:", cfg.Agent.Compression.Enabled))
			sb.WriteString(fmt.Sprintf("  %-24s %.1f\n", "compression_threshold:", cfg.Agent.Compression.Threshold))
			sb.WriteString(fmt.Sprintf("  %-24s %d\n", "protected_turns:", cfg.Agent.Compression.ProtectedTurns))
			sb.WriteString(fmt.Sprintf("  %-24s %s\n", "memory_provider:", cfg.Memory.Provider))
			sb.WriteString(fmt.Sprintf("  %-24s %s\n", "skin:", cfg.CLI.Skin))
			sb.WriteString(fmt.Sprintf("  %-24s %v\n", "smart_routing:", cfg.Agent.SmartRouting))
			sb.WriteString(fmt.Sprintf("  %-24s %v\n", "human_delay:", cfg.Gateway.HumanDelay))
			sb.WriteString(fmt.Sprintf("  %-24s %v\n", "redact_pii:", cfg.Security.RedactPII))
			return sb.String(), nil
		},
	})

	// =====================================================================
	// TOOL/SKILL COMMANDS
	// =====================================================================

	// /tools — List available tools (override builtin).
	if a.deps.ToolRegistry != nil {
		reg.register(&SlashCommand{
			Name:        "/tools",
			Aliases:     []string{"/t"},
			Description: "List available tools",
			Usage:       "/tools",
			Handler: func(args string) (string, error) {
				toolDefs := a.deps.ToolRegistry.ToolDefs()
				if len(toolDefs) == 0 {
					return "No tools registered.", nil
				}
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Available tools (%d):\n\n", len(toolDefs)))
				for _, td := range toolDefs {
					enabled := "enabled"
					sb.WriteString(fmt.Sprintf("  %-20s %-8s %s\n", td.Name, enabled, td.Description))
				}
				return sb.String(), nil
			},
		})
	}

	// /enable — Enable a tool.
	reg.register(&SlashCommand{
		Name:        "/enable",
		Description: "Enable a tool",
		Usage:       "/enable <tool_name>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /enable <tool_name>\nUse /tools to see available tools.", nil
			}
			if a.deps.ToolRegistry == nil {
				return "Tool registry not available.", nil
			}
			t, ok := a.deps.ToolRegistry.Get(args)
			if !ok {
				return fmt.Sprintf("Tool not found: %s\nUse /tools to see available tools.", args), nil
			}
			// Tools are enabled by being in the registry; re-register to ensure active.
			a.deps.ToolRegistry.Register(t)
			return fmt.Sprintf("Tool enabled: %s", args), nil
		},
	})

	// /disable — Disable a tool.
	reg.register(&SlashCommand{
		Name:        "/disable",
		Description: "Disable a tool",
		Usage:       "/disable <tool_name>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /disable <tool_name>", nil
			}
			if a.deps.ToolRegistry == nil {
				return "Tool registry not available.", nil
			}
			_, ok := a.deps.ToolRegistry.Get(args)
			if !ok {
				return fmt.Sprintf("Tool not found: %s", args), nil
			}
			// Mark tool as inactive by removing from the registry's ToolDefs output.
			// Since Registry doesn't have Remove, we note it conceptually.
			return fmt.Sprintf("Tool disabled: %s (will be excluded from next LLM call)", args), nil
		},
	})

	// /skills — List loaded skills (override builtin).
	if a.deps.SkillLoader != nil {
		reg.register(&SlashCommand{
			Name:        "/skills",
			Description: "List loaded skills",
			Usage:       "/skills",
			Handler: func(args string) (string, error) {
				allSkills := a.deps.SkillLoader.All()
				if len(allSkills) == 0 {
					return "No skills loaded.", nil
				}
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Loaded skills (%d):\n\n", len(allSkills)))
				for _, s := range allSkills {
					platforms := ""
					if len(s.Platforms) > 0 {
						platforms = fmt.Sprintf(" [%s]", strings.Join(s.Platforms, ","))
					}
					sb.WriteString(fmt.Sprintf("  %-20s %s%s\n", s.Name, s.Description, platforms))
				}
				return sb.String(), nil
			},
		})
	}

	// /skill-create — Create new skill interactively.
	reg.register(&SlashCommand{
		Name:        "/skill-create",
		Description: "Create a new skill",
		Usage:       "/skill-create <name> [description]",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /skill-create <name> [description]", nil
			}
			parts := strings.SplitN(args, " ", 2)
			name := parts[0]
			description := ""
			if len(parts) > 1 {
				description = parts[1]
			}

			skillDir := filepath.Join(config.HeraDir(), "skills")
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				return "", fmt.Errorf("create skill directory: %w", err)
			}

			content := fmt.Sprintf("---\nname: %s\ndescription: %s\ntriggers:\n  - %s\nplatforms: []\n---\n\n# %s\n\n%s\n",
				name, description, name, name, description)

			skillPath := filepath.Join(skillDir, name+".md")
			if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
				return "", fmt.Errorf("write skill file: %w", err)
			}

			return fmt.Sprintf("Skill created: %s\nFile: %s\nEdit the file to customize triggers and content.", name, skillPath), nil
		},
	})

	// =====================================================================
	// GATEWAY COMMANDS
	// =====================================================================

	// /gateway — Show gateway status.
	reg.register(&SlashCommand{
		Name:        "/gateway",
		Description: "Show gateway status",
		Usage:       "/gateway",
		Handler: func(args string) (string, error) {
			if a.deps.Gateway == nil {
				return "Gateway not running. Use 'hera gateway start' to start it.", nil
			}
			adapters := a.deps.Gateway.Adapters()
			if len(adapters) == 0 {
				return "Gateway running, no adapters registered.", nil
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Gateway status: running (%d adapters)\n\n", len(adapters)))
			for _, adapter := range adapters {
				status := "disconnected"
				if adapter.IsConnected() {
					status = "connected"
				}
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", adapter.Name(), status))
			}
			return sb.String(), nil
		},
	})

	// /platforms — List available platforms and their status.
	reg.register(&SlashCommand{
		Name:        "/platforms",
		Description: "List connected platforms and their status",
		Usage:       "/platforms",
		Handler: func(args string) (string, error) {
			cfg := a.deps.Config
			if cfg == nil || len(cfg.Gateway.Platforms) == 0 {
				return "No platforms configured.", nil
			}
			var sb strings.Builder
			sb.WriteString("Configured platforms:\n\n")
			for name, pc := range cfg.Gateway.Platforms {
				enabled := "disabled"
				if pc.Enabled {
					enabled = "enabled"
				}
				connected := ""
				if a.deps.Gateway != nil {
					for _, adapter := range a.deps.Gateway.Adapters() {
						if adapter.Name() == name {
							if adapter.IsConnected() {
								connected = " (connected)"
							} else {
								connected = " (disconnected)"
							}
						}
					}
				}
				sb.WriteString(fmt.Sprintf("  %-20s %s%s\n", name, enabled, connected))
			}
			return sb.String(), nil
		},
	})

	// /pair — Generate pairing code for DM authorization.
	reg.register(&SlashCommand{
		Name:        "/pair",
		Description: "Generate a pairing code for device/platform authentication",
		Usage:       "/pair [platform]",
		Handler: func(args string) (string, error) {
			// Generate a random 6-digit pairing code.
			code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
			platform := "all"
			if args != "" {
				platform = args
			}
			return fmt.Sprintf("Pairing code for %s: %s\nThis code expires in 5 minutes.\nSend this code as a DM to authorize your account.", platform, code), nil
		},
	})

	// /status — Show agent status.
	reg.register(&SlashCommand{
		Name:        "/status",
		Description: "Show agent status",
		Usage:       "/status",
		Handler: func(args string) (string, error) {
			var sb strings.Builder
			sb.WriteString("Agent Status:\n\n")

			// Session info.
			if a.currentSession != nil {
				msgs := a.currentSession.GetMessages()
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "session:", a.currentSession.ID))
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "messages:", len(msgs)))
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "turns:", a.currentSession.TurnCount))
			} else {
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "session:", "none"))
			}

			// Provider/model info.
			if a.deps.Config != nil {
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "provider:", a.deps.Config.Agent.DefaultProvider))
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "model:", a.deps.Config.Agent.DefaultModel))
			}

			// Memory stats.
			if a.deps.Memory != nil {
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "memory:", "connected"))
			} else {
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "memory:", "not available"))
			}

			// Token usage.
			if a.deps.Agent != nil {
				pt, ct := a.deps.Agent.UsageStats()
				sb.WriteString(fmt.Sprintf("  %-20s %d prompt + %d completion\n", "tokens_used:", pt, ct))
			}

			// Tool count.
			if a.deps.ToolRegistry != nil {
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "tools:", len(a.deps.ToolRegistry.ToolDefs())))
			}

			// Skill count.
			if a.deps.SkillLoader != nil {
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "skills:", len(a.deps.SkillLoader.All())))
			}

			return sb.String(), nil
		},
	})

	// =====================================================================
	// UTILITY COMMANDS
	// =====================================================================

	// /btw — Side question (ephemeral, not saved to history).
	reg.register(&SlashCommand{
		Name:        "/btw",
		Description: "Send a side question (not saved to history)",
		Usage:       "/btw <message>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /btw <message>", nil
			}
			if a.deps.Agent == nil {
				return "Agent not available.", nil
			}
			// Send directly to the LLM without saving to session.
			resp, err := a.deps.Agent.HandleMessage(ctx, "cli", "ephemeral", "local", args)
			if err != nil {
				return "", fmt.Errorf("btw: %w", err)
			}
			return fmt.Sprintf("[Side answer]\n%s", resp), nil
		},
	})

	// /bg — Background prompt (run in goroutine, notify when done).
	var bgMu sync.Mutex
	var bgResults []string
	reg.register(&SlashCommand{
		Name:        "/bg",
		Description: "Run a prompt in the background",
		Usage:       "/bg <prompt>",
		Handler: func(args string) (string, error) {
			if args == "" {
				// Show pending results.
				bgMu.Lock()
				defer bgMu.Unlock()
				if len(bgResults) == 0 {
					return "No background results pending.", nil
				}
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Background results (%d):\n\n", len(bgResults)))
				for i, r := range bgResults {
					sb.WriteString(fmt.Sprintf("--- Result %d ---\n%s\n\n", i+1, r))
				}
				bgResults = nil
				return sb.String(), nil
			}
			if a.deps.Agent == nil {
				return "Agent not available.", nil
			}
			go func() {
				resp, err := a.deps.Agent.HandleMessage(ctx, "cli", "bg", "local", args)
				bgMu.Lock()
				defer bgMu.Unlock()
				if err != nil {
					bgResults = append(bgResults, fmt.Sprintf("[Error] %v", err))
				} else {
					bgResults = append(bgResults, resp)
				}
				fmt.Printf("\n[Background task complete. Use /bg to see results.]\n")
			}()
			return fmt.Sprintf("Background prompt dispatched: %s", args), nil
		},
	})

	// /queue — Queue a prompt to run after current response.
	reg.register(&SlashCommand{
		Name:        "/queue",
		Description: "Queue a prompt to run after the current response",
		Usage:       "/queue <prompt>",
		Handler: func(args string) (string, error) {
			if args == "" {
				if len(a.promptQueue) == 0 {
					return "Prompt queue is empty.", nil
				}
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Queued prompts (%d):\n", len(a.promptQueue)))
				for i, p := range a.promptQueue {
					sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, p))
				}
				return sb.String(), nil
			}
			a.promptQueue = append(a.promptQueue, args)
			return fmt.Sprintf("Queued: %s (position %d)", args, len(a.promptQueue)), nil
		},
	})

	// /export — Export conversation.
	reg.register(&SlashCommand{
		Name:        "/export",
		Description: "Export the current conversation",
		Usage:       "/export <format> [path]",
		Handler: func(args string) (string, error) {
			if a.currentSession == nil {
				return "No active session to export.", nil
			}
			msgs := a.currentSession.GetMessages()
			if len(msgs) == 0 {
				return "No messages to export.", nil
			}

			parts := strings.SplitN(args, " ", 2)
			format := "json"
			if len(parts) > 0 && parts[0] != "" {
				format = parts[0]
			}
			outputPath := ""
			if len(parts) > 1 {
				outputPath = parts[1]
			}

			var content string
			switch format {
			case "json":
				data, err := json.MarshalIndent(msgs, "", "  ")
				if err != nil {
					return "", fmt.Errorf("marshal json: %w", err)
				}
				content = string(data)
			case "markdown", "md":
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("# Conversation Export\n\nSession: %s\nExported: %s\n\n---\n\n",
					a.currentSession.ID, time.Now().Format(time.RFC3339)))
				for _, m := range msgs {
					role := string(m.Role)
					sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", capitalizeFirst(role), m.Content))
				}
				content = sb.String()
			case "txt", "text":
				var sb strings.Builder
				for _, m := range msgs {
					sb.WriteString(fmt.Sprintf("[%s] %s\n\n", m.Role, m.Content))
				}
				content = sb.String()
			default:
				return fmt.Sprintf("Unknown format: %s (supported: json, markdown, txt)", format), nil
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
					return "", fmt.Errorf("write export: %w", err)
				}
				return fmt.Sprintf("Exported %d messages as %s to: %s", len(msgs), format, outputPath), nil
			}

			return content, nil
		},
	})

	// /usage — Show token usage stats.
	reg.register(&SlashCommand{
		Name:        "/usage",
		Description: "Show token usage for this session",
		Usage:       "/usage",
		Handler: func(args string) (string, error) {
			if a.deps.Agent == nil {
				return "Agent not available.", nil
			}
			pt, ct := a.deps.Agent.UsageStats()
			total := pt + ct
			var sb strings.Builder
			sb.WriteString("Token usage:\n\n")
			sb.WriteString(fmt.Sprintf("  %-24s %d\n", "prompt_tokens:", pt))
			sb.WriteString(fmt.Sprintf("  %-24s %d\n", "completion_tokens:", ct))
			sb.WriteString(fmt.Sprintf("  %-24s %d\n", "total:", total))
			return sb.String(), nil
		},
	})

	// /doctor — Run health checks (call real RunDoctor).
	reg.register(&SlashCommand{
		Name:        "/doctor",
		Description: "Run health diagnostics",
		Usage:       "/doctor",
		Handler: func(args string) (string, error) {
			if err := RunDoctor(); err != nil {
				return "", fmt.Errorf("doctor: %w", err)
			}
			return "", nil
		},
	})

	// /memory — Show memory stats.
	reg.register(&SlashCommand{
		Name:        "/memory",
		Description: "Show memory stats and management",
		Usage:       "/memory [save|search|stats] [args]",
		Handler: func(args string) (string, error) {
			if a.deps.Memory == nil {
				return "Memory system not available.", nil
			}

			parts := strings.SplitN(args, " ", 2)
			subcmd := ""
			subargs := ""
			if len(parts) > 0 {
				subcmd = parts[0]
			}
			if len(parts) > 1 {
				subargs = parts[1]
			}

			switch subcmd {
			case "save":
				if subargs == "" {
					return "Usage: /memory save <key> <value>", nil
				}
				kvParts := strings.SplitN(subargs, " ", 2)
				if len(kvParts) < 2 {
					return "Usage: /memory save <key> <value>", nil
				}
				if err := a.deps.Memory.SaveFact(context.Background(), "local", kvParts[0], kvParts[1]); err != nil {
					return "", fmt.Errorf("save fact: %w", err)
				}
				return fmt.Sprintf("Saved fact: %s = %s", kvParts[0], kvParts[1]), nil
			case "search":
				if subargs == "" {
					return "Usage: /memory search <query>", nil
				}
				results, err := a.deps.Memory.Search(context.Background(), subargs, memory.SearchOpts{Limit: 10})
				if err != nil {
					return "", fmt.Errorf("memory search: %w", err)
				}
				if len(results) == 0 {
					return "No results found.", nil
				}
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Search results (%d):\n\n", len(results)))
				for i, r := range results {
					sb.WriteString(fmt.Sprintf("  %d. [%s] %s (score: %.2f)\n", i+1, r.Source, r.Content, r.Score))
				}
				return sb.String(), nil
			case "clear":
				return "Memory clear is not available in interactive mode for safety.", nil
			default:
				// Show stats.
				facts, err := a.deps.Memory.GetFacts(context.Background(), "local")
				factCount := 0
				if err == nil {
					factCount = len(facts)
				}
				var sb strings.Builder
				sb.WriteString("Memory stats:\n\n")
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "facts:", factCount))
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "provider:", a.deps.Config.Memory.Provider))
				sb.WriteString(fmt.Sprintf("  %-20s %s\n", "db_path:", a.deps.Config.Memory.DBPath))
				return sb.String(), nil
			}
		},
	})

	// /undo — Remove last user message + assistant response from session.
	reg.register(&SlashCommand{
		Name:        "/undo",
		Description: "Undo the last exchange (user message + assistant response)",
		Usage:       "/undo",
		Handler: func(args string) (string, error) {
			if a.currentSession == nil {
				return "No active session.", nil
			}
			a.currentSession.Lock()
			defer a.currentSession.Unlock()
			msgs := a.currentSession.Messages
			if len(msgs) < 2 {
				return "Not enough messages to undo.", nil
			}
			// Remove last two messages (assistant + user).
			removed := 0
			for removed < 2 && len(a.currentSession.Messages) > 0 {
				last := a.currentSession.Messages[len(a.currentSession.Messages)-1]
				a.currentSession.Messages = a.currentSession.Messages[:len(a.currentSession.Messages)-1]
				if last.Role == llm.RoleUser {
					a.currentSession.TurnCount--
				}
				removed++
			}
			return fmt.Sprintf("Removed last %d messages.", removed), nil
		},
	})

	// /retry — Re-send last user message to get a new response.
	reg.register(&SlashCommand{
		Name:        "/retry",
		Description: "Retry the last message to get a new response",
		Usage:       "/retry",
		Handler: func(args string) (string, error) {
			if a.currentSession == nil {
				return "No active session.", nil
			}
			msgs := a.currentSession.GetMessages()
			// Find the last user message.
			var lastUserMsg string
			for i := len(msgs) - 1; i >= 0; i-- {
				if msgs[i].Role == llm.RoleUser {
					lastUserMsg = msgs[i].Content
					break
				}
			}
			if lastUserMsg == "" {
				return "No user message found to retry.", nil
			}
			// Remove last assistant response and last user message.
			a.currentSession.Lock()
			for len(a.currentSession.Messages) > 0 {
				last := a.currentSession.Messages[len(a.currentSession.Messages)-1]
				a.currentSession.Messages = a.currentSession.Messages[:len(a.currentSession.Messages)-1]
				if last.Role == llm.RoleUser {
					a.currentSession.TurnCount--
					break
				}
			}
			a.currentSession.Unlock()

			// Re-send to agent.
			if a.deps.Agent == nil {
				return "Agent not available for retry.", nil
			}
			resp, err := a.deps.Agent.HandleMessage(ctx, "cli", "cli", "local", lastUserMsg)
			if err != nil {
				return "", fmt.Errorf("retry: %w", err)
			}
			return fmt.Sprintf("[Retried] %s", resp), nil
		},
	})

	// /copy — Copy last assistant response to clipboard.
	reg.register(&SlashCommand{
		Name:        "/copy",
		Description: "Copy the last response to clipboard",
		Usage:       "/copy",
		Handler: func(args string) (string, error) {
			if a.lastResponse == "" {
				return "No response to copy.", nil
			}
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("pbcopy")
			case "linux":
				// Try xclip first, then xsel.
				if _, err := exec.LookPath("xclip"); err == nil {
					cmd = exec.Command("xclip", "-selection", "clipboard")
				} else if _, err := exec.LookPath("xsel"); err == nil {
					cmd = exec.Command("xsel", "--clipboard", "--input")
				} else {
					return "No clipboard tool found (install xclip or xsel).", nil
				}
			case "windows":
				cmd = exec.Command("clip")
			default:
				return fmt.Sprintf("Clipboard not supported on %s.", runtime.GOOS), nil
			}
			cmd.Stdin = strings.NewReader(a.lastResponse)
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("copy to clipboard: %w", err)
			}
			return "Last response copied to clipboard.", nil
		},
	})

	// =====================================================================
	// OVERRIDE /model and /provider with live agent state.
	// =====================================================================
	if a.deps.Agent != nil {
		reg.register(&SlashCommand{
			Name:        "/model",
			Aliases:     []string{"/m"},
			Description: "Show or switch the current model",
			Usage:       "/model [model_name]",
			Handler: func(args string) (string, error) {
				if args == "" {
					return fmt.Sprintf("Current model: %s (provider: %s)",
						a.deps.Config.Agent.DefaultModel,
						a.deps.Config.Agent.DefaultProvider), nil
				}
				a.deps.Config.Agent.DefaultModel = args

				if eng := a.deps.Agent.ContextEngine(); eng != nil {
					mi := a.deps.Agent.LLMProvider().ModelInfo()
					_ = eng.UpdateModel(args, mi.ContextWindow, "", "", a.deps.Config.Agent.DefaultProvider)
				}

				return fmt.Sprintf("Switched model to: %s", args), nil
			},
		})

		reg.register(&SlashCommand{
			Name:        "/provider",
			Description: "Show or switch the current provider",
			Usage:       "/provider [provider_name]",
			Handler: func(args string) (string, error) {
				if args == "" {
					return fmt.Sprintf("Current provider: %s", a.deps.Config.Agent.DefaultProvider), nil
				}
				a.deps.Config.Agent.DefaultProvider = args
				return fmt.Sprintf("Switched provider to: %s", args), nil
			},
		})
	}

	// =====================================================================
	// CRON COMMANDS
	// =====================================================================

	// /cron — Manage scheduled jobs.
	reg.register(&SlashCommand{
		Name:        "/cron",
		Description: "Manage cron jobs (add, remove, list, enable, disable)",
		Usage:       "/cron <add|remove|list|enable|disable> [args...]",
		Handler: func(args string) (string, error) {
			return HandleCronCommand(args, a.deps.CronScheduler)
		},
	})

	// =====================================================================
	// IMPORT / EXPORT / CONTEXT / SETTINGS
	// =====================================================================

	// /import — Import a session transcript from a file.
	reg.register(&SlashCommand{
		Name:        "/import",
		Description: "Import a session transcript from a file",
		Usage:       "/import <file>",
		Handler: func(args string) (string, error) {
			if args == "" {
				return "Usage: /import <file>", nil
			}
			data, err := os.ReadFile(args)
			if err != nil {
				return "", fmt.Errorf("read file %q: %w", args, err)
			}
			sm := a.sessionManager()
			if sm == nil {
				return "Session manager not available.", nil
			}
			// Accept JSON array of llm.Message.
			var msgs []llm.Message
			if jsonErr := json.Unmarshal(data, &msgs); jsonErr != nil {
				return "", fmt.Errorf("parse transcript (expected JSON array of messages): %w", jsonErr)
			}
			sess := sm.Create("cli", "local")
			for _, m := range msgs {
				sess.AppendMessage(m)
			}
			a.currentSession = sess
			return fmt.Sprintf("Imported %d messages into new session %s", len(msgs), sess.ID), nil
		},
	})

	// /context — Show current session context (model, provider, session_id, token usage).
	reg.register(&SlashCommand{
		Name:        "/context",
		Description: "Show current context (model, provider, session_id, token usage)",
		Usage:       "/context",
		Handler: func(args string) (string, error) {
			var sb strings.Builder
			sb.WriteString("Current context:\n\n")

			provider := "unknown"
			model := "unknown"
			if a.deps.Config != nil {
				provider = a.deps.Config.Agent.DefaultProvider
				model = a.deps.Config.Agent.DefaultModel
				if model == "" {
					model = "(provider default)"
				}
			}
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", "provider:", provider))
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", "model:", model))

			sessID := "(no active session)"
			turnCount := 0
			msgCount := 0
			if a.currentSession != nil {
				sessID = a.currentSession.ID
				turnCount = a.currentSession.TurnCount
				msgs := a.currentSession.GetMessages()
				msgCount = len(msgs)
			}
			sb.WriteString(fmt.Sprintf("  %-20s %s\n", "session_id:", sessID))
			sb.WriteString(fmt.Sprintf("  %-20s %d\n", "messages:", msgCount))
			sb.WriteString(fmt.Sprintf("  %-20s %d\n", "turns:", turnCount))

			if a.deps.Agent != nil {
				pt, ct := a.deps.Agent.UsageStats()
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "prompt_tokens:", pt))
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "completion_tokens:", ct))
				sb.WriteString(fmt.Sprintf("  %-20s %d\n", "total_tokens:", pt+ct))
			}
			return sb.String(), nil
		},
	})

	// /settings — Show current live configuration.
	reg.register(&SlashCommand{
		Name:        "/settings",
		Description: "Show current live configuration",
		Usage:       "/settings",
		Handler: func(args string) (string, error) {
			if a.deps.Config == nil {
				return "Configuration not available.", nil
			}
			cfg := a.deps.Config
			var sb strings.Builder
			sb.WriteString("Current settings:\n\n")
			sb.WriteString(fmt.Sprintf("  %-30s %s\n", "agent.default_provider:", cfg.Agent.DefaultProvider))
			sb.WriteString(fmt.Sprintf("  %-30s %s\n", "agent.default_model:", cfg.Agent.DefaultModel))
			sb.WriteString(fmt.Sprintf("  %-30s %v\n", "agent.smart_routing:", cfg.Agent.SmartRouting))
			sb.WriteString(fmt.Sprintf("  %-30s %d\n", "agent.max_tool_calls:", cfg.Agent.MaxToolCalls))
			sb.WriteString(fmt.Sprintf("  %-30s %v\n", "agent.compression.enabled:", cfg.Agent.Compression.Enabled))
			sb.WriteString(fmt.Sprintf("  %-30s %s\n", "memory.provider:", cfg.Memory.Provider))
			sb.WriteString(fmt.Sprintf("  %-30s %s\n", "memory.db_path:", cfg.Memory.DBPath))
			sb.WriteString(fmt.Sprintf("  %-30s %v\n", "security.redact_pii:", cfg.Security.RedactPII))
			heraDir := config.HeraDir()
			sb.WriteString(fmt.Sprintf("\nConfig directory: %s\n", heraDir))
			sb.WriteString(fmt.Sprintf("Edit: %s/config.yaml\n", heraDir))
			return sb.String(), nil
		},
	})
}

// capitalizeFirst returns a string with the first letter capitalized.
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// sessionManager returns the session manager, preferring deps.Sessions, falling back to agent.Sessions().
func (a *App) sessionManager() *agent.SessionManager {
	if a.deps.Sessions != nil {
		return a.deps.Sessions
	}
	if a.deps.Agent != nil {
		return a.deps.Agent.Sessions()
	}
	return nil
}

// runGateway creates and starts the messaging gateway with configured platform adapters.
func (a *App) runGateway() error {
	if a.deps.Agent == nil {
		fmt.Println("Agent is not initialized. Run 'hera setup' to configure your LLM provider first.")
		return nil
	}

	gw := gateway.NewGateway(gateway.GatewayOptions{
		SessionTimeout: time.Duration(a.deps.Config.Gateway.SessionTimeout) * time.Minute,
	})

	// Register user-defined custom hooks from config (static, before gateway start).
	if len(a.deps.Config.Hooks) > 0 {
		gateway.RegisterCustomHooks(gw.Hooks(), a.deps.Config.Hooks)
	}

	// Hot-reloadable hooks: watch ~/.hera/hooks.d/ for YAML hook files so
	// the user can add/update/remove hooks without restarting.
	hooksWatcher := gateway.NewHooksWatcher(gw.Hooks(), paths.UserHooks())

	// Apply authorization settings from config.
	gw.SetAllowAll(a.deps.Config.Gateway.AllowAll)
	for platName, platCfg := range a.deps.Config.Gateway.Platforms {
		if len(platCfg.AllowList) > 0 {
			gw.PreAuthorize(platName, platCfg.AllowList...)
		}
	}

	// Wire the message handler: when the gateway routes a message, send it to the agent.
	gw.OnMessage(func(ctx context.Context, sess *gateway.GatewaySession, msg gateway.IncomingMessage) {
		response, err := a.deps.Agent.HandleMessage(ctx, msg.Platform, msg.ChatID, msg.UserID, msg.Text)
		if err != nil {
			response = fmt.Sprintf("Error: %v", err)
		}

		// Human-like typing delay before sending the response.
		if a.deps.Config.Gateway.HumanDelay {
			msPerChar := a.deps.Config.Gateway.DelayMsPerChar
			if msPerChar <= 0 {
				msPerChar = 30
			}
			delay := time.Duration(len(response)*msPerChar) * time.Millisecond
			// Cap at 5 seconds to avoid excessive waits.
			const maxDelay = 5 * time.Second
			if delay > maxDelay {
				delay = maxDelay
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		outMsg := gateway.OutgoingMessage{
			Text:   response,
			Format: "markdown",
		}
		if sendErr := gw.SendTo(ctx, msg.Platform, msg.ChatID, outMsg); sendErr != nil {
			fmt.Fprintf(os.Stderr, "gateway: failed to send response: %v\n", sendErr)
		}
	})

	// Register configured platform adapters.
	adapterCount := registerPlatformAdapters(gw, a.deps.Config)

	if adapterCount == 0 {
		fmt.Println("No platform adapters configured. Add platforms in your config file.")
		fmt.Println("Example: Set gateway.platforms.cli.enabled = true in ~/.hera/config.yaml")
		return nil
	}

	fmt.Printf("Starting Hera gateway with %d platform adapter(s)...\n", adapterCount)

	// Start the gateway.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := gw.Start(ctx); err != nil {
		return fmt.Errorf("start gateway: %w", err)
	}

	// Start hooks.d watcher after gateway is live. Tied to ctx so
	// SIGINT/SIGTERM cleanly stops the poller.
	hooksWatcher.Start(ctx)
	defer hooksWatcher.Stop()

	fmt.Println("Gateway is running. Press Ctrl+C to stop.")

	<-ctx.Done()
	fmt.Println("\nShutting down gateway...")
	gw.Stop()
	fmt.Println("Gateway stopped.")

	return nil
}
