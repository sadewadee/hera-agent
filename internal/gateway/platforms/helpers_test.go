package platforms

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MessageDeduplicator ---

func TestMessageDeduplicator_NewDefaults(t *testing.T) {
	d := NewMessageDeduplicator(0, 0)
	require.NotNil(t, d)
	assert.Equal(t, 2000, d.maxSize)
	assert.Equal(t, 5*time.Minute, d.ttl)
}

func TestMessageDeduplicator_FirstSeen(t *testing.T) {
	d := NewMessageDeduplicator(100, time.Minute)
	assert.False(t, d.IsDuplicate("msg-1"))
}

func TestMessageDeduplicator_SecondSeen(t *testing.T) {
	d := NewMessageDeduplicator(100, time.Minute)
	d.IsDuplicate("msg-1")
	assert.True(t, d.IsDuplicate("msg-1"))
}

func TestMessageDeduplicator_EmptyID(t *testing.T) {
	d := NewMessageDeduplicator(100, time.Minute)
	// Empty msgID is never duplicate.
	assert.False(t, d.IsDuplicate(""))
	assert.False(t, d.IsDuplicate(""))
}

func TestMessageDeduplicator_Clear(t *testing.T) {
	d := NewMessageDeduplicator(100, time.Minute)
	d.IsDuplicate("msg-1")
	d.Clear()
	assert.False(t, d.IsDuplicate("msg-1"))
}

func TestMessageDeduplicator_Concurrent(t *testing.T) {
	d := NewMessageDeduplicator(1000, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.IsDuplicate("shared-id")
		}()
	}
	wg.Wait()
}

// --- StripMarkdown ---

func TestStripMarkdown_Bold(t *testing.T) {
	assert.Equal(t, "hello world", StripMarkdown("**hello** world"))
}

func TestStripMarkdown_Italic(t *testing.T) {
	assert.Equal(t, "hello world", StripMarkdown("*hello* world"))
}

func TestStripMarkdown_InlineCode(t *testing.T) {
	assert.Equal(t, "hello world", StripMarkdown("`hello` world"))
}

func TestStripMarkdown_Heading(t *testing.T) {
	result := StripMarkdown("## My Title\ntext")
	assert.NotContains(t, result, "##")
	assert.Contains(t, result, "My Title")
}

func TestStripMarkdown_Link(t *testing.T) {
	result := StripMarkdown("[click here](https://example.com)")
	assert.Equal(t, "click here", result)
}

func TestStripMarkdown_Empty(t *testing.T) {
	assert.Equal(t, "", StripMarkdown(""))
}

func TestStripMarkdown_CodeBlock(t *testing.T) {
	text := "```go\nfmt.Println(\"hi\")\n```"
	result := StripMarkdown(text)
	assert.NotContains(t, result, "```")
}

// --- TextBatchAggregator ---

func TestTextBatchAggregator_IsEnabled(t *testing.T) {
	var _ bool
	handler := func(_, _ string) error { _ = true; return nil }
	agg := NewTextBatchAggregator(handler, 50*time.Millisecond, 200*time.Millisecond, 1000)
	assert.True(t, agg.IsEnabled())
}

func TestTextBatchAggregator_FlushesAfterDelay(t *testing.T) {
	done := make(chan string, 1)
	handler := func(key, text string) error {
		done <- text
		return nil
	}
	agg := NewTextBatchAggregator(handler, 50*time.Millisecond, 500*time.Millisecond, 1000)
	agg.Enqueue("key1", "hello world")

	select {
	case result := <-done:
		assert.Equal(t, "hello world", result)
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not _  within timeout")
	}
}

func TestTextBatchAggregator_BatchesMultiple(t *testing.T) {
	done := make(chan string, 1)
	handler := func(key, text string) error {
		done <- text
		return nil
	}
	agg := NewTextBatchAggregator(handler, 80*time.Millisecond, 500*time.Millisecond, 1000)
	agg.Enqueue("key1", "line1")
	agg.Enqueue("key1", "line2")

	select {
	case result := <-done:
		assert.Contains(t, result, "line1")
		assert.Contains(t, result, "line2")
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not _  within timeout")
	}
}

func TestTextBatchAggregator_CancelAll(t *testing.T) {
	var _ bool
	handler := func(_, _ string) error { _ = true; return nil }
	agg := NewTextBatchAggregator(handler, 200*time.Millisecond, 500*time.Millisecond, 1000)
	agg.Enqueue("key1", "text")
	agg.CancelAll()
	time.Sleep(300 * time.Millisecond)
	// CancelAll should not panic — that's the test.
	assert.True(t, true)
}

// --- ThreadParticipationTracker ---

func TestThreadParticipationTracker_MarkAndContains(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)
	tracker := NewThreadParticipationTracker("test", 100)

	assert.False(t, tracker.Contains("thread-1"))
	tracker.Mark("thread-1")
	assert.True(t, tracker.Contains("thread-1"))
}

func TestThreadParticipationTracker_Clear(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)
	tracker := NewThreadParticipationTracker("test", 100)
	tracker.Mark("thread-1")
	tracker.Clear()
	assert.False(t, tracker.Contains("thread-1"))
}

func TestThreadParticipationTracker_Persistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	tracker1 := NewThreadParticipationTracker("persist", 100)
	tracker1.Mark("thread-persist")

	// New instance should load from disk.
	tracker2 := NewThreadParticipationTracker("persist", 100)
	assert.True(t, tracker2.Contains("thread-persist"))
}

// --- RedactPhone ---

func TestRedactPhone_Empty(t *testing.T) {
	assert.Equal(t, "<none>", RedactPhone(""))
}

func TestRedactPhone_Short(t *testing.T) {
	result := RedactPhone("12345")
	assert.Contains(t, result, "****")
}

func TestRedactPhone_Long(t *testing.T) {
	result := RedactPhone("+14155552671")
	assert.Contains(t, result, "****")
	assert.Contains(t, result, "+141")
	assert.Contains(t, result, "2671")
}

func TestRedactPhone_VeryShort(t *testing.T) {
	result := RedactPhone("1234")
	assert.Equal(t, "****", result)
}
