// Command hera-supervisor is the agent fleet supervisor daemon.
//
// It reads an agents YAML file, spawns all configured agents, and exposes a
// minimal HTTP status endpoint. Each agent is started via the configured
// factory (defaulting to a simple logged stub for standalone use).
//
// Usage:
//
//	hera-supervisor --config agents.yaml --addr :9090
//	hera-supervisor --config agents.yaml  # uses :9090 by default
//
// The /supervisor/status endpoint returns JSON:
//
//	{
//	  "running": 2,
//	  "stopped": 0,
//	  "failed": 0,
//	  "agents": [
//	    {"name": "coder", "state": "running", "restarts": 0, "started_at": "..."},
//	    ...
//	  ]
//	}
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sadewadee/hera/internal/supervisor"
)

// agentsFile is the structure of the agents YAML configuration.
type agentsFile struct {
	Agents []agentDef `yaml:"agents"`
}

type agentDef struct {
	Name  string `yaml:"name"`
	Model string `yaml:"model,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hera-supervisor: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configFlag := flag.String("config", "agents.yaml", "Path to agents YAML file")
	addrFlag := flag.String("addr", ":9090", "HTTP status endpoint address")
	maxFlag := flag.Int("max-concurrent", 0, "Maximum concurrent agents (0 = unlimited)")
	flag.Parse()

	logger := slog.Default()

	// Load agent definitions.
	defs, err := loadAgents(*configFlag)
	if err != nil {
		return fmt.Errorf("load agents config %q: %w", *configFlag, err)
	}
	if len(defs) == 0 {
		logger.Warn("no agents defined in config; supervisor will idle")
	}

	// Build supervisor with a stub factory. In production usage, callers
	// inject a real AgentFactory that starts actual hera Agent processes.
	sup, err := supervisor.New(supervisor.Config{
		MaxConcurrent: *maxFlag,
		Factory: func(ctx context.Context, name string) error {
			logger.Info("agent started", "name", name)
			<-ctx.Done()
			logger.Info("agent stopped", "name", name)
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("create supervisor: %w", err)
	}

	// Spawn all configured agents.
	for _, def := range defs {
		if spawnErr := sup.Spawn(def.Name); spawnErr != nil {
			logger.Warn("spawn failed", "name", def.Name, "error", spawnErr)
		}
	}

	// HTTP status endpoint.
	mux := http.NewServeMux()
	mux.HandleFunc("/supervisor/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sup.HealthReport()); err != nil {
			http.Error(w, `{"error":"encode failed"}`, http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/supervisor/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	srv := &http.Server{
		Addr:         *addrFlag,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("hera-supervisor: listening", "addr", *addrFlag, "agents", len(defs))
		if serveErr := srv.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			serverErr <- serveErr
		}
	}()

	select {
	case err := <-serverErr:
		sup.StopAll()
		return fmt.Errorf("HTTP server error: %w", err)
	case <-ctx.Done():
		logger.Info("hera-supervisor: shutting down")
		sup.StopAll()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}

// loadAgents reads the agents YAML file and returns the definitions.
// Returns an empty slice (not an error) if the file doesn't exist.
func loadAgents(path string) ([]agentDef, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var af agentsFile
	if err := yaml.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return af.Agents, nil
}
