package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// SystemInfoTool retrieves system information (CPU, memory, disk, OS).
type SystemInfoTool struct{}

type systemInfoArgs struct {
	Category string `json:"category,omitempty"`
}

func (t *SystemInfoTool) Name() string { return "system_info" }

func (t *SystemInfoTool) Description() string {
	return "Returns system information: OS, CPU, memory, environment variables, and Go runtime stats."
}

func (t *SystemInfoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"category": {
				"type": "string",
				"enum": ["all", "os", "cpu", "memory", "env", "runtime"],
				"description": "Information category. Defaults to 'all'."
			}
		}
	}`)
}

func (t *SystemInfoTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params systemInfoArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
		}
	}

	category := params.Category
	if category == "" {
		category = "all"
	}

	var sb strings.Builder
	hostname, _ := os.Hostname()
	wd, _ := os.Getwd()

	if category == "all" || category == "os" {
		fmt.Fprintf(&sb, "=== Operating System ===\n")
		fmt.Fprintf(&sb, "OS:       %s\n", runtime.GOOS)
		fmt.Fprintf(&sb, "Arch:     %s\n", runtime.GOARCH)
		fmt.Fprintf(&sb, "Hostname: %s\n", hostname)
		fmt.Fprintf(&sb, "CWD:      %s\n", wd)
		fmt.Fprintf(&sb, "PID:      %d\n", os.Getpid())
		fmt.Fprintf(&sb, "\n")
	}

	if category == "all" || category == "cpu" {
		fmt.Fprintf(&sb, "=== CPU ===\n")
		fmt.Fprintf(&sb, "Logical CPUs:  %d\n", runtime.NumCPU())
		fmt.Fprintf(&sb, "GOMAXPROCS:    %d\n", runtime.GOMAXPROCS(0))
		fmt.Fprintf(&sb, "\n")
	}

	if category == "all" || category == "memory" {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Fprintf(&sb, "=== Memory (Go Runtime) ===\n")
		fmt.Fprintf(&sb, "Alloc:      %s\n", formatBytes(m.Alloc))
		fmt.Fprintf(&sb, "TotalAlloc: %s\n", formatBytes(m.TotalAlloc))
		fmt.Fprintf(&sb, "Sys:        %s\n", formatBytes(m.Sys))
		fmt.Fprintf(&sb, "HeapAlloc:  %s\n", formatBytes(m.HeapAlloc))
		fmt.Fprintf(&sb, "HeapSys:    %s\n", formatBytes(m.HeapSys))
		fmt.Fprintf(&sb, "NumGC:      %d\n", m.NumGC)
		fmt.Fprintf(&sb, "\n")
	}

	if category == "all" || category == "runtime" {
		fmt.Fprintf(&sb, "=== Go Runtime ===\n")
		fmt.Fprintf(&sb, "Version:     %s\n", runtime.Version())
		fmt.Fprintf(&sb, "Goroutines:  %d\n", runtime.NumGoroutine())
		fmt.Fprintf(&sb, "Compiler:    %s\n", runtime.Compiler)
		fmt.Fprintf(&sb, "Uptime:      %s\n", time.Since(startTime).Truncate(time.Second))
		fmt.Fprintf(&sb, "\n")
	}

	if category == "env" {
		fmt.Fprintf(&sb, "=== Environment (non-sensitive) ===\n")
		for _, env := range os.Environ() {
			key := strings.SplitN(env, "=", 2)[0]
			lower := strings.ToLower(key)
			if strings.Contains(lower, "key") || strings.Contains(lower, "secret") ||
				strings.Contains(lower, "password") || strings.Contains(lower, "token") {
				continue // skip sensitive values
			}
			fmt.Fprintf(&sb, "%s\n", env)
		}
	}

	return &tools.Result{Content: strings.TrimSpace(sb.String())}, nil
}

var startTime = time.Now()

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RegisterSystemInfo registers the system info tool with the given registry.
func RegisterSystemInfo(registry *tools.Registry) {
	registry.Register(&SystemInfoTool{})
}
