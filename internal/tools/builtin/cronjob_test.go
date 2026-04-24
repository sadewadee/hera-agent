package builtin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/cron"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronJobTool_Name(t *testing.T) {
	tool := &CronJobTool{}
	assert.Equal(t, "cronjob", tool.Name())
}

func TestCronJobTool_InvalidArgs(t *testing.T) {
	tool := &CronJobTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// TestCronJobTool_NilScheduler verifies the graceful "cron disabled" path.
func TestCronJobTool_NilScheduler(t *testing.T) {
	tool := &CronJobTool{scheduler: nil}
	for _, action := range []string{"create", "list", "remove"} {
		args, _ := json.Marshal(cronjobArgs{Action: action})
		result, err := tool.Execute(context.Background(), args)
		require.NoError(t, err)
		assert.False(t, result.IsError, "nil-scheduler should not return IsError=true")
		assert.Contains(t, result.Content, "not enabled")
	}
}

// TestCronJobTool_Create_MissingFields ensures required fields are validated.
func TestCronJobTool_Create_MissingFields(t *testing.T) {
	tool := &CronJobTool{scheduler: mustNewScheduler(t)}
	args, _ := json.Marshal(cronjobArgs{Action: "create", Name: "backup"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// TestCronJobTool_CreateListRemove exercises the full lifecycle with a real scheduler.
func TestCronJobTool_CreateListRemove(t *testing.T) {
	tool := &CronJobTool{scheduler: mustNewScheduler(t)}

	// Create a job using a standard cron expression.
	createArgs, _ := json.Marshal(cronjobArgs{
		Action:   "create",
		Name:     "test-backup",
		Schedule: "0 2 * * *",
		Command:  "echo hello",
	})
	createResult, err := tool.Execute(context.Background(), createArgs)
	require.NoError(t, err)
	require.False(t, createResult.IsError, createResult.Content)
	assert.Contains(t, createResult.Content, "test-backup")
	assert.Contains(t, createResult.Content, "created")

	// List should include the job.
	listArgs, _ := json.Marshal(cronjobArgs{Action: "list"})
	listResult, err := tool.Execute(context.Background(), listArgs)
	require.NoError(t, err)
	assert.Contains(t, listResult.Content, "test-backup")

	// Extract job ID from create result ("id: <uuid>").
	// The format is: Cron job "test-backup" created (id: <uuid>, schedule: ...)
	content := createResult.Content
	idStart := indexOf(content, "id: ") + 4
	idEnd := indexOf(content[idStart:], ",")
	if idEnd == -1 {
		idEnd = indexOf(content[idStart:], ")")
	}
	jobID := content[idStart : idStart+idEnd]

	// Remove the job.
	removeArgs, _ := json.Marshal(cronjobArgs{Action: "remove", ID: jobID})
	removeResult, err := tool.Execute(context.Background(), removeArgs)
	require.NoError(t, err)
	assert.False(t, removeResult.IsError, removeResult.Content)
	assert.Contains(t, removeResult.Content, "removed")

	// List should be empty now.
	listResult2, err := tool.Execute(context.Background(), listArgs)
	require.NoError(t, err)
	assert.Contains(t, listResult2.Content, "No cron jobs")
}

// TestCronJobTool_NaturalLanguageSchedule verifies NL cron parsing path.
func TestCronJobTool_NaturalLanguageSchedule(t *testing.T) {
	tool := &CronJobTool{scheduler: mustNewScheduler(t)}
	createArgs, _ := json.Marshal(cronjobArgs{
		Action:   "create",
		Name:     "daily-report",
		Schedule: "every day at 9am",
		Command:  "echo report",
	})
	result, err := tool.Execute(context.Background(), createArgs)
	require.NoError(t, err)
	// Either succeeds or fails with a clear error — no panic.
	if result.IsError {
		t.Logf("NL cron parse rejected: %s", result.Content)
	} else {
		assert.Contains(t, result.Content, "daily-report")
	}
}

// TestCronJobTool_RemoveMissingID ensures remove validates ID.
func TestCronJobTool_RemoveMissingID(t *testing.T) {
	tool := &CronJobTool{scheduler: mustNewScheduler(t)}
	args, _ := json.Marshal(cronjobArgs{Action: "remove"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "id is required")
}

// TestCronJobTool_UnknownAction verifies unknown actions return errors.
func TestCronJobTool_UnknownAction(t *testing.T) {
	tool := &CronJobTool{scheduler: mustNewScheduler(t)}
	args, _ := json.Marshal(cronjobArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// mustNewScheduler creates a real scheduler backed by a temp SQLite DB.
func mustNewScheduler(t *testing.T) *cron.Scheduler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cron_test.db")
	s, err := cron.NewScheduler(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// indexOf returns the index of needle in s, or -1 if not found.
func indexOf(s, needle string) int {
	for i := 0; i <= len(s)-len(needle); i++ {
		if s[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
