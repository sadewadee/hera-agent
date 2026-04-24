# Cron / Scheduled Tasks

Hera includes a cron scheduler that can run agent tasks on a schedule. Jobs are persisted in SQLite and survive restarts.

## Enabling cron

```yaml
cron:
  enabled: true
```

## How it works

The cron scheduler polls registered jobs on a tick rate (1 second by default). When a job's `NextRunAt` time arrives, it executes the registered `JobFunc` in a goroutine.

Job state (last run time, next run time, enabled/disabled) is stored in `~/.hera/hera.db` alongside the memory database.

## Cron expressions

Hera uses standard 5-field cron expressions:

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

Examples:

| Expression | Meaning |
|------------|---------|
| `0 9 * * 1-5` | Weekdays at 9:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 0 * * *` | Daily at midnight |
| `0 */6 * * *` | Every 6 hours |
| `30 18 * * 5` | Fridays at 6:30 PM |

## Creating jobs programmatically

The `cronjob` built-in tool lets the agent manage scheduled tasks:

```
You: Schedule a daily report every morning at 9am

Hera: [using cronjob tool]
I'll create a cron job "daily-report" with expression "0 9 * * *".
The job will run a shell command that generates the report.

Job created. It will first run tomorrow at 09:00.
```

## Job persistence

```go
// Job schema in SQLite
type Job struct {
    ID          string    // UUID
    Name        string    // human-readable name
    CronExpr    string    // "0 9 * * 1-5"
    Description string
    Enabled     bool
    LastRunAt   time.Time
    NextRunAt   time.Time
    CreatedAt   time.Time
}
```

## The cronjob tool

**Tool name:** `cronjob`

The agent uses this tool to schedule recurring tasks. Underlying, it registers a job with the Scheduler:

```
You: Run "go test ./..." every hour and store results in test-results.log

Hera: [using cronjob]
Created job "hourly-tests" (*/1 * * * *)
Command: go test ./... >> test-results.log 2>&1
First run: next hour
```

## Managing jobs

```
You: List my scheduled jobs

Hera: [using cronjob: list]
Active jobs:
  - hourly-tests: "*/1 * * * *", last run: 10:00, next: 11:00
  - daily-report: "0 9 * * *", next: tomorrow 09:00

You: Disable the hourly-tests job

Hera: [using cronjob: disable]
Job "hourly-tests" disabled.

You: Delete the daily-report job

Hera: [using cronjob: delete]
Job "daily-report" deleted.
```

## Use cases

- Daily standup report generation
- Periodic health checks and alerts
- Scheduled file cleanup
- Nightly test runs
- Regular memory consolidation
