package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SearchMessagesResult is one FTS5 match from the conversations table.
// Fields mirror builtin.SessionMatch but use only stdlib types so that the
// memory package stays free of the builtin import (which would be circular).
type SearchMessagesResult struct {
	SessionID      string
	Source         string
	Model          string
	SessionStarted int64
}

// SearchMessages searches conversation history via FTS5 on memory_fts.
// roleFilter restricts to the given roles (empty = all roles; currently unused
// in the FTS query because FTS5 doesn't filter by role column, but reserved
// for post-filter use by callers).
// excludeSources excludes sessions whose platform portion matches any entry.
// Returns up to limit matches starting from offset, ordered by FTS5 rank.
func (p *SQLiteProvider) SearchMessages(
	query string,
	_ []string, // roleFilter — reserved, not yet used in FTS query
	excludeSources []string,
	limit, offset int,
) ([]SearchMessagesResult, error) {
	if limit <= 0 {
		limit = 50
	}
	safeQuery := sanitizeFTSQuery(query)
	if safeQuery == "" {
		return nil, nil
	}

	// source_id format: "{sessionID}:{index}" per SaveConversation.
	// Fetch source_id as-is; strip the trailing ":{N}" suffix in Go
	// (pure-Go SQLite driver does not support reverse()).
	rows, err := p.db.QueryContext(context.Background(),
		`SELECT source_id, content, rank
		 FROM memory_fts
		 WHERE memory_fts MATCH ?
		   AND source = 'conversation'
		 ORDER BY rank
		 LIMIT ? OFFSET ?`,
		safeQuery, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	excludeSet := make(map[string]bool, len(excludeSources))
	for _, s := range excludeSources {
		excludeSet[s] = true
	}

	var results []SearchMessagesResult
	for rows.Next() {
		var sourceID, content string
		var rank float64
		if err := rows.Scan(&sourceID, &content, &rank); err != nil {
			return nil, fmt.Errorf("scan match: %w", err)
		}

		// Strip the trailing ":{index}" to recover the sessionID.
		sessionID := stripLastColonSuffix(sourceID)

		// Extract source (platform) from "platform:userID:uuid".
		source := sessionIDPlatform(sessionID)
		if excludeSet[source] {
			continue
		}

		results = append(results, SearchMessagesResult{
			SessionID: sessionID,
			Source:    source,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}

// GetSessionMeta returns metadata for a single session from the conversations table.
func (p *SQLiteProvider) GetSessionMeta(sessionID string) (*SessionMetaInfo, error) {
	var minTS string
	err := p.db.QueryRowContext(context.Background(),
		`SELECT MIN(timestamp) FROM conversations WHERE session_id = ?`, sessionID,
	).Scan(&minTS)
	if err == sql.ErrNoRows || minTS == "" {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("query session meta: %w", err)
	}

	t := parseSQLiteTime(minTS)
	return &SessionMetaInfo{
		ID:        sessionID,
		Source:    sessionIDPlatform(sessionID),
		StartedAt: t.Unix(),
	}, nil
}

// SessionMetaInfo holds lightweight metadata about a session.
type SessionMetaInfo struct {
	ID              string
	Source          string
	ParentSessionID string
	StartedAt       int64
}

// GetSessionMessages returns all messages for a session ordered by timestamp.
func (p *SQLiteProvider) GetSessionMessages(sessionID string) ([]SessionMessageInfo, error) {
	rows, err := p.db.QueryContext(context.Background(),
		`SELECT role, content, COALESCE(name, '')
		 FROM conversations
		 WHERE session_id = ?
		 ORDER BY timestamp ASC, id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var msgs []SessionMessageInfo
	for rows.Next() {
		var m SessionMessageInfo
		if err := rows.Scan(&m.Role, &m.Content, &m.ToolName); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return msgs, nil
}

// SessionMessageInfo holds a single message retrieved by GetSessionMessages.
type SessionMessageInfo struct {
	Role     string
	Content  string
	ToolName string
}

// ListSessionsRich returns up to limit recent sessions with metadata, excluding
// sessions whose platform source appears in excludeSources.
func (p *SQLiteProvider) ListSessionsRich(limit int, excludeSources []string) ([]SessionRichEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	// Over-fetch to account for post-filter exclusions.
	fetchLimit := limit * 3

	rows, err := p.db.QueryContext(context.Background(),
		`SELECT
			session_id,
			MIN(timestamp) AS first_ts,
			MAX(timestamp) AS last_ts,
			COUNT(*)       AS msg_count,
			(SELECT content FROM conversations c2
			   WHERE c2.session_id = conversations.session_id
			     AND c2.role = 'user'
			   ORDER BY c2.timestamp ASC LIMIT 1) AS preview
		 FROM conversations
		 GROUP BY session_id
		 ORDER BY last_ts DESC
		 LIMIT ?`,
		fetchLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	excludeSet := make(map[string]bool, len(excludeSources))
	for _, s := range excludeSources {
		excludeSet[s] = true
	}

	var results []SessionRichEntry
	for rows.Next() {
		var sessionID, firstTS, lastTS string
		var msgCount int
		var preview sql.NullString
		if err := rows.Scan(&sessionID, &firstTS, &lastTS, &msgCount, &preview); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		source := sessionIDPlatform(sessionID)
		if excludeSet[source] {
			continue
		}

		first := parseSQLiteTime(firstTS)
		last := parseSQLiteTime(lastTS)

		previewStr := ""
		if preview.Valid {
			previewStr = preview.String
			if len(previewStr) > 120 {
				previewStr = previewStr[:117] + "..."
			}
		}

		results = append(results, SessionRichEntry{
			ID:           sessionID,
			Source:       source,
			StartedAt:    first.Format(time.RFC3339),
			LastActive:   last.Format(time.RFC3339),
			MessageCount: msgCount,
			Preview:      previewStr,
		})

		if len(results) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}

// SessionRichEntry is one entry in a ListSessionsRich response.
type SessionRichEntry struct {
	ID           string
	Title        string
	Source       string
	StartedAt    string
	LastActive   string
	MessageCount int
	Preview      string
}

// sessionIDPlatform extracts the platform portion from "platform:userID:uuid".
func sessionIDPlatform(sessionID string) string {
	for i, c := range sessionID {
		if c == ':' {
			return sessionID[:i]
		}
	}
	return ""
}

// stripLastColonSuffix removes the last ":{suffix}" segment from s.
// Used to recover sessionID from source_id format "sessionID:{index}".
func stripLastColonSuffix(s string) string {
	// Walk from the end to find the last colon.
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return s[:i]
		}
	}
	return s // no colon found, return as-is
}
