package cron

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// JobFunc is the function signature executed by a scheduled job.
type JobFunc func(ctx context.Context) error

// Job represents a scheduled task.
type Job struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CronExpr    string    `json:"cron_expr"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	LastRunAt   time.Time `json:"last_run_at"`
	NextRunAt   time.Time `json:"next_run_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Scheduler manages cron-based job scheduling with SQLite persistence.
type Scheduler struct {
	mu       sync.RWMutex
	db       *sql.DB
	jobs     map[string]*registeredJob
	running  bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *slog.Logger
	tickRate time.Duration
}

type registeredJob struct {
	Job  Job
	Func JobFunc
}

// NewScheduler creates a new scheduler with SQLite persistence at the given path.
func NewScheduler(dbPath string) (*Scheduler, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	s := &Scheduler{
		db:       db,
		jobs:     make(map[string]*registeredJob),
		logger:   slog.Default(),
		tickRate: 1 * time.Minute,
	}

	if err := s.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return s, nil
}

func (s *Scheduler) createTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS cron_jobs (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL UNIQUE,
			cron_expr   TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			enabled     INTEGER NOT NULL DEFAULT 1,
			last_run_at DATETIME,
			next_run_at DATETIME,
			created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)
	`)
	return err
}

// AddJob registers a new job with the scheduler and persists it.
func (s *Scheduler) AddJob(name, cronExpr, description string, fn JobFunc) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	nextRun, err := nextCronTime(cronExpr, now)
	if err != nil {
		return "", fmt.Errorf("parse cron expression %q: %w", cronExpr, err)
	}

	job := Job{
		ID:          id,
		Name:        name,
		CronExpr:    cronExpr,
		Description: description,
		Enabled:     true,
		NextRunAt:   nextRun,
		CreatedAt:   now,
	}

	_, err = s.db.Exec(
		`INSERT INTO cron_jobs (id, name, cron_expr, description, enabled, next_run_at, created_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?)`,
		id, name, cronExpr, description, nextRun.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("persist job: %w", err)
	}

	s.jobs[id] = &registeredJob{Job: job, Func: fn}
	return id, nil
}

// RemoveJob removes a job by ID.
func (s *Scheduler) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.jobs, id)
	_, err := s.db.Exec("DELETE FROM cron_jobs WHERE id = ?", id)
	return err
}

// ListJobs returns all registered jobs.
func (s *Scheduler) ListJobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Job, 0, len(s.jobs))
	for _, rj := range s.jobs {
		result = append(result, rj.Job)
	}
	return result
}

// EnableJob enables or disables a job.
func (s *Scheduler) EnableJob(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rj, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	rj.Job.Enabled = enabled

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.db.Exec("UPDATE cron_jobs SET enabled = ? WHERE id = ?", enabledInt, id)
	return err
}

// Start begins the scheduler tick loop.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.tickLoop(ctx)
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.running = false
	s.mu.Unlock()

	s.wg.Wait()
}

// Close stops the scheduler and closes the database.
func (s *Scheduler) Close() error {
	s.Stop()
	return s.db.Close()
}

func (s *Scheduler) tickLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.tickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.checkAndRun(ctx, now)
		}
	}
}

func (s *Scheduler) checkAndRun(ctx context.Context, now time.Time) {
	s.mu.RLock()
	var dueJobs []*registeredJob
	for _, rj := range s.jobs {
		if rj.Job.Enabled && !rj.Job.NextRunAt.IsZero() && now.After(rj.Job.NextRunAt) {
			dueJobs = append(dueJobs, rj)
		}
	}
	s.mu.RUnlock()

	for _, rj := range dueJobs {
		s.runJob(ctx, rj)
	}
}

func (s *Scheduler) runJob(ctx context.Context, rj *registeredJob) {
	s.logger.Info("running cron job", "name", rj.Job.Name, "id", rj.Job.ID)

	if err := rj.Func(ctx); err != nil {
		s.logger.Error("cron job failed", "name", rj.Job.Name, "error", err)
	}

	now := time.Now()
	nextRun, err := nextCronTime(rj.Job.CronExpr, now)
	if err != nil {
		s.logger.Error("compute next run", "name", rj.Job.Name, "error", err)
		return
	}

	s.mu.Lock()
	rj.Job.LastRunAt = now
	rj.Job.NextRunAt = nextRun
	s.mu.Unlock()

	_, _ = s.db.Exec(
		"UPDATE cron_jobs SET last_run_at = ?, next_run_at = ? WHERE id = ?",
		now.Format(time.RFC3339), nextRun.Format(time.RFC3339), rj.Job.ID,
	)
}
