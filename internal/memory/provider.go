package memory

import (
	"context"

	"github.com/sadewadee/hera/internal/llm"
)

// Provider is the interface for memory storage backends.
type Provider interface {
	SaveFact(ctx context.Context, userID, key, value string) error
	GetFacts(ctx context.Context, userID string) ([]Fact, error)
	Search(ctx context.Context, query string, opts SearchOpts) ([]MemoryResult, error)
	SaveConversation(ctx context.Context, sessionID string, messages []llm.Message) error
	GetConversation(ctx context.Context, sessionID string) ([]llm.Message, error)
	SessionSearch(ctx context.Context, query string) ([]SessionResult, error)

	// Typed notes (Harness-style auto-memory).
	SaveNote(ctx context.Context, note Note) error
	UpdateNote(ctx context.Context, userID, name, description, content string) error
	DeleteNote(ctx context.Context, userID, name string) error
	GetNote(ctx context.Context, userID, name string) (*Note, error)
	ListNotes(ctx context.Context, userID string, typ NoteType) ([]Note, error)

	// ListUserSessions returns metadata for the user's past sessions,
	// ordered by most recent first. Lightweight — does not load
	// message contents. Use GetConversation(sessionID) for full text.
	ListUserSessions(ctx context.Context, userID string, limit int) ([]SessionSummary, error)

	Close() error
}

// Summarizer generates summaries of conversations using an LLM.
type Summarizer interface {
	Summarize(ctx context.Context, messages []llm.Message) (string, error)
}

// Manager wraps a Provider with LLM-powered features like summarization.
type Manager struct {
	provider   Provider
	summarizer Summarizer
}

// NewManager creates a new memory manager.
func NewManager(provider Provider, summarizer Summarizer) *Manager {
	return &Manager{
		provider:   provider,
		summarizer: summarizer,
	}
}

// SaveFact delegates to the underlying provider.
func (m *Manager) SaveFact(ctx context.Context, userID, key, value string) error {
	return m.provider.SaveFact(ctx, userID, key, value)
}

// GetFacts delegates to the underlying provider.
func (m *Manager) GetFacts(ctx context.Context, userID string) ([]Fact, error) {
	return m.provider.GetFacts(ctx, userID)
}

// Search delegates to the underlying provider.
func (m *Manager) Search(ctx context.Context, query string, opts SearchOpts) ([]MemoryResult, error) {
	return m.provider.Search(ctx, query, opts)
}

// SaveConversation delegates to the underlying provider.
func (m *Manager) SaveConversation(ctx context.Context, sessionID string, messages []llm.Message) error {
	return m.provider.SaveConversation(ctx, sessionID, messages)
}

// GetConversation delegates to the underlying provider.
func (m *Manager) GetConversation(ctx context.Context, sessionID string) ([]llm.Message, error) {
	return m.provider.GetConversation(ctx, sessionID)
}

// SummarizeSession generates a summary for a session's conversation.
func (m *Manager) SummarizeSession(ctx context.Context, sessionID string) (*Summary, error) {
	messages, err := m.provider.GetConversation(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	content, err := m.summarizer.Summarize(ctx, messages)
	if err != nil {
		return nil, err
	}

	return &Summary{
		SessionID: sessionID,
		Content:   content,
	}, nil
}

// SessionSearch delegates to the underlying provider.
func (m *Manager) SessionSearch(ctx context.Context, query string) ([]SessionResult, error) {
	return m.provider.SessionSearch(ctx, query)
}

// SaveNote delegates to the underlying provider.
func (m *Manager) SaveNote(ctx context.Context, note Note) error {
	return m.provider.SaveNote(ctx, note)
}

// UpdateNote delegates to the underlying provider.
func (m *Manager) UpdateNote(ctx context.Context, userID, name, description, content string) error {
	return m.provider.UpdateNote(ctx, userID, name, description, content)
}

// DeleteNote delegates to the underlying provider.
func (m *Manager) DeleteNote(ctx context.Context, userID, name string) error {
	return m.provider.DeleteNote(ctx, userID, name)
}

// GetNote delegates to the underlying provider.
func (m *Manager) GetNote(ctx context.Context, userID, name string) (*Note, error) {
	return m.provider.GetNote(ctx, userID, name)
}

// ListNotes delegates to the underlying provider.
func (m *Manager) ListNotes(ctx context.Context, userID string, typ NoteType) ([]Note, error) {
	return m.provider.ListNotes(ctx, userID, typ)
}

// ListUserSessions delegates to the underlying provider.
func (m *Manager) ListUserSessions(ctx context.Context, userID string, limit int) ([]SessionSummary, error) {
	return m.provider.ListUserSessions(ctx, userID, limit)
}

// Close closes the underlying provider.
func (m *Manager) Close() error {
	return m.provider.Close()
}

// UnderlyingProvider returns the raw Provider backing this Manager.
// Callers may type-assert to *SQLiteProvider when they need access to
// methods not in the Provider interface (e.g. SearchMessages for SessionDB).
func (m *Manager) UnderlyingProvider() Provider {
	return m.provider
}
