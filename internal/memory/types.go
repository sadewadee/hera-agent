package memory

import "time"

// Fact represents a persistent fact about a user.
type Fact struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoryResult represents a search result from memory.
type MemoryResult struct {
	Content  string  `json:"content"`
	Source   string  `json:"source"` // "fact", "conversation", "summary"
	SourceID string  `json:"source_id"`
	Score    float64 `json:"score"`
}

// Summary represents an LLM-generated conversation summary.
type Summary struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionResult represents a session search result.
type SessionResult struct {
	SessionID string    `json:"session_id"`
	Preview   string    `json:"preview"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchOpts configures a memory search.
type SearchOpts struct {
	Limit  int    `json:"limit"`
	Source string `json:"source,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// NoteType categorizes a memory note. The four types follow the same
// semantic taxonomy as the Hera harness auto-memory system:
//   - user: who the user is — role, goals, responsibilities, knowledge
//   - feedback: durable guidance from the user about how to work
//   - project: ongoing work, goals, incidents, deadlines
//   - reference: pointers to external systems (Slack channel, dashboard, issue tracker)
type NoteType string

const (
	NoteTypeUser      NoteType = "user"
	NoteTypeFeedback  NoteType = "feedback"
	NoteTypeProject   NoteType = "project"
	NoteTypeReference NoteType = "reference"
)

// ValidNoteType reports whether t is one of the recognised note types.
func ValidNoteType(t NoteType) bool {
	switch t {
	case NoteTypeUser, NoteTypeFeedback, NoteTypeProject, NoteTypeReference:
		return true
	}
	return false
}

// Note is a typed memory entry. Name is unique per user and is used as
// the addressable handle (e.g. for update or delete). Description is a
// short hook shown in listings; Content holds the full body.
type Note struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Type        NoteType  `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SessionSummary is a lightweight metadata record for a past session,
// used for listing and browsing. It does not carry the actual message
// contents — call GetConversation(SessionID) when you need those.
type SessionSummary struct {
	SessionID    string    `json:"session_id"`
	FirstMessage time.Time `json:"first_message"`
	LastMessage  time.Time `json:"last_message"`
	MessageCount int       `json:"message_count"`
	Preview      string    `json:"preview"` // first user message, truncated to ~120 chars
}
