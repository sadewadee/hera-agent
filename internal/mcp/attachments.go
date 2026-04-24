package mcp

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Attachment represents a media attachment tracked for a session.
type Attachment struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"` // "image", "file", "audio", "video"
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	Size      int64     `json:"size,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// AttachmentStore tracks media attachments associated with sessions.
// It is safe for concurrent use.
type AttachmentStore struct {
	mu    sync.Mutex
	items map[string][]Attachment // sessionID -> attachments
}

// NewAttachmentStore creates an empty attachment store.
func NewAttachmentStore() *AttachmentStore {
	return &AttachmentStore{
		items: make(map[string][]Attachment),
	}
}

// Track records a new attachment for a session.
func (as *AttachmentStore) Track(sessionID, mediaType, url, name string) *Attachment {
	att := Attachment{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Type:      mediaType,
		URL:       url,
		Name:      name,
		Timestamp: time.Now(),
	}

	as.mu.Lock()
	as.items[sessionID] = append(as.items[sessionID], att)
	as.mu.Unlock()

	return &att
}

// List returns all attachments for a session.
func (as *AttachmentStore) List(sessionID string) []Attachment {
	as.mu.Lock()
	defer as.mu.Unlock()

	atts, ok := as.items[sessionID]
	if !ok {
		return nil
	}

	// Return a copy.
	result := make([]Attachment, len(atts))
	copy(result, atts)
	return result
}

// Count returns the number of attachments for a session.
func (as *AttachmentStore) Count(sessionID string) int {
	as.mu.Lock()
	defer as.mu.Unlock()
	return len(as.items[sessionID])
}
