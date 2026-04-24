package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/cli"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/contextengine"
	"github.com/sadewadee/hera/internal/cron"
	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/hcore"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/mcp"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/paths"
	internalplugins "github.com/sadewadee/hera/internal/plugins"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/syncer"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/internal/tools/builtin"
	"github.com/sadewadee/hera/plugins"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load config: %v (using defaults)", err)
		cfg = &config.Config{}
	}

	// Load enabled third-party plugins from $HERA_HOME/plugins/. Must happen
	// before skills/tools/MCP wiring so plugin-provided content is included.
	pluginResult := internalplugins.LoadEnabled(paths.UserPlugins())
	if len(pluginResult.MCPEntries) > 0 {
		cfg.MCPServers = append(cfg.MCPServers, pluginResult.MCPEntries...)
		log.Printf("info: plugins added %d MCP server(s)", len(pluginResult.MCPEntries))
	}

	// Set up structured logging: stderr + rotated file at ~/.hera/logs/agent.log.
	level := os.Getenv("HERA_LOG_LEVEL")
	if level == "" {
		level = "info"
	}
	_, closeLogs := hcore.SetupLogging(hcore.LogConfig{
		Level:  level,
		LogDir: config.HeraDir() + "/logs",
	})
	defer closeLogs()

	// Initialize memory. Always creates a SQLite primary store; optionally
	// adds a cloud sidecar when cfg.Memory.Provider names one of the 8
	// plugin providers.
	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = config.HeraDir() + "/hera.db"
	}
	pluginRegistry := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(pluginRegistry)
	memResult, err := memory.NewFromConfig(cfg.Memory, pluginRegistry, dbPath)
	var memProvider memory.Provider
	if err != nil {
		log.Printf("warning: could not initialize memory: %v", err)
	} else {
		memProvider = memResult.Primary
		if memResult.Sidecar != nil {
			log.Printf("info: memory sidecar enabled: %s", cfg.Memory.Provider)
		}
	}

	// Initialize tool registry.
	toolRegistry := tools.NewRegistry()

	// Sync bundled skills into $HERA_HOME/skills/ before loading.
	// Idempotent: user-modified skills are preserved, new bundled skills
	// are seeded, unchanged bundled skills are updated on upgrade.
	skillsSyncer := syncer.New(paths.BundledSkills(), paths.UserSkills())
	if syncStats, syncErr := skillsSyncer.Sync(); syncErr != nil {
		log.Printf("warning: skills sync failed: %v", syncErr)
	} else {
		log.Printf("info: skills sync — %d copied, %d preserved, %d skipped",
			syncStats.Copied, syncStats.Preserved, syncStats.Skipped)
	}

	// Load skills from $HERA_HOME/skills/ plus any plugin-provided skill dirs.
	skillDirs := append([]string{paths.UserSkills()}, pluginResult.SkillDirs...)
	skillLoader := skills.NewLoader(skillDirs...)
	if err := skillLoader.LoadAll(); err != nil {
		log.Printf("warning: could not load skills: %v", err)
	}

	// Initialize LLM provider registry with all 12 built-in providers.
	llmRegistry := llm.NewRegistry()
	llm.RegisterAll(llmRegistry)

	// Create the LLM provider chain (primary + fallbacks). Only warn on
	// failure — hera CLI can still run `setup` and `doctor` without an LLM.
	var llmProvider llm.Provider
	var memManager *memory.Manager

	if cfg.Agent.DefaultProvider == "" {
		cfg.Agent.DefaultProvider = "openai"
	}
	llmProvider, err = hcore.BuildLLMProvider(cfg, llmRegistry)
	if err != nil {
		log.Printf("warning: could not create LLM provider: %v", err)
	}

	// Create memory manager (needs LLM for summarization).
	if memProvider != nil {
		var summarizer memory.Summarizer
		if llmProvider != nil {
			summarizer = &llmMemorySummarizer{llm: llmProvider}
		}
		memManager = memory.NewManager(memProvider, summarizer)
	}

	// Register built-in context engines and resolve the active one from config.
	// Requires llmProvider for the summarizer used by the default "compressor"
	// engine; with no LLM we skip engine wiring and let agent fall back to no
	// compression.
	var contextEngine plugins.ContextEngine
	if llmProvider != nil {
		contextengine.RegisterBuiltinEngines(pluginRegistry, agent.NewLLMSummarizer(llmProvider))
		ce, cerr := contextengine.NewFromConfig(cfg.Agent, llmProvider.ModelInfo(), pluginRegistry)
		if cerr != nil {
			log.Printf("warning: context engine: %v", cerr)
		} else {
			contextEngine = ce
		}
	}

	// Wire the LLM-powered skill generator. It produces skill bodies from a
	// description when skill_create is called without a content field.
	var skillGen builtin.SkillGenerator
	if llmProvider != nil {
		skillGen = skills.NewGenerator(llmProvider, paths.UserSkills())
	}

	// Initialize cron scheduler if enabled.
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

	// Register built-in tools (now with memory manager, skill generator, and cron).
	builtin.RegisterAll(toolRegistry, builtin.ToolDeps{
		Config:         cfg,
		MemoryManager:  memManager,
		SkillGenerator: skillGen,
		CronScheduler:  cronScheduler,
		SessionDB:      builtin.SessionDBFromManager(memManager),
		Version:        hcore.Version,
	})

	// Harvest engine-exposed tools (no-op for built-in compressor engine).
	if contextEngine != nil {
		builtin.RegisterEngineTools(toolRegistry, contextEngine)
	}

	// Harvest tools from the active memory-provider sidecar (hindsight_recall,
	// mem0_search, etc.). Built-in sqlite has no sidecar so this is a no-op
	// unless the user selected an external provider via memory.provider.
	if memResult.Sidecar != nil {
		builtin.RegisterMemoryProviderTools(toolRegistry, memResult.Sidecar)
	}

	// Register user-defined custom tools from config.
	if len(cfg.CustomTools) > 0 {
		builtin.RegisterCustomTools(toolRegistry, cfg.CustomTools)
		log.Printf("info: loaded %d custom tools from config", len(cfg.CustomTools))
	}

	// Hot-reloadable custom tools from ~/.hera/tools.d/ (register_custom_tool
	// writes new YAML here so tools survive without a restart).
	customToolsDir := builtin.DefaultCustomToolsDir(config.HeraDir())
	builtin.RegisterCustomToolTool(toolRegistry, customToolsDir)

	// Connect to MCP servers and register their tools. Each server is
	// wrapped in a ManagedClient so on_demand mode (default) kills
	// the subprocess after idle and respawns on next call.
	var mcpClients []*mcp.ManagedClient
	if len(cfg.MCPServers) > 0 {
		mcpConfigs := make([]mcp.MCPServerConfig, len(cfg.MCPServers))
		for i, s := range cfg.MCPServers {
			mcpConfigs[i] = mcp.MCPServerConfig{
				Name:        s.Name,
				Command:     s.Command,
				Args:        s.Args,
				Env:         s.Env,
				Mode:        s.Mode,
				IdleTimeout: parseDurationOrZero(s.IdleTimeout),
			}
		}
		mcpClients = builtin.RegisterMCPTools(toolRegistry, mcpConfigs)
		defer func() {
			for _, c := range mcpClients {
				c.Close()
			}
		}()
	}

	// Create the agent if we have an LLM provider.
	var agentInstance *agent.Agent
	if llmProvider != nil {
		sessionMgr := agent.NewSessionManager(30 * time.Minute)
		agentInstance, err = agent.NewAgent(agent.AgentDeps{
			LLM:           llmProvider,
			Tools:         toolRegistry,
			Memory:        memManager,
			Skills:        skillLoader,
			Sessions:      sessionMgr,
			Config:        cfg,
			ContextEngine: contextEngine,
			MemorySidecar: memResult.Sidecar,
		})
		if err != nil {
			log.Printf("warning: could not create agent: %v", err)
		}

		if agentInstance != nil {
			// Wire delegate_task tool + AgentBus so agent-to-agent delegation is
			// reachable from LLM tool calls.
			agentRegistry := agent.NewAgentRegistry()
			agentRegistry.Register("main", agentInstance)
			agentBus := gateway.NewAgentBus()
			dt := builtin.NewDelegateTaskTool(agentRegistry).WithCallerName("main").WithObserver(agentBus)
			toolRegistry.Register(dt)
			log.Printf("hera: delegate_task tool wired (AgentBus active)")
		}
	} else {
		log.Printf("info: no LLM provider configured (run 'hera setup' or set API key env var)")
	}

	// Start cron scheduler if wired; stop when CLI exits.
	if cronScheduler != nil {
		cronScheduler.Start(context.Background())
		defer cronScheduler.Stop()
		log.Println("hera: cron scheduler started")
	}

	// Run the CLI application with all dependencies wired.
	app := cli.NewApp(cli.AppDeps{
		Config:        cfg,
		Agent:         agentInstance,
		LLMRegistry:   llmRegistry,
		ToolRegistry:  toolRegistry,
		SkillLoader:   skillLoader,
		Memory:        memManager,
		CronScheduler: cronScheduler,
	})
	return app.Run()
}

// llmMemorySummarizer wraps an LLM provider to implement the memory.Summarizer interface.
type llmMemorySummarizer struct {
	llm llm.Provider
}

func (s *llmMemorySummarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
	prompt := llm.Message{
		Role:    llm.RoleSystem,
		Content: "Summarize the following conversation concisely, preserving key facts and context. Output only the summary.",
	}
	req := llm.ChatRequest{
		Messages: append([]llm.Message{prompt}, messages...),
	}
	resp, err := s.llm.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("summarize: %w", err)
	}
	return resp.Message.Content, nil
}

// parseDurationOrZero parses a Go duration string ("5m", "30s", etc.)
// and returns 0 on empty/invalid input. Callers treat 0 as "use the
// layer's default" — that's how MCP lifecycle defaults propagate.
func parseDurationOrZero(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
