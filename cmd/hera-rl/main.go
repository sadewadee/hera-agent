// Command hera-rl is a dedicated CLI runner for RL training workflows.
//
// It provides extended timeouts for long-running training, RL-focused
// system prompts, and the full RL toolset. Usage:
//
//	hera-rl "Train a model on GSM8k for math reasoning"
//	hera-rl --interactive
//	hera-rl --check-server
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/contextengine"
	"github.com/sadewadee/hera/internal/hcore"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/internal/tools/builtin"
	"github.com/sadewadee/hera/plugins"
)

const (
	defaultModel         = "anthropic/claude-opus-4.5"
	defaultMaxIterations = 200
)

var rlToolsets = []string{"terminal", "web", "rl"}

const rlSystemPrompt = `You are an automated post-training engineer specialising in reinforcement learning for language models.

## Your Capabilities

You have access to RL training tools for running reinforcement learning on models:

1. RECORD: Use rl_training with action=start_episode to begin an episode, record_reward to log rewards, end_episode when done
2. EXPORT: Use rl_training with action=export to export accumulated training data
3. INSPECT: Read environment files and training logs to understand current state
4. CONFIGURE: Edit config files to adjust hyperparameters before starting a run
5. MONITOR: Use terminal tools to watch running training processes and log files

## Important Guidelines

- Always test before training: Training runs take hours - verify everything works first
- Monitor metrics: Check for reward/mean and percent_correct
- Status check intervals: Wait at least 30 minutes between status checks
- Early stopping: Stop training early if metrics look bad or stagnant
- Iterate quickly: Start with small total_steps to validate, then scale up`

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera-rl: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse flags.
	var (
		model         string
		apiKey        string
		baseURL       string
		maxIterations int
		interactive   bool
		checkServer   bool
		verbose       bool
	)
	flag.StringVar(&model, "model", "", "Model to use (reads from config if not provided)")
	flag.StringVar(&apiKey, "api-key", "", "API key (uses OPENROUTER_API_KEY if not provided)")
	flag.StringVar(&baseURL, "base-url", "", "API base URL (reads from config if not provided)")
	flag.IntVar(&maxIterations, "max-iterations", defaultMaxIterations, "Maximum agent iterations")
	flag.BoolVar(&interactive, "interactive", false, "Run in interactive mode")
	flag.BoolVar(&checkServer, "check-server", false, "Check if RL setup is ready and exit")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	task := strings.Join(flag.Args(), " ")

	// Set up logging.
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("failed to load config, using defaults", "error", err)
		cfg = &config.Config{}
	}

	// Resolve model.
	if model == "" {
		model = cfg.Agent.DefaultModel
	}
	if model == "" {
		model = defaultModel
	}

	// Resolve API key.
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	fmt.Fprintln(os.Stderr, "RL Training Agent")
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))

	// Handle check-server mode.
	if checkServer {
		return checkSetup()
	}

	// Validate requirements.
	if err := checkRequirements(apiKey); err != nil {
		return err
	}

	if task == "" && !interactive {
		fmt.Fprintln(os.Stderr, "No task provided. Use --interactive or provide a task argument.")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  hera-rl \"Train a model on GSM8k math problems\"")
		fmt.Fprintln(os.Stderr, "  hera-rl --interactive")
		return nil
	}

	// Initialise memory.
	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = filepath.Join(config.HeraDir(), "hera.db")
	}
	// Initialise memory using the plugin-aware factory so users who configure
	// memory.provider: mem0 (or any plugin provider) get their chosen backend.
	pluginReg := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(pluginReg)
	memResult, memErr := memory.NewFromConfig(cfg.Memory, pluginReg, dbPath)
	if memErr != nil {
		return fmt.Errorf("initialise memory: %w", memErr)
	}
	memManager := memory.NewManager(memResult.Primary, nil)

	// Initialise LLM provider with all 12 built-in providers.
	llmRegistry := llm.NewRegistry()
	llm.RegisterAll(llmRegistry)

	providerName := cfg.Agent.DefaultProvider
	if providerName == "" {
		providerName = "openrouter"
	}
	provider, ok := llmRegistry.Get(providerName)
	if !ok {
		provider, _ = llmRegistry.Get("openrouter")
	}

	// Register built-in context engines and resolve active one from config. Skip
	// when no LLM provider is available (hera-rl falls back to the legacy path
	// in that case).
	var contextEngine plugins.ContextEngine
	if provider != nil {
		contextengine.RegisterBuiltinEngines(pluginReg, agent.NewLLMSummarizer(provider))
		ce, ceErr := contextengine.NewFromConfig(cfg.Agent, provider.ModelInfo(), pluginReg)
		if ceErr != nil {
			slog.Warn("context engine init failed", "error", ceErr)
		} else {
			contextEngine = ce
		}
	}

	// Initialise tools.
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, builtin.ToolDeps{
		Config:    cfg,
		SessionDB: builtin.SessionDBFromManager(memManager),
		Version:   hcore.Version,
	})
	if contextEngine != nil {
		builtin.RegisterEngineTools(toolRegistry, contextEngine)
	}
	// Memory-provider tool harvest is intentionally skipped here: hera-rl
	// doesn't construct a SessionManager so there is no session lifecycle
	// event to fire Sidecar.Initialize on, which means the harvested tools
	// would run against an uninitialised sidecar. Use hera/hera-agent for
	// chat workflows that need memory-plugin tools.
	builtin.RegisterCustomToolTool(toolRegistry, builtin.DefaultCustomToolsDir(config.HeraDir()))

	// Create agent.
	ag, err := agent.NewAgent(agent.AgentDeps{
		LLM:           provider,
		Tools:         toolRegistry,
		Memory:        memManager,
		Config:        cfg,
		ContextEngine: contextEngine,
	})
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	// Context with signal handling.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Model: %s\n", model)
	fmt.Fprintf(os.Stderr, "Max iterations: %d\n", maxIterations)
	fmt.Fprintf(os.Stderr, "Toolsets: %s\n", strings.Join(rlToolsets, ", "))
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))

	if interactive {
		return runInteractive(ctx, ag)
	}

	// Single task mode.
	fmt.Fprintf(os.Stderr, "Task: %s\n", task)
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 40))

	response, err := ag.HandleMessage(ctx, "rl-cli", "rl", "rl-user", task)
	if err != nil {
		return fmt.Errorf("task failed: %w", err)
	}

	fmt.Println(response)
	fmt.Fprintln(os.Stderr, "Task completed")
	return nil
}

func runInteractive(ctx context.Context, ag *agent.Agent) error {
	fmt.Fprintln(os.Stderr, "Interactive RL Training Mode")
	fmt.Fprintln(os.Stderr, "Type 'quit' or 'exit' to end the session.")
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 40))

	scanner := newLineScanner()
	for {
		fmt.Fprint(os.Stderr, "\nRL Task> ")
		line, ok := scanner.Scan()
		if !ok {
			break
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" || input == "q" {
			fmt.Fprintln(os.Stderr, "Goodbye!")
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		response, err := ag.HandleMessage(ctx, "rl-cli", "rl", "rl-user", input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Println(response)
	}
	return nil
}

func checkSetup() error {
	heraDir := config.HeraDir()
	fmt.Fprintf(os.Stderr, "Hera home: %s\n", heraDir)

	configPath := filepath.Join(heraDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintln(os.Stderr, "Config: found")
	} else {
		fmt.Fprintln(os.Stderr, "Config: not found")
	}

	if os.Getenv("OPENROUTER_API_KEY") != "" {
		fmt.Fprintln(os.Stderr, "OPENROUTER_API_KEY: set")
	} else {
		fmt.Fprintln(os.Stderr, "OPENROUTER_API_KEY: NOT SET")
	}
	return nil
}

func checkRequirements(apiKey string) error {
	var missing []string
	if apiKey == "" {
		missing = append(missing, "OPENROUTER_API_KEY")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing requirements: %s", strings.Join(missing, ", "))
	}
	return nil
}

// lineScanner wraps os.Stdin for interactive line reading.
type lineScanner struct {
	buf []byte
}

func newLineScanner() *lineScanner { return &lineScanner{} }

func (s *lineScanner) Scan() (string, bool) {
	s.buf = s.buf[:0]
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return string(s.buf), true
			}
			s.buf = append(s.buf, buf[0])
		}
		if err != nil {
			if len(s.buf) > 0 {
				return string(s.buf), true
			}
			return "", false
		}
	}
}
