package cron

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSchedulerTestHelper(t *testing.T) *Scheduler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cron.db")
	s, err := NewScheduler(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.db.Close() })
	return s
}

func TestNewScheduler_Success(t *testing.T) {
	s := newSchedulerTestHelper(t)
	require.NotNil(t, s)
	assert.NotNil(t, s.db)
	assert.NotNil(t, s.jobs)
}

func TestNewScheduler_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "cron.db")
	s, err := NewScheduler(dbPath)
	require.NoError(t, err)
	require.NotNil(t, s)
	s.db.Close()
}

func TestAddJob_Success(t *testing.T) {
	s := newSchedulerTestHelper(t)
	fn := func(ctx context.Context) error { return nil }
	id, err := s.AddJob("test-job", "0 * * * *", "runs hourly", fn)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestAddJob_InvalidCron(t *testing.T) {
	s := newSchedulerTestHelper(t)
	fn := func(ctx context.Context) error { return nil }
	_, err := s.AddJob("bad-cron", "invalid", "desc", fn)
	assert.Error(t, err)
}

func TestListJobs_Empty(t *testing.T) {
	s := newSchedulerTestHelper(t)
	jobs := s.ListJobs()
	assert.Empty(t, jobs)
}

func TestListJobs_WithJobs(t *testing.T) {
	s := newSchedulerTestHelper(t)
	fn := func(ctx context.Context) error { return nil }
	_, err := s.AddJob("job1", "0 * * * *", "desc1", fn)
	require.NoError(t, err)
	_, err = s.AddJob("job2", "*/5 * * * *", "desc2", fn)
	require.NoError(t, err)

	jobs := s.ListJobs()
	assert.Len(t, jobs, 2)
}

func TestRemoveJob(t *testing.T) {
	s := newSchedulerTestHelper(t)
	fn := func(ctx context.Context) error { return nil }
	id, err := s.AddJob("removable", "0 * * * *", "desc", fn)
	require.NoError(t, err)

	err = s.RemoveJob(id)
	require.NoError(t, err)

	jobs := s.ListJobs()
	assert.Empty(t, jobs)
}

func TestEnableJob(t *testing.T) {
	s := newSchedulerTestHelper(t)
	fn := func(ctx context.Context) error { return nil }
	id, err := s.AddJob("toggleable", "0 * * * *", "desc", fn)
	require.NoError(t, err)

	err = s.EnableJob(id, false)
	require.NoError(t, err)

	jobs := s.ListJobs()
	for _, j := range jobs {
		if j.ID == id {
			assert.False(t, j.Enabled)
		}
	}

	err = s.EnableJob(id, true)
	require.NoError(t, err)

	jobs = s.ListJobs()
	for _, j := range jobs {
		if j.ID == id {
			assert.True(t, j.Enabled)
		}
	}
}

func TestEnableJob_NotFound(t *testing.T) {
	s := newSchedulerTestHelper(t)
	err := s.EnableJob("nonexistent-id", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
