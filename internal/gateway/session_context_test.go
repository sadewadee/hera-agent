package gateway

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionContext(t *testing.T) {
	sc := NewSessionContext()
	require.NotNil(t, sc)
}

func TestSessionContext_SetAndGet(t *testing.T) {
	sc := NewSessionContext()
	sc.Set("key1", "value1")
	sc.Set("key2", 42)

	v1, ok := sc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v1)

	v2, ok := sc.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, v2)
}

func TestSessionContext_GetNotFound(t *testing.T) {
	sc := NewSessionContext()
	_, ok := sc.Get("missing")
	assert.False(t, ok)
}

func TestSessionContext_All(t *testing.T) {
	sc := NewSessionContext()
	sc.Set("a", 1)
	sc.Set("b", "two")

	all := sc.All()
	assert.Equal(t, 1, all["a"])
	assert.Equal(t, "two", all["b"])
}

func TestSessionContext_AllReturnsCopy(t *testing.T) {
	sc := NewSessionContext()
	sc.Set("key", "original")

	all := sc.All()
	all["key"] = "modified"

	v, _ := sc.Get("key")
	assert.Equal(t, "original", v)
}

func TestSessionContext_Overwrite(t *testing.T) {
	sc := NewSessionContext()
	sc.Set("key", "old")
	sc.Set("key", "new")
	v, _ := sc.Get("key")
	assert.Equal(t, "new", v)
}

func TestSessionContext_ConcurrentAccess(t *testing.T) {
	sc := NewSessionContext()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sc.Set("key", n)
			sc.Get("key")
			sc.All()
		}(i)
	}
	wg.Wait()
}
