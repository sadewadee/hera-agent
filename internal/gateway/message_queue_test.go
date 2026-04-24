package gateway

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageQueue(t *testing.T) {
	mq := NewMessageQueue(4, 3)
	require.NotNil(t, mq)
	assert.Equal(t, 4, mq.workers)
	assert.Equal(t, 3, mq.maxRetry)
}

func TestNewMessageQueue_Defaults(t *testing.T) {
	mq := NewMessageQueue(0, 0)
	assert.Equal(t, 1, mq.workers)
	assert.Equal(t, 3, mq.maxRetry)
}

func TestMessageQueue_EnqueueAndLen(t *testing.T) {
	mq := NewMessageQueue(1, 3)
	assert.Equal(t, 0, mq.Len())

	mq.Enqueue(IncomingMessage{Text: "hello"})
	assert.Equal(t, 1, mq.Len())

	mq.Enqueue(IncomingMessage{Text: "world"})
	assert.Equal(t, 2, mq.Len())
}

func TestMessageQueue_ProcessMessage(t *testing.T) {
	mq := NewMessageQueue(1, 3)
	var processed int32

	mq.SetConsumer(func(ctx context.Context, msg IncomingMessage) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	mq.Start(ctx)

	mq.Enqueue(IncomingMessage{Text: "test"})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	cancel()
	mq.Stop()

	assert.Equal(t, int32(1), atomic.LoadInt32(&processed))
}

func TestMessageQueue_Stats(t *testing.T) {
	mq := NewMessageQueue(1, 3)
	pending, processed, failed := mq.Stats()
	assert.Equal(t, 0, pending)
	assert.Equal(t, int64(0), processed)
	assert.Equal(t, int64(0), failed)
}

func TestMessageQueue_EnqueueAfterClose(t *testing.T) {
	mq := NewMessageQueue(1, 3)
	mq.SetConsumer(func(ctx context.Context, msg IncomingMessage) error { return nil })

	ctx := context.Background()
	mq.Start(ctx)
	mq.Stop()

	// Should not panic
	mq.Enqueue(IncomingMessage{Text: "after close"})
}
