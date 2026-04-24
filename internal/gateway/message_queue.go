package gateway

import (
	"context"
	"sync"
	"time"
)

// QueuedMessage wraps an incoming message with queue metadata.
type QueuedMessage struct {
	Message   IncomingMessage `json:"message"`
	EnqueueAt time.Time       `json:"enqueue_at"`
	Attempts  int             `json:"attempts"`
	ID        string          `json:"id"`
}

// MessageConsumer is called to process a message from the queue.
type MessageConsumer func(ctx context.Context, msg IncomingMessage) error

// MessageQueue provides an in-memory async message queue for gateway processing.
// Messages are processed by worker goroutines in FIFO order.
type MessageQueue struct {
	mu        sync.Mutex
	queue     []QueuedMessage
	consumer  MessageConsumer
	workers   int
	maxRetry  int
	wg        sync.WaitGroup
	signal    chan struct{}
	closed    bool
	nextID    int
	processed int64
	failed    int64
}

// NewMessageQueue creates a message queue with the given number of workers.
func NewMessageQueue(workers int, maxRetry int) *MessageQueue {
	if workers <= 0 {
		workers = 1
	}
	if maxRetry <= 0 {
		maxRetry = 3
	}
	return &MessageQueue{
		queue:    make([]QueuedMessage, 0, 100),
		workers:  workers,
		maxRetry: maxRetry,
		signal:   make(chan struct{}, workers),
	}
}

// SetConsumer sets the message processing function.
func (mq *MessageQueue) SetConsumer(fn MessageConsumer) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.consumer = fn
}

// Enqueue adds a message to the queue for async processing.
func (mq *MessageQueue) Enqueue(msg IncomingMessage) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.closed {
		return
	}

	mq.nextID++
	qm := QueuedMessage{
		Message:   msg,
		EnqueueAt: time.Now(),
		ID:        generateSessionID(), // reuse the helper from session.go
	}
	mq.queue = append(mq.queue, qm)

	// Signal a worker
	select {
	case mq.signal <- struct{}{}:
	default:
	}
}

// Start begins processing messages with worker goroutines.
func (mq *MessageQueue) Start(ctx context.Context) {
	for i := 0; i < mq.workers; i++ {
		mq.wg.Add(1)
		go mq.worker(ctx)
	}
}

// Stop signals workers to drain and waits for completion.
func (mq *MessageQueue) Stop() {
	mq.mu.Lock()
	mq.closed = true
	mq.mu.Unlock()

	close(mq.signal)
	mq.wg.Wait()
}

// Len returns the current number of queued messages.
func (mq *MessageQueue) Len() int {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.queue)
}

// Stats returns queue processing statistics.
func (mq *MessageQueue) Stats() (pending int, processed, failed int64) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.queue), mq.processed, mq.failed
}

func (mq *MessageQueue) worker(ctx context.Context) {
	defer mq.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-mq.signal:
			if !ok {
				// Drain remaining messages
				for {
					msg, found := mq.dequeue()
					if !found {
						return
					}
					mq.process(ctx, msg)
				}
			}
			msg, found := mq.dequeue()
			if !found {
				continue
			}
			mq.process(ctx, msg)
		}
	}
}

func (mq *MessageQueue) dequeue() (QueuedMessage, bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.queue) == 0 {
		return QueuedMessage{}, false
	}

	msg := mq.queue[0]
	mq.queue = mq.queue[1:]
	return msg, true
}

func (mq *MessageQueue) process(ctx context.Context, qm QueuedMessage) {
	mq.mu.Lock()
	consumer := mq.consumer
	mq.mu.Unlock()

	if consumer == nil {
		return
	}

	err := consumer(ctx, qm.Message)
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if err != nil {
		qm.Attempts++
		if qm.Attempts < mq.maxRetry {
			// Re-enqueue for retry
			mq.queue = append(mq.queue, qm)
		} else {
			mq.failed++
		}
	} else {
		mq.processed++
	}
}
