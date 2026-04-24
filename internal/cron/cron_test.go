package cron

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// --- parseField tests ---

func TestParseField_Wildcard(t *testing.T) {
	result, err := parseField("*", 0, 59)
	if err != nil {
		t.Fatalf("parseField('*', 0, 59) error = %v", err)
	}
	if len(result) != 60 {
		t.Errorf("parseField('*', 0, 59) returned %d values, want 60", len(result))
	}
	for i := 0; i <= 59; i++ {
		if !result[i] {
			t.Errorf("parseField('*', 0, 59) missing value %d", i)
		}
	}
}

func TestParseField_Step(t *testing.T) {
	result, err := parseField("*/5", 0, 59)
	if err != nil {
		t.Fatalf("parseField('*/5', 0, 59) error = %v", err)
	}
	expected := map[int]bool{0: true, 5: true, 10: true, 15: true, 20: true, 25: true, 30: true, 35: true, 40: true, 45: true, 50: true, 55: true}
	if len(result) != len(expected) {
		t.Errorf("parseField('*/5', 0, 59) returned %d values, want %d", len(result), len(expected))
	}
	for v := range expected {
		if !result[v] {
			t.Errorf("parseField('*/5', 0, 59) missing value %d", v)
		}
	}
}

func TestParseField_Range(t *testing.T) {
	result, err := parseField("1-5", 0, 10)
	if err != nil {
		t.Fatalf("parseField('1-5', 0, 10) error = %v", err)
	}
	if len(result) != 5 {
		t.Errorf("parseField('1-5', 0, 10) returned %d values, want 5", len(result))
	}
	for i := 1; i <= 5; i++ {
		if !result[i] {
			t.Errorf("parseField('1-5', 0, 10) missing value %d", i)
		}
	}
}

func TestParseField_Single(t *testing.T) {
	result, err := parseField("3", 0, 10)
	if err != nil {
		t.Fatalf("parseField('3', 0, 10) error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("parseField('3', 0, 10) returned %d values, want 1", len(result))
	}
	if !result[3] {
		t.Error("parseField('3', 0, 10) missing value 3")
	}
}

func TestParseField_Comma(t *testing.T) {
	result, err := parseField("1,3,5", 0, 10)
	if err != nil {
		t.Fatalf("parseField('1,3,5', 0, 10) error = %v", err)
	}
	if len(result) != 3 {
		t.Errorf("parseField('1,3,5', 0, 10) returned %d values, want 3", len(result))
	}
	for _, v := range []int{1, 3, 5} {
		if !result[v] {
			t.Errorf("parseField('1,3,5', 0, 10) missing value %d", v)
		}
	}
}

// --- nextCronTime tests ---

func TestNextCronTime_EveryMinute(t *testing.T) {
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("* * * * *", now)
	if err != nil {
		t.Fatalf("nextCronTime('* * * * *') error = %v", err)
	}
	want := time.Date(2026, 4, 10, 14, 31, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('* * * * *') = %v, want %v", next, want)
	}
}

func TestNextCronTime_SpecificTime(t *testing.T) {
	// "30 14 * * *" means 14:30 every day.
	// If current time is 14:30, the next occurrence should be the next day at 14:30.
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("30 14 * * *", now)
	if err != nil {
		t.Fatalf("nextCronTime('30 14 * * *') error = %v", err)
	}
	want := time.Date(2026, 4, 11, 14, 30, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('30 14 * * *') = %v, want %v", next, want)
	}
}

func TestNextCronTime_InvalidExpr(t *testing.T) {
	_, err := nextCronTime("* * *", time.Now())
	if err == nil {
		t.Error("nextCronTime('* * *') expected error for wrong field count")
	}
}

// --- nextCronTime shorthand tests ---

func TestNextCronTime_Monthly(t *testing.T) {
	// @monthly = "0 0 1 * *" — first of every month at 00:00
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("@monthly", now)
	if err != nil {
		t.Fatalf("nextCronTime('@monthly') error = %v", err)
	}
	want := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('@monthly') = %v, want %v", next, want)
	}
}

func TestNextCronTime_Weekly(t *testing.T) {
	// @weekly = "0 0 * * 0" — every Sunday at 00:00
	// 2026-04-10 is a Friday; next Sunday is 2026-04-12.
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("@weekly", now)
	if err != nil {
		t.Fatalf("nextCronTime('@weekly') error = %v", err)
	}
	want := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('@weekly') = %v, want %v", next, want)
	}
}

func TestNextCronTime_Daily(t *testing.T) {
	// @daily = "0 0 * * *" — every day at 00:00
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("@daily", now)
	if err != nil {
		t.Fatalf("nextCronTime('@daily') error = %v", err)
	}
	want := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('@daily') = %v, want %v", next, want)
	}
}

func TestNextCronTime_Hourly(t *testing.T) {
	// @hourly = "0 * * * *" — every hour at :00
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("@hourly", now)
	if err != nil {
		t.Fatalf("nextCronTime('@hourly') error = %v", err)
	}
	want := time.Date(2026, 4, 10, 15, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('@hourly') = %v, want %v", next, want)
	}
}

func TestNextCronTime_Yearly(t *testing.T) {
	// @yearly = "0 0 1 1 *" — January 1 at 00:00
	now := time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC)
	next, err := nextCronTime("@yearly", now)
	if err != nil {
		t.Fatalf("nextCronTime('@yearly') error = %v", err)
	}
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("nextCronTime('@yearly') = %v, want %v", next, want)
	}
}

// --- Scheduler tests ---

func newTestScheduler(t *testing.T) *Scheduler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cron_test.db")
	s, err := NewScheduler(dbPath)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestScheduler_AddAndList(t *testing.T) {
	s := newTestScheduler(t)

	id, err := s.AddJob("test-job", "*/5 * * * *", "runs every 5 minutes", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}
	if id == "" {
		t.Error("AddJob() returned empty ID")
	}

	jobs := s.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("ListJobs() returned %d jobs, want 1", len(jobs))
	}
	if jobs[0].Name != "test-job" {
		t.Errorf("job name = %q, want %q", jobs[0].Name, "test-job")
	}
	if jobs[0].CronExpr != "*/5 * * * *" {
		t.Errorf("job cron_expr = %q, want %q", jobs[0].CronExpr, "*/5 * * * *")
	}
	if !jobs[0].Enabled {
		t.Error("job should be enabled by default")
	}
}

func TestScheduler_Remove(t *testing.T) {
	s := newTestScheduler(t)

	id, err := s.AddJob("removable", "* * * * *", "will be removed", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	if err := s.RemoveJob(id); err != nil {
		t.Fatalf("RemoveJob() error = %v", err)
	}

	jobs := s.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("ListJobs() returned %d jobs after remove, want 0", len(jobs))
	}
}

func TestScheduler_EnableDisable(t *testing.T) {
	s := newTestScheduler(t)

	id, err := s.AddJob("toggle-job", "0 * * * *", "hourly", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	// Disable.
	if err := s.EnableJob(id, false); err != nil {
		t.Fatalf("EnableJob(false) error = %v", err)
	}
	jobs := s.ListJobs()
	for _, j := range jobs {
		if j.ID == id && j.Enabled {
			t.Error("job should be disabled after EnableJob(false)")
		}
	}

	// Re-enable.
	if err := s.EnableJob(id, true); err != nil {
		t.Fatalf("EnableJob(true) error = %v", err)
	}
	jobs = s.ListJobs()
	for _, j := range jobs {
		if j.ID == id && !j.Enabled {
			t.Error("job should be enabled after EnableJob(true)")
		}
	}

	// Enable nonexistent job.
	if err := s.EnableJob("nonexistent-id", true); err == nil {
		t.Error("EnableJob() expected error for nonexistent job")
	}
}
