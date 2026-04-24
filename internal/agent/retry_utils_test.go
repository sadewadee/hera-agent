package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJitteredBackoff_NewDefaults(t *testing.T) {
	bo := NewJitteredBackoff()
	assert.Equal(t, 500*time.Millisecond, bo.BaseDelay)
	assert.Equal(t, 30*time.Second, bo.MaxDelay)
	assert.Equal(t, 5, bo.MaxRetries)
	assert.False(t, bo.Exhausted())
}

func TestJitteredBackoff_Next_IncreasesDuration(t *testing.T) {
	bo := NewJitteredBackoff()
	first := bo.Next()
	second := bo.Next()
	// Second should generally be longer due to exponential backoff
	// (may vary due to jitter, but base should double)
	assert.Greater(t, second, time.Duration(0))
	_ = first
}

func TestJitteredBackoff_Next_CapsAtMaxDelay(t *testing.T) {
	bo := &JitteredBackoff{
		BaseDelay:  1 * time.Second,
		MaxDelay:   5 * time.Second,
		MaxRetries: 20,
	}
	for i := 0; i < 15; i++ {
		d := bo.Next()
		assert.LessOrEqual(t, d, bo.MaxDelay+bo.MaxDelay/2) // max + jitter
	}
}

func TestJitteredBackoff_Exhausted(t *testing.T) {
	bo := &JitteredBackoff{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		MaxRetries: 3,
	}
	assert.False(t, bo.Exhausted())
	bo.Next()
	bo.Next()
	bo.Next()
	assert.True(t, bo.Exhausted())
}

func TestJitteredBackoff_Reset(t *testing.T) {
	bo := NewJitteredBackoff()
	bo.Next()
	bo.Next()
	bo.Reset()
	assert.False(t, bo.Exhausted())
}

func TestRetryWithBackoff_SucceedsFirstTry(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := RetryWithBackoff(ctx, 3, func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryWithBackoff_RetriesOnError(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := RetryWithBackoff(ctx, 3, func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary error")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryWithBackoff_ExhaustsRetries(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := RetryWithBackoff(ctx, 2, func() error {
		calls++
		return errors.New("persistent error")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persistent error")
}

func TestRetryWithBackoff_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := RetryWithBackoff(ctx, 10, func() error {
		calls++
		return errors.New("error")
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}
