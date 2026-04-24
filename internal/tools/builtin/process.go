package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// procState tracks everything we know about a background process the
// LLM started via this tool. The cmd pointer lets us kill it; the
// other fields are state the goroutines fill in as the process runs
// and exits, so later `list`/`status` calls return useful info.
type procState struct {
	cmd          *exec.Cmd
	command      string
	startedAt    time.Time
	notifyOnDone bool
	patterns     []string

	mu         sync.Mutex
	output     strings.Builder // tail of stdout/stderr (bounded)
	done       bool
	exitErr    error
	finishedAt time.Time
	matches    []patternHit // in arrival order
}

type patternHit struct {
	Pattern string
	Line    string
	At      time.Time
}

const procOutputCap = 16 * 1024 // keep at most 16KiB of tail output per proc

// ProcessTool manages background processes (start, stop, list, status).
type ProcessTool struct {
	mu    sync.Mutex
	procs map[string]*procState
}

type processArgs struct {
	Action        string   `json:"action"`
	Command       string   `json:"command,omitempty"`
	ID            string   `json:"id,omitempty"`
	NotifyOnDone  bool     `json:"notify_on_complete,omitempty"`
	WatchPatterns []string `json:"watch_patterns,omitempty"`
}

func (t *ProcessTool) Name() string { return "process" }
func (t *ProcessTool) Description() string {
	return "Manages background processes (start, stop, list, status). Supports watch_patterns for scanning stdout/stderr and notify_on_complete for surfacing exit state on subsequent list/status calls."
}

func (t *ProcessTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["start", "stop", "list", "status"], "description": "Process action"},
			"command": {"type": "string", "description": "Shell command to start (used with start)"},
			"id": {"type": "string", "description": "Process ID (used with stop and status)"},
			"notify_on_complete": {"type": "boolean", "description": "When true, subsequent list/status calls surface completion time and exit code"},
			"watch_patterns": {"type": "array", "items": {"type": "string"}, "description": "Substrings to match in stdout/stderr; matches are recorded and surfaced via list/status"}
		},
		"required": ["action"]
	}`)
}

func (t *ProcessTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a processArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}

	t.mu.Lock()
	if t.procs == nil {
		t.procs = make(map[string]*procState)
	}
	t.mu.Unlock()

	switch a.Action {
	case "start":
		return t.start(ctx, a)
	case "stop":
		return t.stop(a)
	case "list":
		return t.list()
	case "status":
		return t.status(a)
	default:
		return &tools.Result{Content: "unknown action", IsError: true}, nil
	}
}

func (t *ProcessTool) start(ctx context.Context, a processArgs) (*tools.Result, error) {
	if a.Command == "" {
		return &tools.Result{Content: "command is required for start action", IsError: true}, nil
	}
	if result := checkCommandSafety(a.Command, nil); result != nil {
		return result, nil
	}

	// exec.CommandContext ties lifetime to ctx, but the tool's ctx usually
	// ends with the current turn; we want the background proc to outlive
	// it. Use context.Background() so the proc only dies via stop or OS exit.
	cmd := exec.CommandContext(context.Background(), "sh", "-c", a.Command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("stdout pipe: %v", err), IsError: true}, nil
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("stderr pipe: %v", err), IsError: true}, nil
	}
	if err := cmd.Start(); err != nil {
		return &tools.Result{Content: fmt.Sprintf("failed to start process: %v", err), IsError: true}, nil
	}

	t.mu.Lock()
	id := fmt.Sprintf("proc-%d", len(t.procs)+1)
	ps := &procState{
		cmd:          cmd,
		command:      a.Command,
		startedAt:    time.Now(),
		notifyOnDone: a.NotifyOnDone,
		patterns:     a.WatchPatterns,
	}
	t.procs[id] = ps
	t.mu.Unlock()

	// Scan stdout + stderr so we can record output and match patterns.
	go ps.scan(stdout)
	go ps.scan(stderr)

	// Wait for the process to exit (separate goroutine) so we can mark
	// it done without blocking the tool call.
	go func() {
		err := cmd.Wait()
		ps.markDone(err)
	}()

	ack := fmt.Sprintf("Process %s started: %s", id, a.Command)
	if len(a.WatchPatterns) > 0 {
		ack += fmt.Sprintf("\n  watching patterns: %v", a.WatchPatterns)
	}
	if a.NotifyOnDone {
		ack += "\n  will record completion state (check via list or status)"
	}
	return &tools.Result{Content: ack}, nil
}

func (t *ProcessTool) stop(a processArgs) (*tools.Result, error) {
	t.mu.Lock()
	ps, ok := t.procs[a.ID]
	t.mu.Unlock()
	if !ok {
		return &tools.Result{Content: "process not found", IsError: true}, nil
	}
	if ps.cmd.Process != nil {
		_ = ps.cmd.Process.Kill()
	}
	t.mu.Lock()
	delete(t.procs, a.ID)
	t.mu.Unlock()
	return &tools.Result{Content: fmt.Sprintf("Process %s stopped", a.ID)}, nil
}

func (t *ProcessTool) list() (*tools.Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d active processes", len(t.procs))
	for id, ps := range t.procs {
		ps.mu.Lock()
		status := "running"
		if ps.done {
			status = "done"
			if ps.exitErr != nil {
				status = fmt.Sprintf("done (err: %v)", ps.exitErr)
			}
		}
		sb.WriteString(fmt.Sprintf("\n  %s [%s] %s", id, status, ps.command))
		if len(ps.matches) > 0 {
			sb.WriteString(fmt.Sprintf(" — %d pattern match(es)", len(ps.matches)))
		}
		ps.mu.Unlock()
	}
	return &tools.Result{Content: sb.String()}, nil
}

func (t *ProcessTool) status(a processArgs) (*tools.Result, error) {
	if a.ID == "" {
		return &tools.Result{Content: "id is required for status action", IsError: true}, nil
	}
	t.mu.Lock()
	ps, ok := t.procs[a.ID]
	t.mu.Unlock()
	if !ok {
		return &tools.Result{Content: fmt.Sprintf("process %q not found", a.ID), IsError: true}, nil
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	var sb strings.Builder
	fmt.Fprintf(&sb, "Process %s\n", a.ID)
	fmt.Fprintf(&sb, "  command: %s\n", ps.command)
	fmt.Fprintf(&sb, "  started: %s\n", ps.startedAt.Format(time.RFC3339))
	if ps.done {
		fmt.Fprintf(&sb, "  finished: %s\n", ps.finishedAt.Format(time.RFC3339))
		if ps.exitErr != nil {
			fmt.Fprintf(&sb, "  exit_error: %v\n", ps.exitErr)
		} else {
			fmt.Fprintf(&sb, "  exit_code: 0\n")
		}
	} else {
		fmt.Fprintf(&sb, "  state: running\n")
	}
	if len(ps.matches) > 0 {
		sb.WriteString("  pattern matches:\n")
		for _, m := range ps.matches {
			fmt.Fprintf(&sb, "    [%s] %q → %s\n", m.At.Format("15:04:05"), m.Pattern, m.Line)
		}
	}
	if ps.output.Len() > 0 {
		out := ps.output.String()
		if len(out) > 2000 {
			out = "...\n" + out[len(out)-2000:]
		}
		fmt.Fprintf(&sb, "  tail output:\n%s", out)
	}
	return &tools.Result{Content: sb.String()}, nil
}

// scan reads r line by line, records output (bounded), and matches
// the configured patterns. Terminates when r closes.
func (ps *procState) scan(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		ps.mu.Lock()
		// Bounded output buffer: drop older content once cap exceeded.
		if ps.output.Len()+len(line)+1 > procOutputCap {
			drop := ps.output.Len() + len(line) + 1 - procOutputCap
			cur := ps.output.String()
			if drop >= len(cur) {
				ps.output.Reset()
			} else {
				ps.output.Reset()
				ps.output.WriteString(cur[drop:])
			}
		}
		ps.output.WriteString(line)
		ps.output.WriteByte('\n')

		for _, p := range ps.patterns {
			if p != "" && strings.Contains(line, p) {
				ps.matches = append(ps.matches, patternHit{
					Pattern: p,
					Line:    line,
					At:      time.Now(),
				})
			}
		}
		ps.mu.Unlock()
	}
}

// markDone records exit state so subsequent list/status calls can
// report completion. Only has effect when notifyOnDone is true —
// processes without the flag still get their exitErr recorded so
// `status` stays useful.
func (ps *procState) markDone(err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.done = true
	ps.exitErr = err
	ps.finishedAt = time.Now()
}

// RegisterProcess registers the process management tool.
func RegisterProcess(registry *tools.Registry) { registry.Register(&ProcessTool{}) }
