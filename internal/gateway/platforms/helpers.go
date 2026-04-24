// Package platforms provides platform adapter implementations for the gateway.
//
// helpers.go provides shared helper types used across multiple platform
// adapters: message deduplication, text batch aggregation, markdown
// stripping, thread participation tracking, and phone number redaction.
package platforms

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/paths"
)

// MessageDeduplicator provides TTL-based message deduplication.
type MessageDeduplicator struct {
	mu      sync.Mutex
	seen    map[string]time.Time
	maxSize int
	ttl     time.Duration
}

// NewMessageDeduplicator creates a new deduplicator with the given limits.
func NewMessageDeduplicator(maxSize int, ttl time.Duration) *MessageDeduplicator {
	if maxSize <= 0 {
		maxSize = 2000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &MessageDeduplicator{
		seen:    make(map[string]time.Time),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// IsDuplicate returns true if msgID was already seen within the TTL window.
func (d *MessageDeduplicator) IsDuplicate(msgID string) bool {
	if msgID == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if _, exists := d.seen[msgID]; exists {
		return true
	}
	d.seen[msgID] = now

	if len(d.seen) > d.maxSize {
		cutoff := now.Add(-d.ttl)
		for k, v := range d.seen {
			if v.Before(cutoff) {
				delete(d.seen, k)
			}
		}
	}
	return false
}

// Clear removes all tracked messages.
func (d *MessageDeduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = make(map[string]time.Time)
}

// TextBatchAggregator aggregates rapid-fire text events into single messages.
type TextBatchAggregator struct {
	mu             sync.Mutex
	batchDelay     time.Duration
	splitDelay     time.Duration
	splitThreshold int
	pending        map[string]*pendingBatch
	handler        func(key, text string) error
}

type pendingBatch struct {
	text         string
	lastChunkLen int
	timer        *time.Timer
}

// NewTextBatchAggregator creates a new text batch aggregator.
func NewTextBatchAggregator(handler func(key, text string) error, batchDelay, splitDelay time.Duration, splitThreshold int) *TextBatchAggregator {
	if batchDelay <= 0 {
		batchDelay = 600 * time.Millisecond
	}
	if splitDelay <= 0 {
		splitDelay = 2 * time.Second
	}
	if splitThreshold <= 0 {
		splitThreshold = 4000
	}
	return &TextBatchAggregator{
		batchDelay:     batchDelay,
		splitDelay:     splitDelay,
		splitThreshold: splitThreshold,
		pending:        make(map[string]*pendingBatch),
		handler:        handler,
	}
}

// IsEnabled returns true if batching is active.
func (a *TextBatchAggregator) IsEnabled() bool {
	return a.batchDelay > 0
}

// Enqueue adds text to the pending batch for the given key.
func (a *TextBatchAggregator) Enqueue(key, text string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	chunkLen := len(text)
	existing, ok := a.pending[key]
	if !ok {
		existing = &pendingBatch{}
		a.pending[key] = existing
	}

	if existing.text == "" {
		existing.text = text
	} else {
		existing.text = existing.text + "\n" + text
	}
	existing.lastChunkLen = chunkLen

	if existing.timer != nil {
		existing.timer.Stop()
	}

	delay := a.batchDelay
	if chunkLen >= a.splitThreshold {
		delay = a.splitDelay
	}

	existing.timer = time.AfterFunc(delay, func() {
		a.flush(key)
	})
}

func (a *TextBatchAggregator) flush(key string) {
	a.mu.Lock()
	batch, ok := a.pending[key]
	if ok {
		delete(a.pending, key)
	}
	a.mu.Unlock()

	if ok && batch.text != "" {
		if err := a.handler(key, batch.text); err != nil {
			slog.Error("TextBatchAggregator flush error",
				"key", key,
				"error", err,
			)
		}
	}
}

// CancelAll cancels all pending flush operations.
func (a *TextBatchAggregator) CancelAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, batch := range a.pending {
		if batch.timer != nil {
			batch.timer.Stop()
		}
	}
	a.pending = make(map[string]*pendingBatch)
}

// Markdown stripping regexes.
var (
	reBold         = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalicStar   = regexp.MustCompile(`\*(.+?)\*`)
	reBoldUnder    = regexp.MustCompile(`__(.+?)__`)
	reItalicUnder  = regexp.MustCompile(`_(.+?)_`)
	reCodeBlock    = regexp.MustCompile("```[a-zA-Z0-9_+-]*\n?")
	reInlineCode   = regexp.MustCompile("`(.+?)`")
	reHeading      = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reLink         = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
)

// StripMarkdown strips markdown formatting for plain-text platforms.
func StripMarkdown(text string) string {
	text = reBold.ReplaceAllString(text, "$1")
	text = reItalicStar.ReplaceAllString(text, "$1")
	text = reBoldUnder.ReplaceAllString(text, "$1")
	text = reItalicUnder.ReplaceAllString(text, "$1")
	text = reCodeBlock.ReplaceAllString(text, "")
	text = reInlineCode.ReplaceAllString(text, "$1")
	text = reHeading.ReplaceAllString(text, "")
	text = reLink.ReplaceAllString(text, "$1")
	text = reMultiNewline.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// ThreadParticipationTracker tracks threads the bot has participated in.
type ThreadParticipationTracker struct {
	mu         sync.Mutex
	platform   string
	threads    map[string]bool
	maxTracked int
	heraHome   string
}

// NewThreadParticipationTracker creates a new tracker for the given platform.
func NewThreadParticipationTracker(platform string, maxTracked int) *ThreadParticipationTracker {
	if maxTracked <= 0 {
		maxTracked = 500
	}
	t := &ThreadParticipationTracker{
		platform:   platform,
		threads:    make(map[string]bool),
		maxTracked: maxTracked,
		heraHome:   paths.HeraHome(),
	}
	t.load()
	return t
}

func (t *ThreadParticipationTracker) statePath() string {
	return filepath.Join(t.heraHome, fmt.Sprintf("%s_threads.json", t.platform))
}

func (t *ThreadParticipationTracker) load() {
	data, err := os.ReadFile(t.statePath())
	if err != nil {
		return
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return
	}
	for _, id := range ids {
		t.threads[id] = true
	}
}

func (t *ThreadParticipationTracker) save() {
	ids := make([]string, 0, len(t.threads))
	for id := range t.threads {
		ids = append(ids, id)
	}
	if len(ids) > t.maxTracked {
		ids = ids[len(ids)-t.maxTracked:]
		t.threads = make(map[string]bool, len(ids))
		for _, id := range ids {
			t.threads[id] = true
		}
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return
	}
	dir := filepath.Dir(t.statePath())
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(t.statePath(), data, 0o644)
}

// Mark records that the bot participated in the given thread.
func (t *ThreadParticipationTracker) Mark(threadID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.threads[threadID] {
		t.threads[threadID] = true
		t.save()
	}
}

// Contains returns true if the bot has participated in the given thread.
func (t *ThreadParticipationTracker) Contains(threadID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.threads[threadID]
}

// Clear removes all tracked threads.
func (t *ThreadParticipationTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.threads = make(map[string]bool)
}

// RedactPhone redacts a phone number for logging, preserving country code
// and last 4 digits.
func RedactPhone(phone string) string {
	if phone == "" {
		return "<none>"
	}
	if len(phone) <= 8 {
		if len(phone) > 4 {
			return phone[:2] + "****" + phone[len(phone)-2:]
		}
		return "****"
	}
	return phone[:4] + "****" + phone[len(phone)-4:]
}
