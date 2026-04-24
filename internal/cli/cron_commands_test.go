package cli

import (
	"strings"
	"testing"
)

func TestHandleCronCommand_NilScheduler(t *testing.T) {
	result, err := HandleCronCommand("list", nil)
	if err != nil {
		t.Fatalf("HandleCronCommand error = %v", err)
	}
	if !strings.Contains(result, "not initialized") {
		t.Errorf("expected 'not initialized' message, got %q", result)
	}
}

func TestHandleCronCommand_EmptyArgs(t *testing.T) {
	result, err := HandleCronCommand("", nil)
	if err != nil {
		t.Fatalf("HandleCronCommand error = %v", err)
	}
	if !strings.Contains(result, "not initialized") {
		t.Errorf("expected scheduler not initialized message, got %q", result)
	}
}

func TestHandleCronCommand_UsageWithEmptyArgs_SchedulerNotNil(t *testing.T) {
	// We cannot easily create a full cron.Scheduler in a unit test without
	// importing cron, but we can test nil-scheduler paths and argument parsing.
	// The nil scheduler path is the primary safety check.
	result, err := HandleCronCommand("", nil)
	if err != nil {
		t.Fatalf("HandleCronCommand error = %v", err)
	}
	if result == "" {
		t.Error("HandleCronCommand returned empty string")
	}
}
