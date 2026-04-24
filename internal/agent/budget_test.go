package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudget_NoLimits_AlwaysPasses(t *testing.T) {
	b := NewBudget(BudgetConfig{}) // zero limits = unlimited
	b.Record(1_000_000, 9999.0)
	assert.NoError(t, b.Check())
}

func TestBudget_TokenLimit_Enforced(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 100})
	b.Record(50, 0)
	assert.NoError(t, b.Check(), "50/100 tokens — should pass")

	b.Record(50, 0) // now at 100/100
	err := b.Check()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBudgetExceeded), "should be ErrBudgetExceeded")
	assert.Contains(t, err.Error(), "token")
}

func TestBudget_USDLimit_Enforced(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxUSD: 1.0})
	b.Record(0, 0.50)
	assert.NoError(t, b.Check())

	b.Record(0, 0.50) // now at $1.00
	err := b.Check()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBudgetExceeded))
	assert.Contains(t, err.Error(), "$")
}

func TestBudget_BothLimits_TokenHitsFirst(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 10, MaxUSD: 100.0})
	b.Record(10, 0.001)
	err := b.Check()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token")
}

func TestBudget_Window_ResetsCounters(t *testing.T) {
	// Use a very short window so we can test the reset in-process.
	b := NewBudget(BudgetConfig{MaxTokens: 50, Window: 10 * time.Millisecond})
	b.Record(50, 0) // hit limit

	err := b.Check()
	require.Error(t, err, "should be exceeded before window reset")

	// Wait for window to expire.
	time.Sleep(20 * time.Millisecond)

	// After window resets, counters are 0 again.
	assert.NoError(t, b.Check(), "budget should reset after window expiry")
	used, cost := b.Stats()
	assert.Equal(t, 0, used)
	assert.Equal(t, 0.0, cost)
}

func TestBudget_Stats(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 1000, MaxUSD: 10.0})
	b.Record(200, 0.30)
	b.Record(100, 0.15)

	tokens, cost := b.Stats()
	assert.Equal(t, 300, tokens)
	assert.InDelta(t, 0.45, cost, 0.0001)
}

func TestCostForTokens(t *testing.T) {
	// 1000 tokens at $0.01/1k = $0.01
	cost := costForTokens(1000, 0.01)
	assert.InDelta(t, 0.01, cost, 0.0001)

	// 500 tokens at $0.03/1k = $0.015
	cost = costForTokens(500, 0.03)
	assert.InDelta(t, 0.015, cost, 0.0001)
}

func TestBudget_ConcurrentAccess(t *testing.T) {
	b := NewBudget(BudgetConfig{MaxTokens: 10_000})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			b.Record(10, 0.001)
		}
	}()
	for i := 0; i < 100; i++ {
		_ = b.Check()
	}
	<-done
}
