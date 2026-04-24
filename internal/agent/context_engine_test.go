package agent

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextEngine_NewContextEngine(t *testing.T) {
	ce := NewContextEngine(100000)
	require.NotNil(t, ce)
	assert.Equal(t, 100000, ce.Available())
}

func TestContextEngine_Allocate_Success(t *testing.T) {
	ce := NewContextEngine(1000)
	ok := ce.Allocate("system", 200)
	assert.True(t, ok)
	assert.Equal(t, 800, ce.Available())
}

func TestContextEngine_Allocate_ExceedsBudget(t *testing.T) {
	ce := NewContextEngine(1000)
	ce.Allocate("system", 800)
	ok := ce.Allocate("conversation", 300)
	assert.False(t, ok)
	assert.Equal(t, 200, ce.Available())
}

func TestContextEngine_Allocate_ExactBudget(t *testing.T) {
	ce := NewContextEngine(500)
	ok := ce.Allocate("all", 500)
	assert.True(t, ok)
	assert.Equal(t, 0, ce.Available())
}

func TestContextEngine_Allocate_ReplacesSameSource(t *testing.T) {
	ce := NewContextEngine(1000)
	ce.Allocate("system", 200)
	ce.Allocate("system", 300)
	// The second allocation replaces the first
	assert.Equal(t, 700, ce.Available())
}

func TestContextEngine_Usage_ReturnsAllSources(t *testing.T) {
	ce := NewContextEngine(10000)
	ce.Allocate("system", 100)
	ce.Allocate("tools", 200)
	ce.Allocate("memory", 300)

	usage := ce.Usage()
	assert.Len(t, usage, 3)

	names := make(map[string]int)
	for _, u := range usage {
		names[u.Name] = u.Tokens
	}
	assert.Equal(t, 100, names["system"])
	assert.Equal(t, 200, names["tools"])
	assert.Equal(t, 300, names["memory"])
}

func TestContextEngine_Usage_EmptyEngine(t *testing.T) {
	ce := NewContextEngine(1000)
	usage := ce.Usage()
	assert.Empty(t, usage)
}

func TestContextEngine_ConcurrentAccess(t *testing.T) {
	ce := NewContextEngine(100000)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ce.Allocate("source", 10)
			_ = ce.Available()
			_ = ce.Usage()
		}(i)
	}
	wg.Wait()
	// Should not panic or race
}
