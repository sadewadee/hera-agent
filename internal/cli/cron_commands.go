package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sadewadee/hera/internal/cron"
)

// HandleCronCommand processes /cron subcommands for managing scheduled jobs.
func HandleCronCommand(args string, scheduler *cron.Scheduler) (string, error) {
	if scheduler == nil {
		return "Cron scheduler not initialized.", nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "Usage: /cron <add|remove|list|enable|disable> [args...]", nil
	}

	switch parts[0] {
	case "add":
		if len(parts) < 3 {
			return "Usage: /cron add <cron-expression> <command>", nil
		}
		expr := parts[1]
		command := strings.Join(parts[2:], " ")
		name := fmt.Sprintf("cli-job-%s", expr)

		fn := func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "sh", "-c", command)
			return cmd.Run()
		}

		id, err := scheduler.AddJob(name, expr, command, fn)
		if err != nil {
			return fmt.Sprintf("Error adding job: %v", err), nil
		}
		return fmt.Sprintf("Added cron job '%s' (id: %s): %s -> %s", name, id, expr, command), nil

	case "remove":
		if len(parts) < 2 {
			return "Usage: /cron remove <job-id>", nil
		}
		jobID := parts[1]
		if err := scheduler.RemoveJob(jobID); err != nil {
			return fmt.Sprintf("Error removing job: %v", err), nil
		}
		return fmt.Sprintf("Removed job '%s'", jobID), nil

	case "list":
		jobs := scheduler.ListJobs()
		if len(jobs) == 0 {
			return "No cron jobs configured.", nil
		}
		var sb strings.Builder
		sb.WriteString("Cron jobs:\n")
		for _, j := range jobs {
			status := "enabled"
			if !j.Enabled {
				status = "disabled"
			}
			sb.WriteString(fmt.Sprintf("  %-36s %-15s %-10s %s\n", j.ID, j.CronExpr, status, j.Name))
		}
		return sb.String(), nil

	case "enable":
		if len(parts) < 2 {
			return "Usage: /cron enable <job-id>", nil
		}
		if err := scheduler.EnableJob(parts[1], true); err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		return fmt.Sprintf("Enabled job '%s'", parts[1]), nil

	case "disable":
		if len(parts) < 2 {
			return "Usage: /cron disable <job-id>", nil
		}
		if err := scheduler.EnableJob(parts[1], false); err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		return fmt.Sprintf("Disabled job '%s'", parts[1]), nil

	default:
		return "Usage: /cron <add|remove|list|enable|disable> [args...]", nil
	}
}
