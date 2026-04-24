package holographic

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Fact represents a stored fact in the holographic memory.
type Fact struct {
	ID        int     `json:"id"`
	Content   string  `json:"content"`
	Category  string  `json:"category"`
	Tags      string  `json:"tags,omitempty"`
	Trust     float64 `json:"trust"`
	CreatedAt string  `json:"created_at"`
}

// Store manages the SQLite-backed fact storage with FTS5 search.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// NewStore opens or creates a holographic memory database.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite optimizations
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma: %w", err)
		}
	}

	// Create tables
	schema := `
		CREATE TABLE IF NOT EXISTS facts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'general',
			tags TEXT DEFAULT '',
			trust REAL NOT NULL DEFAULT 0.5,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS fact_entities (
			fact_id INTEGER NOT NULL REFERENCES facts(id) ON DELETE CASCADE,
			entity_id INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
			PRIMARY KEY (fact_id, entity_id)
		);
		CREATE VIRTUAL TABLE IF NOT EXISTS facts_fts USING fts5(
			content, category, tags,
			content='facts',
			content_rowid='id',
			tokenize='porter unicode61'
		);
		CREATE TRIGGER IF NOT EXISTS facts_ai AFTER INSERT ON facts BEGIN
			INSERT INTO facts_fts(rowid, content, category, tags)
			VALUES (new.id, new.content, new.category, new.tags);
		END;
		CREATE TRIGGER IF NOT EXISTS facts_ad AFTER DELETE ON facts BEGIN
			INSERT INTO facts_fts(facts_fts, rowid, content, category, tags)
			VALUES ('delete', old.id, old.content, old.category, old.tags);
		END;
		CREATE TRIGGER IF NOT EXISTS facts_au AFTER UPDATE ON facts BEGIN
			INSERT INTO facts_fts(facts_fts, rowid, content, category, tags)
			VALUES ('delete', old.id, old.content, old.category, old.tags);
			INSERT INTO facts_fts(rowid, content, category, tags)
			VALUES (new.id, new.content, new.category, new.tags);
		END;
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &Store{db: db}, nil
}

// AddFact stores a new fact and returns its ID.
func (s *Store) AddFact(content, category, tags string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"INSERT INTO facts (content, category, tags, trust, created_at, updated_at) VALUES (?, ?, ?, 0.5, ?, ?)",
		content, category, tags, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting fact: %w", err)
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

// SearchFTS performs a full-text search on facts.
func (s *Store) SearchFTS(query string, limit int) ([]Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Escape FTS5 special characters
	safeQuery := strings.ReplaceAll(query, `"`, `""`)

	rows, err := s.db.Query(`
		SELECT f.id, f.content, f.category, f.tags, f.trust, f.created_at
		FROM facts_fts fts
		JOIN facts f ON f.id = fts.rowid
		WHERE facts_fts MATCH ?
		ORDER BY -rank
		LIMIT ?
	`, `"`+safeQuery+`"`, limit)
	if err != nil {
		// Fall back to LIKE search if FTS fails
		return s.searchLike(query, limit)
	}
	defer rows.Close()

	return scanFacts(rows)
}

func (s *Store) searchLike(query string, limit int) ([]Fact, error) {
	pattern := "%" + query + "%"
	rows, err := s.db.Query(`
		SELECT id, content, category, tags, trust, created_at
		FROM facts
		WHERE content LIKE ? OR tags LIKE ?
		ORDER BY trust DESC
		LIMIT ?
	`, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// ProbeEntity returns all facts associated with an entity.
func (s *Store) ProbeEntity(entity string) ([]Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT f.id, f.content, f.category, f.tags, f.trust, f.created_at
		FROM facts f
		JOIN fact_entities fe ON f.id = fe.fact_id
		JOIN entities e ON e.id = fe.entity_id
		WHERE LOWER(e.name) = LOWER(?)
		ORDER BY f.trust DESC
	`, entity)
	if err != nil {
		return nil, fmt.Errorf("probe: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// ListFacts returns the most recent facts.
func (s *Store) ListFacts(limit int) ([]Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, content, category, tags, trust, created_at
		FROM facts
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// RemoveFact deletes a fact by ID.
func (s *Store) RemoveFact(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM facts WHERE id = ?", id)
	return err
}

// AdjustTrust modifies a fact's trust score.
func (s *Store) AdjustTrust(id int, delta float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		"UPDATE facts SET trust = MAX(0.0, MIN(1.0, trust + ?)), updated_at = datetime('now') WHERE id = ?",
		delta, id,
	)
	return err
}

// Close closes the database connection.
func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func scanFacts(rows *sql.Rows) ([]Fact, error) {
	var facts []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Content, &f.Category, &f.Tags, &f.Trust, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning fact: %w", err)
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}
