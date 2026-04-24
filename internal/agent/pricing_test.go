package agent

import (
	"sync"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageTracker_NewUsageTracker(t *testing.T) {
	ut := NewUsageTracker()
	require.NotNil(t, ut)
	assert.Equal(t, float64(0), ut.TotalCost())
}

func TestUsageTracker_Track_AccumulatesTokens(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}, "gpt-4o")
	ut.Track("s1", llm.Usage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300}, "gpt-4o")

	sc := ut.GetSessionCost("s1")
	require.NotNil(t, sc)
	assert.Equal(t, 300, sc.PromptTokens)
	assert.Equal(t, 150, sc.CompletionTokens)
	assert.Equal(t, 450, sc.TotalTokens)
}

func TestUsageTracker_Track_CalculatesCost(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 1000, CompletionTokens: 1000, TotalTokens: 2000}, "gpt-4o")

	sc := ut.GetSessionCost("s1")
	require.NotNil(t, sc)
	assert.Greater(t, sc.CostUSD, float64(0))
}

func TestUsageTracker_Track_UnknownModel(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}, "unknown-model")

	sc := ut.GetSessionCost("s1")
	require.NotNil(t, sc)
	assert.Equal(t, float64(0), sc.CostUSD)
}

func TestUsageTracker_GetSessionCost_NotFound(t *testing.T) {
	ut := NewUsageTracker()
	sc := ut.GetSessionCost("nonexistent")
	assert.Nil(t, sc)
}

func TestUsageTracker_TotalCost_MultipleSessions(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500}, "gpt-4o")
	ut.Track("s2", llm.Usage{PromptTokens: 2000, CompletionTokens: 1000, TotalTokens: 3000}, "gpt-4o")

	total := ut.TotalCost()
	assert.Greater(t, total, float64(0))
}

func TestUsageTracker_TotalTokens(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}, "gpt-4o")
	ut.Track("s2", llm.Usage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300}, "gpt-4o")

	prompt, completion := ut.TotalTokens()
	assert.Equal(t, 300, prompt)
	assert.Equal(t, 150, completion)
}

func TestUsageTracker_Format(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}, "gpt-4o")

	formatted := ut.Format()
	assert.Contains(t, formatted, "Tokens:")
	assert.Contains(t, formatted, "Cost:")
}

func TestUsageTracker_Reset(t *testing.T) {
	ut := NewUsageTracker()
	ut.Track("s1", llm.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}, "gpt-4o")
	ut.Reset()

	assert.Equal(t, float64(0), ut.TotalCost())
	assert.Nil(t, ut.GetSessionCost("s1"))
}

func TestUsageTracker_ConcurrentAccess(t *testing.T) {
	ut := NewUsageTracker()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ut.Track("s1", llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}, "gpt-4o")
			_ = ut.TotalCost()
			_ = ut.Format()
		}(i)
	}
	wg.Wait()
}
