package builtin

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBrowserState(t *testing.T) {
	bs := NewBrowserState()
	require.NotNil(t, bs)
	assert.Equal(t, 50, bs.maxHistory)
	assert.False(t, bs.IsActive())
}

func TestBrowserState_Update(t *testing.T) {
	bs := NewBrowserState()
	bs.Update("https://example.com", "Example")
	url, title := bs.Current()
	assert.Equal(t, "https://example.com", url)
	assert.Equal(t, "Example", title)
}

func TestBrowserState_UpdateAddsHistory(t *testing.T) {
	bs := NewBrowserState()
	bs.Update("https://a.com", "A")
	bs.Update("https://b.com", "B")
	history := bs.History()
	assert.Len(t, history, 2)
	assert.Equal(t, "https://a.com", history[0].URL)
	assert.Equal(t, "https://b.com", history[1].URL)
}

func TestBrowserState_UpdateSameURL(t *testing.T) {
	bs := NewBrowserState()
	bs.Update("https://a.com", "A")
	bs.Update("https://a.com", "A updated")
	history := bs.History()
	// Same URL should not add a new history entry
	assert.Len(t, history, 1)
}

func TestBrowserState_HistoryTruncation(t *testing.T) {
	bs := NewBrowserState()
	bs.maxHistory = 3
	for i := 0; i < 5; i++ {
		bs.Update("https://"+string(rune('a'+i))+".com", "")
	}
	history := bs.History()
	assert.Len(t, history, 3)
}

func TestBrowserState_SetLoading(t *testing.T) {
	bs := NewBrowserState()
	bs.SetLoading(true)
	assert.True(t, bs.isLoading)
	bs.SetLoading(false)
	assert.False(t, bs.isLoading)
}

func TestBrowserState_SetScreenshot(t *testing.T) {
	bs := NewBrowserState()
	bs.SetScreenshot("/tmp/screenshot.png")
	assert.Equal(t, "/tmp/screenshot.png", bs.lastScreenshot)
	assert.False(t, bs.lastScreenTime.IsZero())
}

func TestBrowserState_SetActive(t *testing.T) {
	bs := NewBrowserState()
	bs.Update("https://example.com", "Example")
	bs.SetActive(true)
	assert.True(t, bs.IsActive())

	bs.SetActive(false)
	assert.False(t, bs.IsActive())
	url, title := bs.Current()
	assert.Empty(t, url)
	assert.Empty(t, title)
}

func TestBrowserState_HistoryReturnsCopy(t *testing.T) {
	bs := NewBrowserState()
	bs.Update("https://a.com", "A")
	h1 := bs.History()
	h2 := bs.History()
	assert.Equal(t, h1, h2)
	// Modifying the returned slice should not affect internal state
	h1[0].URL = "modified"
	h3 := bs.History()
	assert.Equal(t, "https://a.com", h3[0].URL)
}

func TestBrowserState_ConsoleErrors(t *testing.T) {
	bs := NewBrowserState()
	bs.AddConsoleError("Error 1")
	bs.AddConsoleError("Error 2")
	errors := bs.ConsoleErrors()
	assert.Len(t, errors, 2)
	assert.Equal(t, "Error 1", errors[0])
}

func TestBrowserState_ConsoleErrorsTruncation(t *testing.T) {
	bs := NewBrowserState()
	for i := 0; i < 110; i++ {
		bs.AddConsoleError("error")
	}
	errors := bs.ConsoleErrors()
	assert.Len(t, errors, 100)
}

func TestBrowserState_ConsoleErrorsReturnsCopy(t *testing.T) {
	bs := NewBrowserState()
	bs.AddConsoleError("err")
	e1 := bs.ConsoleErrors()
	e1[0] = "modified"
	e2 := bs.ConsoleErrors()
	assert.Equal(t, "err", e2[0])
}

func TestBrowserState_Reset(t *testing.T) {
	bs := NewBrowserState()
	bs.Update("https://a.com", "A")
	bs.SetActive(true)
	bs.SetScreenshot("/tmp/ss.png")
	bs.AddConsoleError("err")

	bs.Reset()
	url, title := bs.Current()
	assert.Empty(t, url)
	assert.Empty(t, title)
	assert.False(t, bs.IsActive())
	assert.Empty(t, bs.History())
	assert.Empty(t, bs.ConsoleErrors())
}

func TestBrowserState_ConcurrentAccess(t *testing.T) {
	bs := NewBrowserState()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bs.Update("https://"+string(rune('a'+n%26))+".com", "title")
			bs.Current()
			bs.History()
			bs.SetLoading(true)
			bs.AddConsoleError("err")
			bs.ConsoleErrors()
		}(i)
	}
	wg.Wait()
}
