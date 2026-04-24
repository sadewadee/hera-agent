package batch

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressReporter tracks and displays batch run progress.
type ProgressReporter interface {
	// Start initialises the reporter with total prompt count and how many were
	// already completed (from a resumed run).
	Start(total, alreadyDone int)
	// Increment marks one more prompt as done.
	Increment()
	// Finish writes a final newline / summary and releases resources.
	Finish()
}

// TTYProgress renders an in-place progress bar to w (typically os.Stderr).
// It is safe for concurrent calls from multiple goroutines.
type TTYProgress struct {
	mu        sync.Mutex
	w         io.Writer
	total     int
	done      int
	startTime time.Time
	width     int // bar width in chars
}

// NewTTYProgress returns a TTYProgress writing to w.
// width is the number of fill characters in the bar (default 30 if <=0).
func NewTTYProgress(w io.Writer, width int) *TTYProgress {
	if width <= 0 {
		width = 30
	}
	return &TTYProgress{w: w, width: width}
}

// NewStderrProgress is a convenience constructor that writes to os.Stderr.
func NewStderrProgress() *TTYProgress {
	return NewTTYProgress(os.Stderr, 30)
}

func (p *TTYProgress) Start(total, alreadyDone int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
	p.done = alreadyDone
	p.startTime = time.Now()
	p.render()
}

func (p *TTYProgress) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	p.render()
}

func (p *TTYProgress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Overwrite the progress line with a final summary.
	elapsed := time.Since(p.startTime).Round(time.Second)
	fmt.Fprintf(p.w, "\r%-80s\r", "") // clear line
	fmt.Fprintf(p.w, "Done: %d/%d in %s\n", p.done, p.total, elapsed)
}

// render writes the current bar in-place (no trailing newline).
// Caller must hold p.mu.
func (p *TTYProgress) render() {
	if p.total == 0 {
		return
	}
	pct := float64(p.done) / float64(p.total)
	filled := int(pct * float64(p.width))
	if filled > p.width {
		filled = p.width
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", p.width-filled)

	eta := ""
	if p.done > 0 {
		elapsed := time.Since(p.startTime)
		perItem := elapsed / time.Duration(p.done)
		remaining := perItem * time.Duration(p.total-p.done)
		eta = " ETA " + fmtDuration(remaining)
	}

	fmt.Fprintf(p.w, "\r[%s] %d/%d%s", bar, p.done, p.total, eta)
}

// fmtDuration formats d as "Xm Ys" omitting zero components.
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// NoopProgress is a ProgressReporter that does nothing. Used in non-TTY
// contexts or when progress output is suppressed (e.g., piped output).
type NoopProgress struct{}

func (NoopProgress) Start(_, _ int) {}
func (NoopProgress) Increment()     {}
func (NoopProgress) Finish()        {}
