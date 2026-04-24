package builtin

import (
	"github.com/sadewadee/hera/internal/memory"
)

// SessionDBFromManager returns a SessionDB backed by the SQLiteProvider inside
// manager, or nil if manager is nil or its provider is not a *SQLiteProvider.
// Entrypoints use this to wire session_search without a direct SQLiteProvider ref.
func SessionDBFromManager(manager *memory.Manager) SessionDB {
	if manager == nil {
		return nil
	}
	p, ok := manager.UnderlyingProvider().(*memory.SQLiteProvider)
	if !ok {
		return nil
	}
	return NewSQLiteSessionDB(p)
}

// SQLiteSessionDB adapts *memory.SQLiteProvider to the SessionDB interface
// required by SessionSearchTool. It lives in the builtin package so it can
// reference both builtin types (SessionMatch, SessionMeta, etc.) and
// memory package types without creating a circular import.
type SQLiteSessionDB struct {
	p *memory.SQLiteProvider
}

// NewSQLiteSessionDB wraps a SQLiteProvider as a SessionDB.
func NewSQLiteSessionDB(p *memory.SQLiteProvider) *SQLiteSessionDB {
	return &SQLiteSessionDB{p: p}
}

// SearchMessages implements SessionDB.SearchMessages.
func (a *SQLiteSessionDB) SearchMessages(
	query string,
	roleFilter []string,
	excludeSources []string,
	limit, offset int,
) ([]SessionMatch, error) {
	raw, err := a.p.SearchMessages(query, roleFilter, excludeSources, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]SessionMatch, 0, len(raw))
	for _, r := range raw {
		out = append(out, SessionMatch{
			SessionID:      r.SessionID,
			Source:         r.Source,
			Model:          r.Model,
			SessionStarted: r.SessionStarted,
		})
	}
	return out, nil
}

// GetSession implements SessionDB.GetSession.
func (a *SQLiteSessionDB) GetSession(sessionID string) (*SessionMeta, error) {
	m, err := a.p.GetSessionMeta(sessionID)
	if err != nil {
		return nil, err
	}
	return &SessionMeta{
		ID:              m.ID,
		Source:          m.Source,
		ParentSessionID: m.ParentSessionID,
		StartedAt:       m.StartedAt,
	}, nil
}

// GetSessionMessages implements SessionDB.GetSessionMessages.
func (a *SQLiteSessionDB) GetSessionMessages(sessionID string) ([]SessionMessage, error) {
	raw, err := a.p.GetSessionMessages(sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionMessage, 0, len(raw))
	for _, r := range raw {
		out = append(out, SessionMessage{
			Role:     r.Role,
			Content:  r.Content,
			ToolName: r.ToolName,
		})
	}
	return out, nil
}

// ListSessionsRich implements SessionDB.ListSessionsRich.
func (a *SQLiteSessionDB) ListSessionsRich(limit int, excludeSources []string) ([]SessionListEntry, error) {
	raw, err := a.p.ListSessionsRich(limit, excludeSources)
	if err != nil {
		return nil, err
	}
	out := make([]SessionListEntry, 0, len(raw))
	for _, r := range raw {
		out = append(out, SessionListEntry{
			ID:           r.ID,
			Title:        r.Title,
			Source:       r.Source,
			StartedAt:    r.StartedAt,
			LastActive:   r.LastActive,
			MessageCount: r.MessageCount,
			Preview:      r.Preview,
		})
	}
	return out, nil
}
