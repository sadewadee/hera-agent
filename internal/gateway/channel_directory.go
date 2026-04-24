package gateway

import (
	"sync"
)

// ChannelEntry represents a known chat/channel with platform and metadata.
type ChannelEntry struct {
	Platform string `json:"platform"`
	ChatID   string `json:"chat_id"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "private", "group", "channel"
	Active   bool   `json:"active"`
}

// ChannelDirectory maintains a directory of all known channels across platforms.
// It is populated by gateway sessions and adapter events.
type ChannelDirectory struct {
	mu       sync.RWMutex
	channels map[string]*ChannelEntry // key: "platform:chatID"
}

// NewChannelDirectory creates an empty channel directory.
func NewChannelDirectory() *ChannelDirectory {
	return &ChannelDirectory{
		channels: make(map[string]*ChannelEntry),
	}
}

// channelKey builds a map key from platform and chatID.
func channelKey(platform, chatID string) string {
	return platform + ":" + chatID
}

// Register adds or updates a channel entry in the directory.
func (cd *ChannelDirectory) Register(entry ChannelEntry) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	key := channelKey(entry.Platform, entry.ChatID)
	cd.channels[key] = &entry
}

// Get retrieves a channel entry.
func (cd *ChannelDirectory) Get(platform, chatID string) (*ChannelEntry, bool) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	key := channelKey(platform, chatID)
	entry, ok := cd.channels[key]
	if !ok {
		return nil, false
	}
	cp := *entry
	return &cp, true
}

// ListByPlatform returns all channels for a given platform.
func (cd *ChannelDirectory) ListByPlatform(platform string) []ChannelEntry {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	var result []ChannelEntry
	for _, entry := range cd.channels {
		if entry.Platform == platform {
			result = append(result, *entry)
		}
	}
	return result
}

// ListAll returns all channels across all platforms.
func (cd *ChannelDirectory) ListAll() []ChannelEntry {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	result := make([]ChannelEntry, 0, len(cd.channels))
	for _, entry := range cd.channels {
		result = append(result, *entry)
	}
	return result
}

// Remove removes a channel from the directory.
func (cd *ChannelDirectory) Remove(platform, chatID string) bool {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	key := channelKey(platform, chatID)
	_, ok := cd.channels[key]
	if ok {
		delete(cd.channels, key)
	}
	return ok
}

// SetActive marks a channel as active or inactive.
func (cd *ChannelDirectory) SetActive(platform, chatID string, active bool) bool {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	key := channelKey(platform, chatID)
	entry, ok := cd.channels[key]
	if !ok {
		return false
	}
	entry.Active = active
	return true
}

// Count returns the total number of channels.
func (cd *ChannelDirectory) Count() int {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return len(cd.channels)
}

// UpdateFromSession updates the channel directory when a session is active.
// This is called by the session manager to keep the directory synchronized.
func (cd *ChannelDirectory) UpdateFromSession(session *GatewaySession) {
	if session == nil {
		return
	}
	cd.Register(ChannelEntry{
		Platform: session.Platform,
		ChatID:   session.ChatID,
		Type:     "private",
		Active:   true,
	})
}
