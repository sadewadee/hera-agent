package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sadewadee/hera/internal/cron"
	"github.com/sadewadee/hera/internal/tools"
)

// CronJobTool wraps the real cron scheduler so the LLM can create, list, and
// remove scheduled jobs using natural-language schedule descriptions.
type CronJobTool struct {
	scheduler *cron.Scheduler
}

type cronjobArgs struct {
	Action   string `json:"action"`
	Name     string `json:"name,omitempty"`
	Schedule string `json:"schedule,omitempty"`
	Command  string `json:"command,omitempty"`
	ID       string `json:"id,omitempty"`
}

func (t *CronJobTool) Name() string { return "cronjob" }

func (t *CronJobTool) Description() string {
	return "Creates and manages scheduled cron jobs. Accepts natural-language schedules " +
		"(e.g. 'every monday at 9am') or standard 5-field cron expressions."
}

func (t *CronJobTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["create", "list", "remove"],
				"description": "Action to perform on cron jobs."
			},
			"name": {
				"type": "string",
				"description": "Human-readable job name (required for create)."
			},
			"schedule": {
				"type": "string",
				"description": "Natural language (e.g. 'every day at 9am') or cron expression (required for create)."
			},
			"command": {
				"type": "string",
				"description": "Shell command to run (required for create)."
			},
			"id": {
				"type": "string",
				"description": "Job ID returned by create (required for remove)."
			}
		},
		"required": ["action"]
	}`)
}

func (t *CronJobTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a cronjobArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}

	if t.scheduler == nil {
		return &tools.Result{
			Content: "Cron scheduler is not enabled. Set cron.enabled: true in config and restart.",
			IsError: false,
		}, nil
	}

	switch a.Action {
	case "create":
		if a.Name == "" || a.Schedule == "" || a.Command == "" {
			return &tools.Result{Content: "name, schedule, and command are required for create", IsError: true}, nil
		}
		cronExpr, err := cron.ParseNLCron(a.Schedule)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("invalid schedule %q: %v", a.Schedule, err), IsError: true}, nil
		}
		cmd := a.Command // capture for closure
		jobID, err := t.scheduler.AddJob(a.Name, cronExpr, a.Schedule, func(ctx context.Context) error {
			parts := strings.Fields(cmd)
			if len(parts) == 0 {
				return fmt.Errorf("empty command")
			}
			//nolint:gosec // command comes from authenticated LLM tool call
			out, runErr := exec.CommandContext(ctx, parts[0], parts[1:]...).CombinedOutput()
			if runErr != nil {
				return fmt.Errorf("command %q failed: %w\n%s", cmd, runErr, out)
			}
			return nil
		})
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("failed to create job: %v", err), IsError: true}, nil
		}
		return &tools.Result{
			Content: fmt.Sprintf("Cron job %q created (id: %s, schedule: %s → %s)", a.Name, jobID, a.Schedule, cronExpr),
		}, nil

	case "list":
		jobs := t.scheduler.ListJobs()
		if len(jobs) == 0 {
			return &tools.Result{Content: "No cron jobs scheduled."}, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Scheduled jobs (%d):\n", len(jobs)))
		for _, j := range jobs {
			status := "enabled"
			if !j.Enabled {
				status = "disabled"
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s — %s (next: %s, %s)\n",
				j.ID, j.Name, j.CronExpr, j.NextRunAt.Format("2006-01-02 15:04"), status))
		}
		return &tools.Result{Content: sb.String()}, nil

	case "remove":
		if a.ID == "" {
			return &tools.Result{Content: "id is required for remove", IsError: true}, nil
		}
		if err := t.scheduler.RemoveJob(a.ID); err != nil {
			return &tools.Result{Content: fmt.Sprintf("failed to remove job: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: fmt.Sprintf("Cron job %q removed.", a.ID)}, nil

	default:
		return &tools.Result{Content: fmt.Sprintf("unknown action %q; use create, list, or remove", a.Action), IsError: true}, nil
	}
}

// RegisterCronJob registers the cronjob tool. scheduler may be nil when cron
// is disabled; the tool will report a clear message instead of panicking.
func RegisterCronJob(registry *tools.Registry, scheduler *cron.Scheduler) {
	registry.Register(&CronJobTool{scheduler: scheduler})
}
