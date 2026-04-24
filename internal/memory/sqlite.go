package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sadewadee/hera/internal/llm"

	_ "modernc.org/sqlite"
)

// SQLiteProvider implements the Provider interface using SQLite with FTS5.
type SQLiteProvider struct {
	db *sql.DB
}

// NewSQLiteProvider creates a new SQLite-backed memory provider.
// It creates the database file and all required tables on init.
func NewSQLiteProvider(dbPath string) (*SQLiteProvider, error) {
	// Create parent directories if needed.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode for concurrency.
	if _, err := db.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	// Set busy timeout to avoid "database is locked" under concurrency.
	if _, err := db.ExecContext(context.Background(), "PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	p := &SQLiteProvider{db: db}
	if err := p.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return p, nil
}

func (p *SQLiteProvider) createTables() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS facts (
			id         TEXT PRIMARY KEY,
			user_id    TEXT NOT NULL,
			key        TEXT NOT NULL,
			value      TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_user_id ON facts(user_id)`,

		`CREATE TABLE IF NOT EXISTS conversations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role       TEXT NOT NULL,
			content    TEXT NOT NULL,
			name       TEXT,
			tool_calls TEXT,
			tool_call_id TEXT,
			timestamp  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_session_id ON conversations(session_id)`,

		`CREATE TABLE IF NOT EXISTS summaries (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			content    TEXT NOT NULL,
			model      TEXT,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_session_id ON summaries(session_id)`,

		`CREATE TABLE IF NOT EXISTS user_profiles (
			user_id    TEXT PRIMARY KEY,
			data       TEXT NOT NULL DEFAULT '{}',
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		// FTS5 virtual table for full-text search across facts and conversations.
		`CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
			source,
			source_id,
			user_id,
			content,
			tokenize='porter unicode61'
		)`,

		// Typed memory notes (Harness-style auto-memory).
		`CREATE TABLE IF NOT EXISTS notes (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			type        TEXT NOT NULL,
			name        TEXT NOT NULL,
			description TEXT NOT NULL,
			content     TEXT NOT NULL,
			created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_user ON notes(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_user_type ON notes(user_id, type)`,
	}

	for _, stmt := range statements {
		if _, err := p.db.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 60)], err)
		}
	}
	return nil
}

// SaveFact persists a fact about a user. Upserts on (user_id, key).
func (p *SQLiteProvider) SaveFact(ctx context.Context, userID, key, value string) error {
	id := uuid.New().String()
	now := time.Now().UTC()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Upsert into facts table.
	res, err := tx.ExecContext(ctx,
		`INSERT INTO facts (id, user_id, key, value, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, key) DO UPDATE SET
		   value = excluded.value,
		   updated_at = excluded.updated_at`,
		id, userID, key, value, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert fact: %w", err)
	}

	// Determine the actual ID (might be existing row on conflict).
	rowsAffected, _ := res.RowsAffected()
	actualID := id
	if rowsAffected > 0 {
		// Check if it was an insert or update by querying the actual ID.
		row := tx.QueryRowContext(ctx,
			`SELECT id FROM facts WHERE user_id = ? AND key = ?`, userID, key)
		if err := row.Scan(&actualID); err != nil {
			return fmt.Errorf("get fact id: %w", err)
		}
	}

	// Update FTS index: delete old entry if exists, then insert new one.
	_, _ = tx.ExecContext(ctx,
		`DELETE FROM memory_fts WHERE source = 'fact' AND source_id IN
		  (SELECT id FROM facts WHERE user_id = ? AND key = ?)`,
		userID, key,
	)
	ftsContent := fmt.Sprintf("%s: %s", key, value)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO memory_fts (source, source_id, user_id, content)
		 VALUES ('fact', ?, ?, ?)`,
		actualID, userID, ftsContent,
	)
	if err != nil {
		return fmt.Errorf("update fts: %w", err)
	}

	return tx.Commit()
}

// GetFacts returns all facts for a user.
func (p *SQLiteProvider) GetFacts(ctx context.Context, userID string) ([]Fact, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, user_id, key, value, created_at, updated_at
		 FROM facts WHERE user_id = ? ORDER BY key`, userID)
	if err != nil {
		return nil, fmt.Errorf("query facts: %w", err)
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.UserID, &f.Key, &f.Value, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		facts = append(facts, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return facts, nil
}

// sanitizeFTSQuery escapes FTS5 special characters in user input to prevent
// query injection. FTS5 operators like OR, AND, NOT, NEAR, *, ^, and quotes
// are stripped or escaped so user input is always treated as literal terms.
func sanitizeFTSQuery(query string) string {
	if query == "" {
		return query
	}

	// Remove FTS5 special characters that could alter query semantics.
	var b strings.Builder
	for _, r := range query {
		switch r {
		case '"', '\'', '*', '^', '(', ')', '{', '}', ':', '+':
			// Skip FTS5 operators
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}

	sanitized := strings.TrimSpace(b.String())

	// Wrap each word in double quotes to force literal matching,
	// which neutralizes FTS5 keywords (OR, AND, NOT, NEAR).
	words := strings.Fields(sanitized)
	if len(words) == 0 {
		return ""
	}

	var quoted []string
	for _, w := range words {
		// Skip FTS5 keywords when they appear as standalone words.
		upper := strings.ToUpper(w)
		if upper == "OR" || upper == "AND" || upper == "NOT" || upper == "NEAR" {
			// Quote the keyword to treat it as a literal search term.
			quoted = append(quoted, "\""+w+"\"")
		} else {
			quoted = append(quoted, "\""+w+"\"")
		}
	}

	return strings.Join(quoted, " ")
}

// Search performs a full-text search across facts and conversations.
func (p *SQLiteProvider) Search(ctx context.Context, query string, opts SearchOpts) ([]MemoryResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Sanitize user input to prevent FTS5 query injection.
	safeQuery := sanitizeFTSQuery(query)
	if safeQuery == "" {
		return nil, nil
	}

	// Build query based on filters.
	baseQuery := `SELECT source, source_id, content, rank
		FROM memory_fts WHERE memory_fts MATCH ?`
	args := []any{safeQuery}

	if opts.UserID != "" {
		baseQuery += ` AND user_id = ?`
		args = append(args, opts.UserID)
	}
	if opts.Source != "" {
		baseQuery += ` AND source = ?`
		args = append(args, opts.Source)
	}

	baseQuery += ` ORDER BY rank LIMIT ?`
	args = append(args, limit)

	rows, err := p.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var results []MemoryResult
	for rows.Next() {
		var r MemoryResult
		var rank float64
		if err := rows.Scan(&r.Source, &r.SourceID, &r.Content, &rank); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		// FTS5 rank is negative (more negative = better match). Normalize to positive.
		r.Score = -rank
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}

// extractUserID pulls the userID from a session ID (format: "platform:userID:uuid").
func extractUserID(sessionID string) string {
	parts := strings.SplitN(sessionID, ":", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// SaveConversation persists a conversation's messages.
func (p *SQLiteProvider) SaveConversation(ctx context.Context, sessionID string, messages []llm.Message) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete existing messages for this session (full replacement).
	if _, err := tx.ExecContext(ctx, `DELETE FROM conversations WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete old messages: %w", err)
	}

	// Delete old FTS entries for this session's conversations.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM memory_fts WHERE source = 'conversation' AND source_id LIKE ?`,
		sessionID+":%",
	); err != nil {
		return fmt.Errorf("delete old fts: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO conversations (session_id, role, content, name, tool_calls, tool_call_id, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for i, msg := range messages {
		var toolCallsJSON *string
		if len(msg.ToolCalls) > 0 {
			data, err := json.Marshal(msg.ToolCalls)
			if err != nil {
				return fmt.Errorf("marshal tool calls: %w", err)
			}
			s := string(data)
			toolCallsJSON = &s
		}

		ts := msg.Timestamp
		if ts.IsZero() {
			ts = time.Now().UTC()
		}

		if _, err := stmt.ExecContext(ctx, sessionID, string(msg.Role), msg.Content,
			nilIfEmpty(msg.Name), toolCallsJSON, nilIfEmpty(msg.ToolCallID), ts); err != nil {
			return fmt.Errorf("insert message %d: %w", i, err)
		}

		// Index user and assistant messages in FTS for searchability.
		if msg.Role == llm.RoleUser || msg.Role == llm.RoleAssistant {
			sourceID := fmt.Sprintf("%s:%d", sessionID, i)
			uid := extractUserID(sessionID)
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO memory_fts (source, source_id, user_id, content)
				 VALUES ('conversation', ?, ?, ?)`,
				sourceID, uid, msg.Content,
			); err != nil {
				return fmt.Errorf("insert fts %d: %w", i, err)
			}
		}
	}

	return tx.Commit()
}

// GetConversation retrieves all messages for a session.
func (p *SQLiteProvider) GetConversation(ctx context.Context, sessionID string) ([]llm.Message, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT role, content, name, tool_calls, tool_call_id, timestamp
		 FROM conversations WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query conversation: %w", err)
	}
	defer rows.Close()

	var messages []llm.Message
	for rows.Next() {
		var msg llm.Message
		var role string
		var name, toolCallsJSON, toolCallID sql.NullString
		var ts time.Time

		if err := rows.Scan(&role, &msg.Content, &name, &toolCallsJSON, &toolCallID, &ts); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		msg.Role = llm.Role(role)
		msg.Timestamp = ts
		if name.Valid {
			msg.Name = name.String
		}
		if toolCallID.Valid {
			msg.ToolCallID = toolCallID.String
		}
		if toolCallsJSON.Valid && toolCallsJSON.String != "" {
			if err := json.Unmarshal([]byte(toolCallsJSON.String), &msg.ToolCalls); err != nil {
				return nil, fmt.Errorf("unmarshal tool calls: %w", err)
			}
		}

		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return messages, nil
}

// SessionSearch finds sessions matching a query by searching conversation content.
func (p *SQLiteProvider) SessionSearch(ctx context.Context, query string) ([]SessionResult, error) {
	safeQuery := sanitizeFTSQuery(query)
	if safeQuery == "" {
		return nil, nil
	}

	rows, err := p.db.QueryContext(ctx,
		`SELECT DISTINCT
			substr(source_id, 1, instr(source_id, ':') - 1) as session_id,
			content,
			rank
		 FROM memory_fts
		 WHERE memory_fts MATCH ? AND source = 'conversation'
		 ORDER BY rank
		 LIMIT 20`,
		safeQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("session search: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var results []SessionResult
	for rows.Next() {
		var r SessionResult
		var rank float64
		if err := rows.Scan(&r.SessionID, &r.Preview, &rank); err != nil {
			return nil, fmt.Errorf("scan session result: %w", err)
		}
		if seen[r.SessionID] {
			continue
		}
		seen[r.SessionID] = true
		r.Score = -rank
		r.CreatedAt = time.Now() // Approximate; could be improved with a join.
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}

// Close closes the database connection.
func (p *SQLiteProvider) Close() error {
	return p.db.Close()
}

// nilIfEmpty returns a *string that is nil if s is empty.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// parseSQLiteTime parses the TEXT timestamp SQLite emits from
// aggregates like MIN(timestamp) / MAX(timestamp). Tries the two
// formats the driver and CURRENT_TIMESTAMP use; returns zero Time
// on failure so callers get a safe zero value rather than an error.
func parseSQLiteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// SaveNote inserts or replaces a typed memory note. Upserts on
// (user_id, name). Also indexes description + content into memory_fts
// under source "note:<type>" so Search picks it up.
func (p *SQLiteProvider) SaveNote(ctx context.Context, note Note) error {
	if note.UserID == "" || note.Name == "" {
		return fmt.Errorf("note requires user_id and name")
	}
	if !ValidNoteType(note.Type) {
		return fmt.Errorf("invalid note type: %q", note.Type)
	}

	now := time.Now().UTC()
	if note.ID == "" {
		note.ID = uuid.New().String()
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO notes (id, user_id, type, name, description, content, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, name) DO UPDATE SET
		   type = excluded.type,
		   description = excluded.description,
		   content = excluded.content,
		   updated_at = excluded.updated_at`,
		note.ID, note.UserID, string(note.Type), note.Name,
		note.Description, note.Content, now, now,
	); err != nil {
		return fmt.Errorf("upsert note: %w", err)
	}

	// Refresh FTS entry: delete any prior entry for this note, then insert.
	source := "note:" + string(note.Type)
	sourceID := note.Name
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM memory_fts WHERE source LIKE 'note:%' AND source_id = ? AND user_id = ?`,
		sourceID, note.UserID,
	); err != nil {
		return fmt.Errorf("delete stale fts: %w", err)
	}
	ftsContent := note.Description
	if note.Content != "" {
		if ftsContent != "" {
			ftsContent += "\n"
		}
		ftsContent += note.Content
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO memory_fts (source, source_id, user_id, content)
		 VALUES (?, ?, ?, ?)`,
		source, sourceID, note.UserID, ftsContent,
	); err != nil {
		return fmt.Errorf("insert fts: %w", err)
	}

	return tx.Commit()
}

// UpdateNote updates an existing note's description and/or content.
// Passing empty strings leaves the corresponding field unchanged.
func (p *SQLiteProvider) UpdateNote(ctx context.Context, userID, name, description, content string) error {
	existing, err := p.GetNote(ctx, userID, name)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("note %q not found for user %q", name, userID)
	}
	if description != "" {
		existing.Description = description
	}
	if content != "" {
		existing.Content = content
	}
	return p.SaveNote(ctx, *existing)
}

// DeleteNote removes a note and its FTS index entry.
func (p *SQLiteProvider) DeleteNote(ctx context.Context, userID, name string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`DELETE FROM notes WHERE user_id = ? AND name = ?`, userID, name)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("note %q not found for user %q", name, userID)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM memory_fts WHERE source LIKE 'note:%' AND source_id = ? AND user_id = ?`,
		name, userID,
	); err != nil {
		return fmt.Errorf("delete fts: %w", err)
	}
	return tx.Commit()
}

// GetNote returns a single note by (user_id, name). Returns (nil, nil)
// when the note does not exist (not an error).
func (p *SQLiteProvider) GetNote(ctx context.Context, userID, name string) (*Note, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, user_id, type, name, description, content, created_at, updated_at
		 FROM notes WHERE user_id = ? AND name = ?`, userID, name)

	var n Note
	var typ string
	err := row.Scan(&n.ID, &n.UserID, &typ, &n.Name, &n.Description, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan note: %w", err)
	}
	n.Type = NoteType(typ)
	return &n, nil
}

// ListNotes returns all notes for a user. If typ is non-empty, filters
// to that type. Results are ordered by updated_at DESC.
func (p *SQLiteProvider) ListNotes(ctx context.Context, userID string, typ NoteType) ([]Note, error) {
	query := `SELECT id, user_id, type, name, description, content, created_at, updated_at
	          FROM notes WHERE user_id = ?`
	args := []any{userID}
	if typ != "" {
		if !ValidNoteType(typ) {
			return nil, fmt.Errorf("invalid note type: %q", typ)
		}
		query += ` AND type = ?`
		args = append(args, string(typ))
	}
	query += ` ORDER BY updated_at DESC`

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		var t string
		if err := rows.Scan(&n.ID, &n.UserID, &t, &n.Name, &n.Description, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		n.Type = NoteType(t)
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iter: %w", err)
	}
	return notes, nil
}

// ListUserSessions returns metadata for the user's past sessions,
// newest-first. Scopes via the "platform:userID:uuid" session ID
// prefix. Preview is the first user message of the session truncated
// to 120 chars so listings are skimmable without loading full bodies.
func (p *SQLiteProvider) ListUserSessions(ctx context.Context, userID string, limit int) ([]SessionSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID required")
	}
	if limit <= 0 {
		limit = 10
	}

	// The session ID format is "platform:userID:uuid". We match
	// ":userID:" as a substring to avoid accidental prefix collisions
	// (user IDs are not guaranteed to differ in leading bytes).
	likeAny := "%:" + userID + ":%"

	// Aggregate stats per session in a single pass. Preview is the
	// earliest user-role message; SQLite's MIN on timestamp returns
	// the row's text in correlated subquery.
	rows, err := p.db.QueryContext(ctx, `
		SELECT
			session_id,
			MIN(timestamp) AS first_ts,
			MAX(timestamp) AS last_ts,
			COUNT(*)       AS msg_count,
			(SELECT content FROM conversations c2
			   WHERE c2.session_id = conversations.session_id
			     AND c2.role = 'user'
			   ORDER BY c2.timestamp ASC LIMIT 1) AS preview
		FROM conversations
		WHERE session_id LIKE ?
		GROUP BY session_id
		ORDER BY last_ts DESC
		LIMIT ?`,
		likeAny, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var out []SessionSummary
	for rows.Next() {
		var s SessionSummary
		var preview sql.NullString
		// MIN/MAX over a DATETIME column returns TEXT in SQLite, so
		// scan as strings and parse with the well-known SQLite layout.
		var firstTS, lastTS string
		if err := rows.Scan(&s.SessionID, &firstTS, &lastTS, &s.MessageCount, &preview); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.FirstMessage = parseSQLiteTime(firstTS)
		s.LastMessage = parseSQLiteTime(lastTS)
		if preview.Valid {
			s.Preview = preview.String
			if len(s.Preview) > 120 {
				s.Preview = s.Preview[:117] + "..."
			}
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iter: %w", err)
	}
	return out, nil
}
