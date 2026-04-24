package gateway

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelDirectory_NewChannelDirectory(t *testing.T) {
	cd := NewChannelDirectory()
	require.NotNil(t, cd)
	assert.Equal(t, 0, cd.Count())
}

func TestChannelDirectory_Register(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{
		Platform: "telegram",
		ChatID:   "chat1",
		Title:    "Test Chat",
		Type:     "group",
		Active:   true,
	})
	assert.Equal(t, 1, cd.Count())
}

func TestChannelDirectory_Get(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{
		Platform: "telegram",
		ChatID:   "chat1",
		Title:    "Test Chat",
		Type:     "group",
		Active:   true,
	})

	entry, ok := cd.Get("telegram", "chat1")
	assert.True(t, ok)
	assert.Equal(t, "Test Chat", entry.Title)
}

func TestChannelDirectory_Get_NotFound(t *testing.T) {
	cd := NewChannelDirectory()
	entry, ok := cd.Get("telegram", "nonexistent")
	assert.False(t, ok)
	assert.Nil(t, entry)
}

func TestChannelDirectory_Get_ReturnsCopy(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{Platform: "telegram", ChatID: "c1", Title: "Original"})

	entry, _ := cd.Get("telegram", "c1")
	entry.Title = "Modified"

	original, _ := cd.Get("telegram", "c1")
	assert.Equal(t, "Original", original.Title)
}

func TestChannelDirectory_ListByPlatform(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{Platform: "telegram", ChatID: "c1"})
	cd.Register(ChannelEntry{Platform: "telegram", ChatID: "c2"})
	cd.Register(ChannelEntry{Platform: "discord", ChatID: "c3"})

	telegramChans := cd.ListByPlatform("telegram")
	assert.Len(t, telegramChans, 2)

	discordChans := cd.ListByPlatform("discord")
	assert.Len(t, discordChans, 1)
}

func TestChannelDirectory_ListByPlatform_Empty(t *testing.T) {
	cd := NewChannelDirectory()
	chans := cd.ListByPlatform("slack")
	assert.Empty(t, chans)
}

func TestChannelDirectory_ListAll(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{Platform: "telegram", ChatID: "c1"})
	cd.Register(ChannelEntry{Platform: "discord", ChatID: "c2"})

	all := cd.ListAll()
	assert.Len(t, all, 2)
}

func TestChannelDirectory_Remove(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{Platform: "telegram", ChatID: "c1"})

	ok := cd.Remove("telegram", "c1")
	assert.True(t, ok)
	assert.Equal(t, 0, cd.Count())
}

func TestChannelDirectory_Remove_NotFound(t *testing.T) {
	cd := NewChannelDirectory()
	ok := cd.Remove("telegram", "nonexistent")
	assert.False(t, ok)
}

func TestChannelDirectory_SetActive(t *testing.T) {
	cd := NewChannelDirectory()
	cd.Register(ChannelEntry{Platform: "telegram", ChatID: "c1", Active: true})

	ok := cd.SetActive("telegram", "c1", false)
	assert.True(t, ok)

	entry, _ := cd.Get("telegram", "c1")
	assert.False(t, entry.Active)
}

func TestChannelDirectory_SetActive_NotFound(t *testing.T) {
	cd := NewChannelDirectory()
	ok := cd.SetActive("telegram", "nonexistent", true)
	assert.False(t, ok)
}

func TestChannelDirectory_UpdateFromSession_NilSession(t *testing.T) {
	cd := NewChannelDirectory()
	cd.UpdateFromSession(nil) // should not panic
	assert.Equal(t, 0, cd.Count())
}

func TestChannelDirectory_UpdateFromSession(t *testing.T) {
	cd := NewChannelDirectory()
	session := &GatewaySession{
		Platform: "telegram",
		ChatID:   "chat1",
	}
	cd.UpdateFromSession(session)
	assert.Equal(t, 1, cd.Count())

	entry, ok := cd.Get("telegram", "chat1")
	assert.True(t, ok)
	assert.True(t, entry.Active)
}

func TestChannelDirectory_ConcurrentAccess(t *testing.T) {
	cd := NewChannelDirectory()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cd.Register(ChannelEntry{Platform: "test", ChatID: "c1"})
			cd.Get("test", "c1")
			cd.ListAll()
			cd.Count()
		}(i)
	}
	wg.Wait()
}
