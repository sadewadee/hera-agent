package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/contextengine"
	"github.com/sadewadee/hera/internal/cron"
	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/gateway/platforms"
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

// heraVersion is the "v"-prefixed form of hcore.Version used by the health
// endpoint JSON response (e.g. "v0.13.2"). Keep in sync by deriving, not
// hardcoding.
var heraVersion = "v" + hcore.Version

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera-agent: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Load enabled third-party plugins from $HERA_HOME/plugins/. Must happen
	// before skills/hooks/tools/MCP wiring so plugin-provided content is
	// included from the start. MCP entries are appended to cfg.MCPServers so
	// the existing MCP wiring block picks them up automatically.
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
	// plugin providers. A startup health check surfaces silent
	// misconfiguration (missing tables, wrong path, unwritable file)
	// in the first log lines instead of as "bot forgets everything" later.
	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = config.HeraDir() + "/hera.db"
	}
	pluginRegistry := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(pluginRegistry)
	memResult, err := memory.NewFromConfig(cfg.Memory, pluginRegistry, dbPath)
	if err != nil {
		return fmt.Errorf("initialize memory: %w", err)
	}
	memProvider := memResult.Primary
	if sqliteP, ok := memProvider.(*memory.SQLiteProvider); ok {
		memory.LogHealthReport(sqliteP.HealthCheck(context.Background(), dbPath))
	}
	if memResult.Sidecar != nil {
		log.Printf("info: memory sidecar enabled: %s", cfg.Memory.Provider)
	}

	// Initialize LLM provider registry with all 12 built-in providers.
	llmRegistry := llm.NewRegistry()
	llm.RegisterAll(llmRegistry)

	// Build the LLM provider chain: primary + fallbacks wrapped for
	// automatic retry on provider-wide outages.
	providerName := cfg.Agent.DefaultProvider
	if providerName == "" {
		providerName = "openai"
		cfg.Agent.DefaultProvider = "openai"
	}
	llmProvider, err := hcore.BuildLLMProvider(cfg, llmRegistry)
	if err != nil {
		return fmt.Errorf("create LLM provider: %w", err)
	}

	// Create memory manager.
	var summarizer memory.Summarizer
	summarizer = &llmMemorySummarizer{llm: llmProvider}
	memManager := memory.NewManager(memProvider, summarizer)

	// Register built-in context engines and resolve the active one from config.
	contextengine.RegisterBuiltinEngines(pluginRegistry, agent.NewLLMSummarizer(llmProvider))
	contextEngine, ceErr := contextengine.NewFromConfig(cfg.Agent, llmProvider.ModelInfo(), pluginRegistry)
	if ceErr != nil {
		return fmt.Errorf("initialize context engine: %w", ceErr)
	}

	// Wire the LLM-powered skill generator.
	skillGen := skills.NewGenerator(llmProvider, paths.UserSkills())

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

	// Initialize tool registry.
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, builtin.ToolDeps{
		Config:         cfg,
		MemoryManager:  memManager,
		SkillGenerator: skillGen,
		CronScheduler:  cronScheduler,
		SessionDB:      builtin.SessionDBFromManager(memManager),
		Version:        hcore.Version,
	})

	// Harvest engine-exposed tools (no-op for the built-in compressor engine;
	// third-party engines such as LCM may expose context-manipulation tools).
	builtin.RegisterEngineTools(toolRegistry, contextEngine)

	// Harvest memory-provider tools (hindsight_recall, mem0_search, etc.) so
	// plugin providers' tool schemas actually reach the LLM. Nil for sqlite.
	if memResult.Sidecar != nil {
		builtin.RegisterMemoryProviderTools(toolRegistry, memResult.Sidecar)
	}

	// User-defined custom tools from config.yaml are loaded up front.
	if len(cfg.CustomTools) > 0 {
		builtin.RegisterCustomTools(toolRegistry, cfg.CustomTools)
		log.Printf("info: loaded %d custom tools from config", len(cfg.CustomTools))
	}

	// Hot-reloadable custom tools: watch ~/.hera/tools.d/ for YAML
	// definitions so the agent can add tools at runtime via
	// register_custom_tool without needing a restart.
	customToolsDir := builtin.DefaultCustomToolsDir(config.HeraDir())
	builtin.RegisterCustomToolTool(toolRegistry, customToolsDir)
	customWatcher := builtin.NewCustomToolWatcher(toolRegistry, customToolsDir)

	// Connect to MCP servers (Python sidecar, etc.) wrapped in a
	// ManagedClient so on_demand mode (default) keeps idle
	// subprocesses cheap.
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

	// Sync bundled skills into $HERA_HOME/skills/ before loading.
	// This is idempotent: user-modified skills are preserved, new bundled
	// skills are seeded, unchanged bundled skills are updated on upgrade.
	skillsSyncer := syncer.New(paths.BundledSkills(), paths.UserSkills())
	if syncStats, syncErr := skillsSyncer.Sync(); syncErr != nil {
		log.Printf("warning: skills sync failed: %v", syncErr)
	} else {
		log.Printf("info: skills sync — %d copied, %d preserved, %d skipped",
			syncStats.Copied, syncStats.Preserved, syncStats.Skipped)
	}

	// Load skills from $HERA_HOME/skills/ plus any plugin-provided skill dirs.
	// Syncer guarantees $HERA_HOME/skills/ is populated with bundled skills.
	skillDirs := append([]string{paths.UserSkills()}, pluginResult.SkillDirs...)
	skillLoader := skills.NewLoader(skillDirs...)
	if err := skillLoader.LoadAll(); err != nil {
		log.Printf("warning: could not load skills: %v", err)
	}

	// Create the agent.
	sessionMgr := agent.NewSessionManager(30 * time.Minute)
	agentInstance, err := agent.NewAgent(agent.AgentDeps{
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
		return fmt.Errorf("create agent: %w", err)
	}

	// Wire delegate_task tool + AgentBus so agent-to-agent delegation is
	// reachable from LLM tool calls. The registry holds a reference to
	// agentInstance under "main". Additional sub-agents can be registered
	// at runtime via agentRegistry.Register(name, agent).
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.Register("main", agentInstance)
	agentBus := gateway.NewAgentBus()
	dt := builtin.NewDelegateTaskTool(agentRegistry).WithCallerName("main").WithObserver(agentBus)
	toolRegistry.Register(dt)
	log.Printf("hera-agent: delegate_task tool wired (AgentBus active)")

	// Create the gateway.
	gw := gateway.NewGateway(gateway.GatewayOptions{
		SessionTimeout: time.Duration(cfg.Gateway.SessionTimeout) * time.Minute,
	})

	// Wire send_message to the live gateway so the LLM can actually
	// push outgoing messages and file attachments to any registered
	// platform adapter (telegram, discord, slack, etc.).
	builtin.RegisterSendMessage(toolRegistry, gw)

	// Register user-defined custom hooks from config (static, before gateway start).
	if len(cfg.Hooks) > 0 {
		gateway.RegisterCustomHooks(gw.Hooks(), cfg.Hooks)
		log.Printf("info: loaded %d custom hooks from config", len(cfg.Hooks))
	}

	// Hot-reloadable hooks: watch ~/.hera/hooks.d/ for YAML hook files so
	// the user can add/update/remove hooks without restarting the agent.
	hooksDir := paths.UserHooks()
	hooksWatcher := gateway.NewHooksWatcher(gw.Hooks(), hooksDir)

	// Apply authorization settings from config.
	gw.SetAllowAll(cfg.Gateway.AllowAll)
	for platName, platCfg := range cfg.Gateway.Platforms {
		if len(platCfg.AllowList) > 0 {
			gw.PreAuthorize(platName, platCfg.AllowList...)
		}
	}

	// Wire the message handler.
	gw.OnMessage(func(ctx context.Context, sess *gateway.GatewaySession, msg gateway.IncomingMessage) {
		response, err := agentInstance.HandleMessage(ctx, msg.Platform, msg.ChatID, msg.UserID, msg.Text)
		if err != nil {
			response = fmt.Sprintf("Error: %v", err)
		}
		outMsg := gateway.OutgoingMessage{
			Text:   response,
			Format: "markdown",
		}
		if sendErr := gw.SendTo(ctx, msg.Platform, msg.ChatID, outMsg); sendErr != nil {
			log.Printf("gateway: failed to send to %s/%s: %v", msg.Platform, msg.ChatID, sendErr)
		}
	})

	// Register configured platform adapters.
	adapterCount := registerAdapters(gw, cfg)
	if adapterCount == 0 {
		log.Println("warning: no platform adapters configured, adding CLI adapter")
		gw.AddAdapter(platforms.NewCLIAdapter())
		adapterCount = 1
	}

	log.Printf("hera-agent: starting with %d platform adapter(s)", adapterCount)
	log.Printf("hera-agent: provider=%s model=%s", providerName, cfg.Agent.DefaultModel)
	log.Printf("hera-agent: %d tools, %d skills loaded", len(toolRegistry.List()), len(skillLoader.All()))

	// Start the gateway.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start cron scheduler if wired.
	if cronScheduler != nil {
		cronScheduler.Start(ctx)
		defer cronScheduler.Stop()
		log.Println("hera-agent: cron scheduler started")
	}

	// Start watching tools.d/ for hot-reload custom tools. Tied to ctx
	// so SIGINT/SIGTERM cleanly stops the poller.
	customWatcher.Start(ctx)
	defer customWatcher.Stop()

	// Start watching hooks.d/ for hot-reload hook files.
	hooksWatcher.Start(ctx)
	defer hooksWatcher.Stop()

	// Start watchers for plugin-provided hooks.d/ and tools.d/ directories.
	for _, hookDir := range pluginResult.HookDirs {
		pw := gateway.NewHooksWatcher(gw.Hooks(), hookDir)
		pw.Start(ctx)
		defer pw.Stop()
	}
	for _, toolDir := range pluginResult.ToolDirs {
		ptw := builtin.NewCustomToolWatcher(toolRegistry, toolDir)
		ptw.Start(ctx)
		defer ptw.Stop()
	}

	if err := gw.Start(ctx); err != nil {
		return fmt.Errorf("start gateway: %w", err)
	}

	// Start unconditional /health endpoint.
	// Listens on HERA_HEALTH_PORT (default 8080) and answers with JSON
	// {"status":"ok","version":heraVersion} regardless of which gateway
	// adapters are configured. Required by the Docker healthcheck.
	healthAddr := os.Getenv("HERA_HEALTH_PORT")
	if healthAddr == "" {
		healthAddr = "8080"
	}
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":%q}`, heraVersion)
	})
	healthServer := &http.Server{
		Addr:    ":" + healthAddr,
		Handler: healthMux,
	}
	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("hera-agent: health server error: %v", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = healthServer.Shutdown(shutdownCtx)
	}()
	log.Printf("hera-agent: health endpoint on :%s/health", healthAddr)

	log.Println("hera-agent: gateway running, waiting for messages...")

	<-ctx.Done()
	log.Println("hera-agent: shutting down...")
	gw.Stop()
	log.Println("hera-agent: stopped")

	return nil
}

// registerAdapters registers enabled platform adapters from config.
func registerAdapters(gw *gateway.Gateway, cfg *config.Config) int {
	count := 0
	for name, platCfg := range cfg.Gateway.Platforms {
		if !platCfg.Enabled {
			continue
		}
		switch name {
		case "cli":
			gw.AddAdapter(platforms.NewCLIAdapter())
			count++
		case "telegram":
			if platCfg.Token != "" {
				if platCfg.Extra["mode"] == "webhook" {
					gw.AddAdapter(platforms.NewTelegramAdapterWithOptions(platCfg.Token, platforms.TelegramOptions{
						Mode:        "webhook",
						WebhookURL:  platCfg.Extra["webhook_url"],
						WebhookAddr: platCfg.Extra["webhook_addr"],
						WebhookPath: platCfg.Extra["webhook_path"],
					}))
				} else {
					gw.AddAdapter(platforms.NewTelegramAdapter(platCfg.Token))
				}
				count++
			}
		case "discord":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewDiscordAdapter(platCfg.Token))
				count++
			}
		case "slack":
			appToken := platCfg.Extra["app_token"]
			if platCfg.Token != "" && appToken != "" {
				gw.AddAdapter(platforms.NewSlackAdapter(platCfg.Token, appToken))
				count++
			}
		case "apiserver":
			addr := platCfg.Extra["addr"]
			gw.AddAdapter(platforms.NewAPIServerAdapter(addr))
			count++
		case "webhook":
			gw.AddAdapter(platforms.NewWebhookAdapter(platforms.WebhookConfig{
				Addr:        platCfg.Extra["addr"],
				Secret:      platCfg.Extra["secret"],
				CallbackURL: platCfg.Extra["callback_url"],
			}))
			count++
		case "threads":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewThreadsAdapter(platCfg.Token))
				count++
			}
		case "whatsapp":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewWhatsAppAdapter(platforms.WhatsAppConfig{
					PhoneNumberID: platCfg.Extra["phone_number_id"],
					AccessToken:   platCfg.Token,
					VerifyToken:   platCfg.Extra["verify_token"],
					CallbackAddr:  platCfg.Extra["callback_addr"],
				}))
				count++
			}
		case "signal":
			if platCfg.Extra["api_url"] != "" {
				gw.AddAdapter(platforms.NewSignalAdapter(platforms.SignalConfig{
					APIURL:      platCfg.Extra["api_url"],
					PhoneNumber: platCfg.Extra["phone_number"],
				}))
				count++
			}
		case "matrix":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewMatrixAdapter(platforms.MatrixConfig{
					HomeserverURL: platCfg.Extra["homeserver_url"],
					UserID:        platCfg.Extra["user_id"],
					AccessToken:   platCfg.Token,
				}))
				count++
			}
		case "email":
			if platCfg.Extra["smtp_host"] != "" {
				gw.AddAdapter(platforms.NewEmailAdapter(platforms.EmailConfig{
					SMTPHost:     platCfg.Extra["smtp_host"],
					SMTPPort:     platCfg.Extra["smtp_port"],
					SMTPUser:     platCfg.Extra["smtp_user"],
					SMTPPassword: platCfg.Extra["smtp_password"],
					FromAddress:  platCfg.Extra["from_address"],
					WebhookAddr:  platCfg.Extra["webhook_addr"],
				}))
				count++
			}
		case "sms":
			if platCfg.Extra["account_sid"] != "" {
				gw.AddAdapter(platforms.NewSMSAdapter(platforms.SMSConfig{
					AccountSID:  platCfg.Extra["account_sid"],
					AuthToken:   platCfg.Token,
					FromNumber:  platCfg.Extra["from_number"],
					WebhookAddr: platCfg.Extra["webhook_addr"],
				}))
				count++
			}
		case "homeassistant":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewHomeAssistantAdapter(platforms.HomeAssistantConfig{
					HAURL:       platCfg.Extra["ha_url"],
					Token:       platCfg.Token,
					WebhookAddr: platCfg.Extra["webhook_addr"],
				}))
				count++
			}
		case "dingtalk":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewDingTalkAdapter(platforms.DingTalkConfig{
					AccessToken:  platCfg.Token,
					Secret:       platCfg.Extra["secret"],
					CallbackAddr: platCfg.Extra["callback_addr"],
				}))
				count++
			}
		case "feishu":
			if platCfg.Extra["app_id"] != "" {
				gw.AddAdapter(platforms.NewFeishuAdapter(platforms.FeishuConfig{
					AppID:             platCfg.Extra["app_id"],
					AppSecret:         platCfg.Extra["app_secret"],
					VerificationToken: platCfg.Extra["verification_token"],
					CallbackAddr:      platCfg.Extra["callback_addr"],
				}))
				count++
			}
		case "wecom":
			if platCfg.Extra["corp_id"] != "" {
				gw.AddAdapter(platforms.NewWeComAdapter(platforms.WeComConfig{
					CorpID:       platCfg.Extra["corp_id"],
					CorpSecret:   platCfg.Extra["corp_secret"],
					VerifyToken:  platCfg.Extra["verify_token"],
					CallbackAddr: platCfg.Extra["callback_addr"],
				}))
				count++
			}
		case "mattermost":
			if platCfg.Token != "" {
				gw.AddAdapter(platforms.NewMattermostAdapter(platforms.MattermostConfig{
					ServerURL: platCfg.Extra["server_url"],
					Token:     platCfg.Token,
				}))
				count++
			}
		case "bluebubbles":
			if platCfg.Extra["api_url"] != "" {
				gw.AddAdapter(platforms.NewBlueBubblesAdapter(platforms.BlueBubblesConfig{
					APIURL:   platCfg.Extra["api_url"],
					Password: platCfg.Extra["password"],
				}))
				count++
			}
		}
	}
	return count
}

// llmMemorySummarizer wraps an LLM provider to implement memory.Summarizer.
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

// parseDurationOrZero parses a Go duration string ("5m", "30s") and
// returns 0 on empty/invalid input. Callers treat 0 as "use the
// layer's default" so MCP lifecycle defaults propagate cleanly.
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
