// Package batch implements the hera-batch prompt processor.
//
// A Batch reads prompts from a PromptSource, dispatches them to an Agent via
// HandleMessage, writes results via an OutputWriter, and checkpoints progress
// to SQLite so interrupted runs can be resumed with -resume <run-id>.
package batch

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// AgentRunner is the interface the batch runner calls per-prompt.
// *agent.Agent satisfies this interface via its HandleMessage method.
type AgentRunner interface {
	HandleMessage(ctx context.Context, platform, chatID, userID, text string) (string, error)
}

// PromptResult holds the outcome of processing one prompt.
type PromptResult struct {
	Index    int
	Prompt   string
	Response string
	Err      error
	Duration time.Duration
}

// Config holds all knobs for a batch run.
type Config struct {
	RunID         string        // unique run identifier; auto-generated if empty
	Concurrency   int           // worker goroutines (default 1)
	MaxRetries    int           // retry attempts on transient errors (default 3)
	PromptTimeout time.Duration // per-prompt timeout (0 = no timeout)
}

// Batch orchestrates reading, dispatching, and writing batch results.
type Batch struct {
	cfg      Config
	source   PromptSource
	writer   OutputWriter
	store    CheckpointStore
	progress ProgressReporter
	runner   AgentRunner
	logger   *slog.Logger
}

// New creates a Batch. All required fields must be non-nil.
func New(
	cfg Config,
	runner AgentRunner,
	source PromptSource,
	writer OutputWriter,
	store CheckpointStore,
	progress ProgressReporter,
) *Batch {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	return &Batch{
		cfg:      cfg,
		source:   source,
		writer:   writer,
		store:    store,
		progress: progress,
		runner:   runner,
		logger:   slog.Default(),
	}
}

// Run processes all prompts. It reads from source, dispatches to runner,
// writes results to writer, and checkpoints each outcome to store.
// On context cancellation, active workers are allowed to finish their
// current prompt; no new prompts are dispatched.
func (b *Batch) Run(ctx context.Context) error {
	prompts, err := b.source.Prompts()
	if err != nil {
		return fmt.Errorf("read prompts: %w", err)
	}

	// Filter out already-completed prompts for resume.
	var todo []indexedPrompt
	for i, p := range prompts {
		status, _ := b.store.GetStatus(b.cfg.RunID, i)
		if status == StatusCompleted {
			continue
		}
		todo = append(todo, indexedPrompt{index: i, text: p})
	}

	total := len(prompts)
	completed := total - len(todo)

	b.progress.Start(total, completed)
	defer b.progress.Finish()

	if len(todo) == 0 {
		b.logger.Info("batch: all prompts already completed", "run_id", b.cfg.RunID)
		return nil
	}

	// Worker pool: bounded by cfg.Concurrency.
	workCh := make(chan indexedPrompt, b.cfg.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var runErr error

	for i := 0; i < b.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range workCh {
				result := b.processOne(ctx, ip)

				if err := b.writer.Write(result); err != nil {
					mu.Lock()
					if runErr == nil {
						runErr = fmt.Errorf("write result[%d]: %w", ip.index, err)
					}
					mu.Unlock()
				}

				status := StatusCompleted
				errMsg := ""
				if result.Err != nil {
					status = StatusFailed
					errMsg = result.Err.Error()
				}
				if err := b.store.SetStatus(b.cfg.RunID, ip.index, ip.text, status, errMsg); err != nil {
					b.logger.Warn("batch: checkpoint write failed", "index", ip.index, "err", err)
				}

				b.progress.Increment()
			}
		}()
	}

	// Feed prompts; stop feeding on context cancellation.
	for _, ip := range todo {
		select {
		case <-ctx.Done():
			break
		case workCh <- ip:
		}
	}
	close(workCh)
	wg.Wait()

	if ctx.Err() != nil {
		return fmt.Errorf("batch interrupted: %w", ctx.Err())
	}
	return runErr
}

// indexedPrompt pairs a prompt with its original index in the source list.
type indexedPrompt struct {
	index int
	text  string
}

// processOne dispatches one prompt with retry logic.
func (b *Batch) processOne(ctx context.Context, ip indexedPrompt) PromptResult {
	start := time.Now()
	backoff := newExponentialBackoff(b.cfg.MaxRetries, time.Second, 30*time.Second)

	var (
		resp string
		err  error
	)

	for attempt := 0; attempt <= b.cfg.MaxRetries; attempt++ {
		pCtx := ctx
		var cancel context.CancelFunc
		if b.cfg.PromptTimeout > 0 {
			pCtx, cancel = context.WithTimeout(ctx, b.cfg.PromptTimeout)
		}

		resp, err = b.runner.HandleMessage(
			pCtx,
			"batch",                           // platform
			b.cfg.RunID,                       // chatID (run session)
			fmt.Sprintf("batch-%d", ip.index), // userID
			ip.text,
		)

		if cancel != nil {
			cancel()
		}

		if err == nil {
			break
		}

		if !isTransient(err) || attempt == b.cfg.MaxRetries {
			break
		}

		delay := backoff.Next()
		b.logger.Warn("batch: transient error, retrying",
			"index", ip.index,
			"attempt", attempt+1,
			"delay", delay,
			"err", err,
		)
		select {
		case <-ctx.Done():
			err = ctx.Err()
			goto done
		case <-time.After(delay):
		}
	}

done:
	return PromptResult{
		Index:    ip.index,
		Prompt:   ip.text,
		Response: resp,
		Err:      err,
		Duration: time.Since(start),
	}
}

// isTransient returns true for errors that warrant a retry.
// Currently checks for common transient error strings; extend as needed.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	transientPhrases := []string{
		"rate limit", "rate_limit", "too many requests",
		"timeout", "deadline exceeded",
		"connection refused", "connection reset",
		"server error", "503", "502", "429",
	}
	for _, p := range transientPhrases {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}
