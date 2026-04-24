package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sadewadee/hera/internal/acp"
	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/contextengine"
	"github.com/sadewadee/hera/internal/cron"
	"github.com/sadewadee/hera/internal/hcore"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/internal/tools/builtin"
	"github.com/sadewadee/hera/plugins"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera-acp: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize LLM provider with all 12 built-in providers.
	reg := llm.NewRegistry()
	llm.RegisterAll(reg)

	apiKey := config.ResolveAPIKey(cfg, cfg.Agent.DefaultProvider)
	if apiKey == "" && cfg.Agent.DefaultProvider != "ollama" && cfg.Agent.DefaultProvider != "compatible" {
		return fmt.Errorf("no API key configured")
	}

	provider, err := reg.Create(cfg.Agent.DefaultProvider, llm.ProviderConfig{
		APIKey: apiKey,
		Model:  cfg.Agent.DefaultModel,
	})
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}

	// Initialize memory using the plugin-aware factory so users who configure
	// memory.provider: mem0 (or any plugin provider) get their chosen backend
	// instead of silently falling back to SQLite.
	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = config.HeraDir() + "/hera.db"
	}
	pluginRegistry := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(pluginRegistry)
	memResult, err := memory.NewFromConfig(cfg.Memory, pluginRegistry, dbPath)
	if err != nil {
		return fmt.Errorf("init memory: %w", err)
	}
	defer memResult.Primary.Close()
	memManager := memory.NewManager(memResult.Primary, nil)
	if memResult.Sidecar != nil {
		slog.Info("hera-acp: memory sidecar enabled", "provider", cfg.Memory.Provider)
	}

	// Register built-in context engines and resolve active one from config.
	contextengine.RegisterBuiltinEngines(pluginRegistry, agent.NewLLMSummarizer(provider))
	contextEngine, ceErr := contextengine.NewFromConfig(cfg.Agent, provider.ModelInfo(), pluginRegistry)
	if ceErr != nil {
		return fmt.Errorf("initialize context engine: %w", ceErr)
	}

	// Initialize cron scheduler (hera-acp is an agent-serving binary; cron
	// tasks make sense here so the tool returns real results instead of
	// "cron is disabled").
	var cronScheduler *cron.Scheduler
	if cfg.Cron.Enabled {
		cronDBPath := config.HeraDir() + "/cron.db"
		cs, cronErr := cron.NewScheduler(cronDBPath)
		if cronErr != nil {
			log.Printf("warning: could not start cron scheduler: %v", cronErr)
		} else {
			cronScheduler = cs
		}
	}

	// Initialize tools.
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, builtin.ToolDeps{
		MemoryManager: memManager,
		Config:        cfg,
		SessionDB:     builtin.SessionDBFromManager(memManager),
		CronScheduler: cronScheduler,
		Version:       hcore.Version,
	})
	builtin.RegisterEngineTools(toolRegistry, contextEngine)
	if memResult.Sidecar != nil {
		builtin.RegisterMemoryProviderTools(toolRegistry, memResult.Sidecar)
	}
	builtin.RegisterCustomToolTool(toolRegistry, builtin.DefaultCustomToolsDir(config.HeraDir()))

	// Initialize agent.
	sessionMgr := agent.NewSessionManager(30 * time.Minute)
	skillLoader := skills.NewLoader()

	ag, err := agent.NewAgent(agent.AgentDeps{
		LLM:           provider,
		Tools:         toolRegistry,
		Memory:        memManager,
		Skills:        skillLoader,
		Sessions:      sessionMgr,
		Config:        cfg,
		ContextEngine: contextEngine,
		MemorySidecar: memResult.Sidecar,
	})
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	// Create and start ACP server.
	server := acp.NewServer(acp.ServerConfig{
		Addr:      ":9090",
		JWTSecret: os.Getenv("HERA_ACP_SECRET"),
	}, ag, memManager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cron scheduler tied to context lifetime.
	if cronScheduler != nil {
		cronScheduler.Start(ctx)
		defer cronScheduler.Stop()
		slog.Info("hera-acp: cron scheduler started")
	}

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("start ACP server: %w", err)
	}

	slog.Info("Hera ACP server running", "addr", ":9090")

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down ACP server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return server.Stop(shutdownCtx)
}
