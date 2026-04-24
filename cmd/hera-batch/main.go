// Command hera-batch runs a file of prompts against the configured LLM agent,
// writing results to a file or stdout. Interrupted runs can be resumed via -resume.
//
// Usage:
//
//	hera-batch -input prompts.txt -output results.jsonl -concurrency 4
//	hera-batch -input prompts.txt --estimate
//	hera-batch -input prompts.txt -resume <run-id>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/batch"
	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/contextengine"
	"github.com/sadewadee/hera/internal/hcore"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/syncer"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/internal/tools/builtin"
	"github.com/sadewadee/hera/plugins"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera-batch: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// --- Flags ---
	var (
		inputFile   = flag.String("input", "", "File of prompts (one per line). Reads stdin if omitted.")
		outputFile  = flag.String("output", "", "Output file path. Writes to stdout if omitted.")
		format      = flag.String("format", "jsonl", "Output format: jsonl, csv, text")
		concurrency = flag.Int("concurrency", 1, "Number of concurrent workers")
		maxRetries  = flag.Int("retries", 3, "Max retries on transient errors")
		timeout     = flag.Duration("timeout", 0, "Per-prompt timeout (e.g. 30s). 0 = no timeout.")
		resumeID    = flag.String("resume", "", "Resume a previous run by its run-id")
		runID       = flag.String("run-id", "", "Explicit run identifier (auto-generated if empty)")
		estimate    = flag.Bool("estimate", false, "Estimate token count and cost without running prompts")
		quiet       = flag.Bool("quiet", false, "Suppress progress output")
	)
	flag.Parse()

	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load config: %v (using defaults)", err)
		cfg = &config.Config{}
	}

	// --- LLM ---
	llmRegistry := llm.NewRegistry()
	llm.RegisterAll(llmRegistry)

	if cfg.Agent.DefaultProvider == "" {
		cfg.Agent.DefaultProvider = "openai"
	}
	llmProvider, err := hcore.BuildLLMProvider(cfg, llmRegistry)
	if err != nil {
		return fmt.Errorf("build LLM provider: %w", err)
	}

	// --- Prompt source ---
	var src batch.PromptSource
	if *inputFile != "" {
		src = batch.NewFileSource(*inputFile)
	} else {
		src = batch.NewStdinSource()
	}

	// --- Estimate mode (no agent needed) ---
	if *estimate {
		prompts, err := src.Prompts()
		if err != nil {
			return fmt.Errorf("read prompts: %w", err)
		}
		result := batch.Estimate(prompts, llmProvider)
		fmt.Println(result.String())
		return nil
	}

	// --- Memory ---
	dbPath := cfg.Memory.DBPath
	if dbPath == "" {
		dbPath = config.HeraDir() + "/hera.db"
	}
	pluginRegistry := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(pluginRegistry)

	var memProvider memory.Provider
	var memManager *memory.Manager
	memResult, memErr := memory.NewFromConfig(cfg.Memory, pluginRegistry, dbPath)
	if memErr != nil {
		log.Printf("warning: could not initialize memory: %v", memErr)
	} else {
		memProvider = memResult.Primary
		if memProvider != nil {
			var summarizer memory.Summarizer
			if llmProvider != nil {
				summarizer = &llmSummarizer{llm: llmProvider}
			}
			memManager = memory.NewManager(memProvider, summarizer)
		}
	}

	// --- Skills ---
	skillsSyncer := syncer.New(paths.BundledSkills(), paths.UserSkills())
	if _, syncErr := skillsSyncer.Sync(); syncErr != nil {
		log.Printf("warning: skills sync failed: %v", syncErr)
	}
	skillLoader := skills.NewLoader(paths.UserSkills())
	if err := skillLoader.LoadAll(); err != nil {
		log.Printf("warning: could not load skills: %v", err)
	}

	// --- Context engine ---
	var contextEngine plugins.ContextEngine
	if llmProvider != nil {
		contextengine.RegisterBuiltinEngines(pluginRegistry, agent.NewLLMSummarizer(llmProvider))
		ce, ceErr := contextengine.NewFromConfig(cfg.Agent, llmProvider.ModelInfo(), pluginRegistry)
		if ceErr != nil {
			log.Printf("warning: context engine: %v", ceErr)
		} else {
			contextEngine = ce
		}
	}

	// --- Tools ---
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry, builtin.ToolDeps{
		Config:        cfg,
		MemoryManager: memManager,
		SessionDB:     builtin.SessionDBFromManager(memManager),
		Version:       hcore.Version,
	})
	if contextEngine != nil {
		builtin.RegisterEngineTools(toolRegistry, contextEngine)
	}
	if memResult.Sidecar != nil {
		builtin.RegisterMemoryProviderTools(toolRegistry, memResult.Sidecar)
	}

	// --- Agent ---
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

	// --- Output writer ---
	writer, err := makeWriter(*format, *outputFile)
	if err != nil {
		return err
	}
	defer writer.Close()

	// --- Checkpoint store ---
	var store batch.CheckpointStore
	effectiveRunID := *runID
	if *resumeID != "" {
		effectiveRunID = *resumeID
	}
	if effectiveRunID == "" {
		effectiveRunID = fmt.Sprintf("batch-%d", time.Now().UnixNano())
	}

	cpPath := config.HeraDir() + "/batch-checkpoints/" + effectiveRunID + ".db"
	sqliteStore, storeErr := batch.NewSQLiteCheckpointStore(cpPath)
	if storeErr != nil {
		log.Printf("warning: could not create checkpoint store: %v — resume will not be available", storeErr)
		store = batch.NoopCheckpointStore{}
	} else {
		store = sqliteStore
		defer sqliteStore.Close()
	}

	// --- Progress ---
	var progress batch.ProgressReporter
	if *quiet || *outputFile == "" {
		// When writing to stdout, suppress TTY progress to avoid mixing.
		progress = batch.NoopProgress{}
	} else {
		progress = batch.NewStderrProgress()
	}

	// --- Batch config ---
	batchCfg := batch.Config{
		RunID:         effectiveRunID,
		Concurrency:   *concurrency,
		MaxRetries:    *maxRetries,
		PromptTimeout: *timeout,
	}

	b := batch.New(batchCfg, agentInstance, src, writer, store, progress)

	// --- SIGINT / SIGTERM handling ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("hera-batch: run_id=%s concurrency=%d", effectiveRunID, *concurrency)
	if err := b.Run(ctx); err != nil {
		return fmt.Errorf("batch run: %w", err)
	}

	log.Printf("hera-batch: completed run_id=%s", effectiveRunID)
	return nil
}

func makeWriter(format, outputFile string) (batch.OutputWriter, error) {
	if outputFile == "" {
		// Write to stdout.
		switch format {
		case "csv":
			return batch.NewCSVWriter(os.Stdout), nil
		case "text":
			return batch.NewTextWriter(os.Stdout), nil
		default: // jsonl
			return batch.NewJSONLWriter(os.Stdout), nil
		}
	}

	switch format {
	case "csv":
		return batch.NewCSVFileWriter(outputFile)
	case "text":
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open output file %q: %w", outputFile, err)
		}
		return &textFileWriter{TextWriter: batch.NewTextWriter(f), f: f}, nil
	default: // jsonl
		return batch.NewJSONLFileWriter(outputFile)
	}
}

// textFileWriter wraps TextWriter adding file-close behaviour.
type textFileWriter struct {
	*batch.TextWriter
	f *os.File
}

func (w *textFileWriter) Close() error {
	return w.f.Close()
}

// llmSummarizer wraps an llm.Provider to implement memory.Summarizer.
type llmSummarizer struct {
	llm llm.Provider
}

func (s *llmSummarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
	sysMsg := llm.Message{
		Role:    llm.RoleSystem,
		Content: "Summarize the following conversation concisely, preserving key facts and context. Output only the summary.",
	}
	req := llm.ChatRequest{
		Messages: append([]llm.Message{sysMsg}, messages...),
	}
	resp, err := s.llm.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}
