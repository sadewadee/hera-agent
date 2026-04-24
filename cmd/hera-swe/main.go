// Command hera-swe is the software-engineering agent for Hera.
//
// It accepts a natural-language task description, spins up a curated
// agent with file/shell/git/patch tools, and runs a TDD loop to implement
// the task autonomously. Changes are committed to a local git branch.
// No git push is ever issued.
//
// Usage:
//
//	hera-swe -task "fix the null pointer at foo.go:42" \
//	         -test-cmd "go test ./internal/foo/..." \
//	         -branch fix/npe
//
//	hera-swe -task "$(cat issue.md)" -max-iterations 5 -base main
//
//	# Create a PR after work is committed to a branch:
//	hera-swe -task "fix auth bug" -branch fix/auth -pr
//
// Non-goals for v0.12.2:
//   - Human-in-loop approval gates
//   - Cost tracking
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/contextengine"
	"github.com/sadewadee/hera/internal/hcore"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/swe"
	"github.com/sadewadee/hera/internal/tools/builtin"
	"github.com/sadewadee/hera/plugins"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera-swe: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		taskFlag     = flag.String("task", "", "Task description (required; or leave blank to read from stdin)")
		testCmdFlag  = flag.String("test-cmd", "go test ./...", "Shell command to run after each iteration")
		branchFlag   = flag.String("branch", "", "Git branch to create/checkout for changes")
		baseFlag     = flag.String("base", "main", "Base branch to branch from")
		maxIterFlag  = flag.Int("max-iterations", 10, "Maximum number of agent iterations")
		dryRunFlag   = flag.Bool("dry-run", false, "Print proposed actions without applying")
		prFlag       = flag.Bool("pr", false, "Create a pull request via gh CLI after tests pass")
		repoPathFlag = flag.String("repo", "", "Path to git repository root (defaults to current directory)")
	)
	flag.Parse()

	// Resolve task: flag → positional arg → stdin.
	task := strings.TrimSpace(*taskFlag)
	if task == "" && flag.NArg() > 0 {
		task = strings.Join(flag.Args(), " ")
	}
	if task == "" {
		// Try stdin if it's piped.
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice == 0 {
			var sb strings.Builder
			buf := make([]byte, 4096)
			for {
				n, rdErr := os.Stdin.Read(buf)
				if n > 0 {
					sb.Write(buf[:n])
				}
				if rdErr != nil {
					break
				}
			}
			task = strings.TrimSpace(sb.String())
		}
	}
	if task == "" {
		flag.Usage()
		return fmt.Errorf("task description is required (-task flag or stdin)")
	}

	// Resolve repo path.
	repo := strings.TrimSpace(*repoPathFlag)
	if repo == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("resolve working directory: %w", cwdErr)
		}
		repo = cwd
	}
	var absErr error
	repo, absErr = filepath.Abs(repo)
	if absErr != nil {
		return fmt.Errorf("resolve repo path: %w", absErr)
	}

	// Load configuration.
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		log.Printf("warning: could not load config: %v (using defaults)", cfgErr)
		cfg = &config.Config{}
	}

	// Set up logging.
	_, closeLogs := hcore.SetupLogging(hcore.LogConfig{
		Level:  envOrDefault("HERA_LOG_LEVEL", "info"),
		LogDir: config.HeraDir() + "/logs",
	})
	defer closeLogs()

	// Initialize memory (optional for SWE mode). initMemory returns the plugin
	// registry so the context-engine bootstrap below reuses it.
	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = config.HeraDir() + "/hera.db"
	}
	memManager, pluginReg := initMemory(cfg, dbPath)

	// Initialize LLM.
	if cfg.Agent.DefaultProvider == "" {
		cfg.Agent.DefaultProvider = "openai"
	}
	llmRegistry := llm.NewRegistry()
	llm.RegisterAll(llmRegistry)
	llmProvider, llmErr := hcore.BuildLLMProvider(cfg, llmRegistry)
	if llmErr != nil {
		return fmt.Errorf("init LLM provider: %w", llmErr)
	}

	// Register built-in context engines and resolve active one from config.
	contextengine.RegisterBuiltinEngines(pluginReg, agent.NewLLMSummarizer(llmProvider))
	contextEngine, ceErr := contextengine.NewFromConfig(cfg.Agent, llmProvider.ModelInfo(), pluginReg)
	if ceErr != nil {
		return fmt.Errorf("initialize context engine: %w", ceErr)
	}

	// Build the curated SWE tool registry — only the 6 tools needed.
	toolReg := swe.BuildToolset(repo, false)
	builtin.RegisterEngineTools(toolReg, contextEngine)

	// Build the agent with the SWE toolset.
	sessionMgr := agent.NewSessionManager(30 * time.Minute)
	agentInst, agentErr := agent.NewAgent(agent.AgentDeps{
		LLM:           llmProvider,
		Tools:         toolReg,
		Memory:        memManager,
		Sessions:      sessionMgr,
		Config:        cfg,
		ContextEngine: contextEngine,
	})
	if agentErr != nil {
		return fmt.Errorf("init agent: %w", agentErr)
	}

	// Build and run the SWE engine.
	engineCfg := swe.Config{
		Task:          task,
		RepoPath:      repo,
		Branch:        *branchFlag,
		BaseBranch:    *baseFlag,
		TestCmd:       *testCmdFlag,
		MaxIterations: *maxIterFlag,
		DryRun:        *dryRunFlag,
	}

	gitOps := &swe.ExecGitOperator{}
	tddRunner := swe.NewShellTDDRunner(*testCmdFlag, repo, 0)

	engine, engErr := swe.NewEngine(engineCfg, agentInst, gitOps, tddRunner)
	if engErr != nil {
		return engErr
	}

	// Context with SIGINT/SIGTERM cancellation.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintf(os.Stderr, "hera-swe: task=%q repo=%s branch=%q test=%q iterations=%d dry-run=%v\n",
		task, repo, *branchFlag, *testCmdFlag, *maxIterFlag, *dryRunFlag)

	if runErr := engine.Run(ctx); runErr != nil {
		return runErr
	}

	fmt.Fprintln(os.Stderr, "hera-swe: done — tests passed")

	// Optionally create a pull request via gh CLI.
	if *prFlag {
		diffSum, _ := gitSummary(repo)
		prTitle, prBody := swe.GeneratePRContent(task, diffSum, *maxIterFlag)
		fmt.Fprintf(os.Stderr, "hera-swe: creating PR: %q\n", prTitle)
		if prErr := swe.CreatePR(prTitle, prBody); prErr != nil {
			return fmt.Errorf("create PR: %w", prErr)
		}
		fmt.Fprintln(os.Stderr, "hera-swe: pull request created")
	}
	return nil
}

// gitSummary returns a short `git diff --stat HEAD` summary for PR body generation.
func gitSummary(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "diff", "--stat", "HEAD~1", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// initMemory initialises the memory provider using the plugin-aware factory,
// returning both the memory manager and the plugin registry so callers can
// continue registering additional plugin kinds (e.g. context engines). Memory
// init failure yields a nil manager plus a still-valid registry.
func initMemory(cfg *config.Config, dbPath string) (*memory.Manager, *plugins.Registry) {
	pluginReg := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(pluginReg)
	result, err := memory.NewFromConfig(cfg.Memory, pluginReg, dbPath)
	if err != nil {
		log.Printf("warning: memory init failed: %v", err)
		return nil, pluginReg
	}
	return memory.NewManager(result.Primary, nil), pluginReg
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
