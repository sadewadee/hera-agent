package mcp

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentStore_NewAttachmentStore(t *testing.T) {
	as := NewAttachmentStore()
	require.NotNil(t, as)
}

func TestAttachmentStore_Track(t *testing.T) {
	as := NewAttachmentStore()
	att := as.Track("session1", "image", "https://example.com/photo.jpg", "photo.jpg")
	require.NotNil(t, att)
	assert.NotEmpty(t, att.ID)
	assert.Equal(t, "session1", att.SessionID)
	assert.Equal(t, "image", att.Type)
	assert.Equal(t, "https://example.com/photo.jpg", att.URL)
	assert.Equal(t, "photo.jpg", att.Name)
	assert.False(t, att.Timestamp.IsZero())
}

func TestAttachmentStore_List(t *testing.T) {
	as := NewAttachmentStore()
	as.Track("s1", "image", "url1", "f1.jpg")
	as.Track("s1", "file", "url2", "f2.pdf")
	as.Track("s2", "audio", "url3", "f3.mp3")

	list := as.List("s1")
	assert.Len(t, list, 2)
}

func TestAttachmentStore_List_EmptySession(t *testing.T) {
	as := NewAttachmentStore()
	list := as.List("nonexistent")
	assert.Nil(t, list)
}

func TestAttachmentStore_List_ReturnsCopy(t *testing.T) {
	as := NewAttachmentStore()
	as.Track("s1", "image", "url1", "f1.jpg")

	list := as.List("s1")
	list[0].Name = "modified"

	original := as.List("s1")
	assert.Equal(t, "f1.jpg", original[0].Name)
}

func TestAttachmentStore_Count(t *testing.T) {
	as := NewAttachmentStore()
	assert.Equal(t, 0, as.Count("s1"))

	as.Track("s1", "image", "url1", "f1.jpg")
	as.Track("s1", "file", "url2", "f2.pdf")
	assert.Equal(t, 2, as.Count("s1"))
}

func TestAttachmentStore_Count_EmptySession(t *testing.T) {
	as := NewAttachmentStore()
	assert.Equal(t, 0, as.Count("nonexistent"))
}

func TestAttachmentStore_ConcurrentAccess(t *testing.T) {
	as := NewAttachmentStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			as.Track("s1", "image", "url", "file.jpg")
			_ = as.List("s1")
			_ = as.Count("s1")
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 50, as.Count("s1"))
}
