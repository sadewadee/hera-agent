package gateway

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStickerCache_NewStickerCache(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	require.NotNil(t, sc)
	assert.Equal(t, 0, sc.Size())
}

func TestStickerCache_DefaultValues(t *testing.T) {
	sc := NewStickerCache(0, 0)
	assert.Equal(t, 1000, sc.maxSize)
	assert.Equal(t, 24*time.Hour, sc.ttl)
}

func TestStickerCache_PutAndGet(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	data := []byte("sticker-data")
	sc.Put("telegram", "sticker1", data)

	got, ok := sc.Get("telegram", "sticker1")
	assert.True(t, ok)
	assert.Equal(t, data, got)
}

func TestStickerCache_Get_NotFound(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	_, ok := sc.Get("telegram", "nonexistent")
	assert.False(t, ok)
}

func TestStickerCache_Get_Expired(t *testing.T) {
	sc := NewStickerCache(100, 1*time.Millisecond)
	sc.Put("telegram", "sticker1", []byte("data"))
	time.Sleep(10 * time.Millisecond)

	_, ok := sc.Get("telegram", "sticker1")
	assert.False(t, ok)
}

func TestStickerCache_Size(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	sc.Put("telegram", "s1", []byte("a"))
	sc.Put("telegram", "s2", []byte("b"))
	sc.Put("discord", "s3", []byte("c"))
	assert.Equal(t, 3, sc.Size())
}

func TestStickerCache_Put_EvictsAtCapacity(t *testing.T) {
	sc := NewStickerCache(2, time.Hour)
	sc.Put("t", "s1", []byte("a"))
	sc.Put("t", "s2", []byte("b"))
	sc.Put("t", "s3", []byte("c")) // should evict oldest

	assert.LessOrEqual(t, sc.Size(), 2)
}

func TestStickerCache_Put_EvictsExpiredFirst(t *testing.T) {
	sc := NewStickerCache(3, 1*time.Millisecond)
	sc.Put("t", "s1", []byte("a"))
	sc.Put("t", "s2", []byte("b"))
	time.Sleep(10 * time.Millisecond)
	sc.Put("t", "s3", []byte("c")) // should evict expired

	assert.Equal(t, 1, sc.Size())
}

func TestStickerCache_Clear(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	sc.Put("t", "s1", []byte("a"))
	sc.Put("t", "s2", []byte("b"))
	sc.Clear()
	assert.Equal(t, 0, sc.Size())
}

func TestStickerCache_OverwritesSameKey(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	sc.Put("t", "s1", []byte("old"))
	sc.Put("t", "s1", []byte("new"))

	got, ok := sc.Get("t", "s1")
	assert.True(t, ok)
	assert.Equal(t, []byte("new"), got)
}

func TestStickerCache_DifferentPlatformsSameID(t *testing.T) {
	sc := NewStickerCache(100, time.Hour)
	sc.Put("telegram", "s1", []byte("telegram-data"))
	sc.Put("discord", "s1", []byte("discord-data"))

	d1, ok := sc.Get("telegram", "s1")
	assert.True(t, ok)
	assert.Equal(t, []byte("telegram-data"), d1)

	d2, ok := sc.Get("discord", "s1")
	assert.True(t, ok)
	assert.Equal(t, []byte("discord-data"), d2)
}
